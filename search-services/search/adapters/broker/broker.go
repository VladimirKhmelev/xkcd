package broker

import (
	"context"
	"log/slog"

	"github.com/nats-io/nats.go"
	"yadro.com/course/search/core"
)

const (
	topicDBUpdated = "xkcd.db.updated"
	topicDBDropped = "xkcd.db.dropped"
)

type Subscriber struct {
	log     *slog.Logger
	nc      *nats.Conn
	builder core.IndexBuilder
	resetter core.IndexResetter
}

func New(address string, log *slog.Logger, builder core.IndexBuilder, resetter core.IndexResetter) (*Subscriber, error) {
	nc, err := nats.Connect(address)
	if err != nil {
		return nil, err
	}
	log.Info("connected to broker", "address", address)
	return &Subscriber{log: log, nc: nc, builder: builder, resetter: resetter}, nil
}

func (s *Subscriber) Run(ctx context.Context) {
	updatedCh := make(chan *nats.Msg, 10)
	droppedCh := make(chan *nats.Msg, 10)

	subUpdated, err := s.nc.ChanSubscribe(topicDBUpdated, updatedCh)
	if err != nil {
		s.log.Error("failed to subscribe to updated topic", "error", err)
		return
	}
	subDropped, err := s.nc.ChanSubscribe(topicDBDropped, droppedCh)
	if err != nil {
		s.log.Error("failed to subscribe to dropped topic", "error", err)
		return
	}

	defer func() {
		_ = subUpdated.Unsubscribe()
		_ = subDropped.Unsubscribe()
		s.nc.Close()
	}()

	s.log.Info("broker subscriber running")

	for {
		select {
		case <-ctx.Done():
			return
		case <-updatedCh:
			s.log.Info("received db updated event, rebuilding index")
			if err := s.builder.BuildIndex(ctx); err != nil {
				s.log.Error("failed to rebuild index after update", "error", err)
			}
		case <-droppedCh:
			s.log.Info("received db dropped event, resetting index")
			s.resetter.ResetIndex()
		}
	}
}
