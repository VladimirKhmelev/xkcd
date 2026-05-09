package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	searchpb "yadro.com/course/proto/search"
	"yadro.com/course/pkg/tracing"
	"yadro.com/course/search/adapters/broker"
	"yadro.com/course/search/adapters/cache"
	"yadro.com/course/search/adapters/db"
	searchgrpc "yadro.com/course/search/adapters/grpc"
	"yadro.com/course/search/adapters/initiator"
	"yadro.com/course/search/adapters/words"
	"yadro.com/course/search/config"
	"yadro.com/course/search/core"
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
	log.Info("starting search server")

	if cfg.OTLPEndpoint != "" {
		shutdown, err := tracing.Init(context.Background(), "search", cfg.OTLPEndpoint)
		if err != nil {
			log.Warn("tracing unavailable", "error", err)
		} else {
			defer shutdown(context.Background())
			log.Info("tracing enabled", "endpoint", cfg.OTLPEndpoint)
		}
	}

	storage, err := db.New(log, cfg.DBAddress)
	if err != nil {
		return fmt.Errorf("failed to connect to db: %w", err)
	}

	wordsClient, err := words.NewClient(cfg.WordsAddress, log)
	if err != nil {
		return fmt.Errorf("failed to create words client: %w", err)
	}

	var redisCache core.Cache
	rc, err := cache.New(cfg.RedisAddress)
	if err != nil {
		log.Warn("redis unavailable, running without cache", "error", err)
		redisCache = cache.Noop{}
	} else {
		redisCache = rc
	}

	service := core.NewService(log, storage, wordsClient, redisCache)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	sub, err := broker.New(cfg.BrokerAddress, log, service, service, redisCache)
	if err != nil {
		return fmt.Errorf("failed to connect to broker: %w", err)
	}
	go sub.Run(ctx)

	idx := initiator.New(log, service, cfg.IndexTTL)
	go idx.Run(ctx)

	listener, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s := grpc.NewServer(grpc.StatsHandler(otelgrpc.NewServerHandler()))
	searchpb.RegisterSearchServer(s, searchgrpc.NewServer(service))
	reflection.Register(s)

	go func() {
		<-ctx.Done()
		log.Debug("shutting down server")
		s.GracefulStop()
	}()

	log.Info("running gRPC server", "address", cfg.Address)
	if err := s.Serve(listener); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
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
		level = slog.LevelInfo
	}
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	return slog.New(handler)
}
