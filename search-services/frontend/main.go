package main

import (
	"context"
	"errors"
	"flag"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"os/signal"

	"yadro.com/course/frontend/config"
	"yadro.com/course/frontend/handlers"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "frontend configuration file")
	flag.Parse()

	cfg := config.MustLoad(configPath)

	log := mustMakeLogger(cfg.LogLevel)
	log.Info("starting frontend server")

	tmpl := template.Must(template.ParseGlob("/templates/*.html"))

	h := handlers.New(log, cfg.APIAddress, tmpl)

	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("/static"))))
	mux.HandleFunc("GET /", h.SearchPage)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	server := http.Server{
		Addr:              cfg.Address,
		ReadHeaderTimeout: cfg.Timeout,
		Handler:           mux,
	}

	go func() {
		<-ctx.Done()
		log.Debug("shutting down frontend server")
		if err := server.Shutdown(context.Background()); err != nil {
			log.Error("erroneous shutdown", "error", err)
		}
	}()

	log.Info("Running HTTP server", "address", cfg.Address)
	if err := server.ListenAndServe(); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			log.Error("server closed unexpectedly", "error", err)
		}
	}
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
