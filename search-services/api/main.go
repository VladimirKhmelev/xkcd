// @title           XKCD Search API
// @version         1.0
// @description     API for searching XKCD comics
// @host            localhost:28080
// @BasePath        /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"

	_ "yadro.com/course/api/docs"

	"yadro.com/course/api/adapters/aaa"
	apidb "yadro.com/course/api/adapters/db"
	"yadro.com/course/api/adapters/rest"
	"yadro.com/course/api/adapters/rest/middleware"
	searchadapter "yadro.com/course/api/adapters/search"
	"yadro.com/course/api/adapters/update"
	"yadro.com/course/api/adapters/words"
	"yadro.com/course/api/config"
	"yadro.com/course/api/core"
	"yadro.com/course/pkg/tracing"

	httpSwagger "github.com/swaggo/http-swagger"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "server configuration file")
	flag.Parse()

	cfg := config.MustLoad(configPath)
	log := mustMakeLogger(cfg.LogLevel)

	if err := run(cfg, log); err != nil {
		log.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func run(cfg config.Config, log *slog.Logger) error {
	log.Info("starting server")
	log.Debug("debug messages are enabled")

	if cfg.OTLPEndpoint != "" {
		shutdown, err := tracing.Init(context.Background(), "api", cfg.OTLPEndpoint)
		if err != nil {
			log.Warn("tracing unavailable", "error", err)
		} else {
			defer func() {
				if err := shutdown(context.Background()); err != nil {
					log.Error("tracing shutdown failed", "error", err)
				}
			}()
			log.Info("tracing enabled", "endpoint", cfg.OTLPEndpoint)
		}
	}

	updateClient, err := update.NewClient(cfg.UpdateAddress, log)
	if err != nil {
		return fmt.Errorf("cannot init update adapter: %w", err)
	}

	wordsClient, err := words.NewClient(cfg.WordsAddress, log)
	if err != nil {
		return fmt.Errorf("cannot init words adapter: %w", err)
	}

	searchClient, err := searchadapter.NewClient(cfg.SearchAddress, log)
	if err != nil {
		return fmt.Errorf("cannot init search adapter: %w", err)
	}

	auth, err := aaa.New(cfg.TokenTTL, log)
	if err != nil {
		return fmt.Errorf("cannot init aaa: %w", err)
	}

	userDB, err := apidb.New(log, cfg.DBAddress)
	if err != nil {
		return fmt.Errorf("cannot init user db: %w", err)
	}
	if err := userDB.Migrate(); err != nil {
		return fmt.Errorf("cannot migrate user db: %w", err)
	}

	mux := http.NewServeMux()

	mux.Handle("GET /swagger/", httpSwagger.WrapHandler)
	mux.Handle("GET /metrics", rest.NewMetricsHandler())
	mux.Handle("GET /api/ping", rest.NewPingHandler(log, map[string]core.Pinger{
		"update": updateClient,
		"words":  wordsClient,
		"search": searchClient,
	}))
	mux.Handle("POST /api/login", rest.NewLoginHandler(log, auth))
	mux.Handle("POST /api/register", rest.NewRegisterHandler(log, userDB))
	mux.Handle("POST /api/user/login", rest.NewUserLoginHandler(log, auth, userDB))
	mux.Handle("POST /api/db/update", middleware.Auth(rest.NewUpdateHandler(log, updateClient), auth))
	mux.Handle("GET /api/db/stats", rest.NewUpdateStatsHandler(log, updateClient))
	mux.Handle("GET /api/db/status", rest.NewUpdateStatusHandler(log, updateClient))
	mux.Handle("DELETE /api/db", middleware.Auth(rest.NewDropHandler(log, updateClient), auth))
	mux.Handle("GET /api/search", middleware.Concurrency(rest.NewSearchHandler(log, searchClient), cfg.SearchConcurrency))
	mux.Handle("GET /api/isearch", middleware.Rate(rest.NewSearchIndexHandler(log, searchClient), cfg.SearchRate))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	server := http.Server{
		Addr:              cfg.HTTPConfig.Address,
		ReadHeaderTimeout: cfg.HTTPConfig.Timeout,
		Handler:           otelhttp.NewHandler(middleware.WithMetrics(mux), "api"),
		BaseContext:       func(_ net.Listener) context.Context { return ctx },
	}

	go func() {
		<-ctx.Done()
		log.Debug("shutting down server")
		if err := server.Shutdown(context.Background()); err != nil {
			log.Error("erroneous shutdown", "error", err)
		}
	}()

	log.Info("Running HTTP server", "address", cfg.HTTPConfig.Address)
	if err := server.ListenAndServe(); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server closed unexpectedly: %w", err)
		}
	}
	return nil
}

func mustMakeLogger(logLevel string) *slog.Logger {
	var level slog.Level
	switch logLevel {
	case "DEBUG":
		level = slog.LevelDebug
	case "INFO":
		level = slog.LevelInfo
	case "ERROR":
		level = slog.LevelError
	default:
		panic("unknown log level: " + logLevel)
	}
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level, AddSource: true})
	return slog.New(handler)
}
