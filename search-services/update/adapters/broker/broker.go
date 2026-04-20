package broker

import (
	"log/slog"

	"github.com/nats-io/nats.go"
)

const (
	TopicDBUpdated = "xkcd.db.updated"
	TopicDBDropped = "xkcd.db.dropped"
)

type Publisher struct {
	log *slog.Logger
	nc  *nats.Conn
}

func New(address string, log *slog.Logger) (*Publisher, error) {
	nc, err := nats.Connect(address)
	if err != nil {
		return nil, err
	}
	log.Info("connected to broker", "address", address)
	return &Publisher{log: log, nc: nc}, nil
}

func (p *Publisher) PublishUpdated() {
	if err := p.nc.Publish(TopicDBUpdated, []byte("updated")); err != nil {
		p.log.Error("failed to publish updated event", "error", err)
		return
	}
	if err := p.nc.Flush(); err != nil {
		p.log.Error("failed to flush updated event", "error", err)
	}
	p.log.Info("published db updated event")
}

func (p *Publisher) PublishDropped() {
	if err := p.nc.Publish(TopicDBDropped, []byte("dropped")); err != nil {
		p.log.Error("failed to publish dropped event", "error", err)
		return
	}
	if err := p.nc.Flush(); err != nil {
		p.log.Error("failed to flush dropped event", "error", err)
	}
	p.log.Info("published db dropped event")
}

func (p *Publisher) Close() {
	p.nc.Close()
}
