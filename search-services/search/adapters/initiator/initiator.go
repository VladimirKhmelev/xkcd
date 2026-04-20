package initiator

import (
	"context"
	"log/slog"
	"time"

	"yadro.com/course/search/core"
)

type Initiator struct {
	log     *slog.Logger
	builder core.IndexBuilder
	ttl     time.Duration
}

func New(log *slog.Logger, builder core.IndexBuilder, ttl time.Duration) *Initiator {
	return &Initiator{log: log, builder: builder, ttl: ttl}
}

func (i *Initiator) Run(ctx context.Context) {
	i.build(ctx)
	ticker := time.NewTicker(i.ttl)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			i.build(ctx)
		}
	}
}

func (i *Initiator) build(ctx context.Context) {
	i.log.Info("building search index")
	if err := i.builder.BuildIndex(ctx); err != nil {
		i.log.Error("failed to build index", "error", err)
	}
}
