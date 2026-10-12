package nats

import (
	"log/slog"
	"time"
)

type Option func(*Driver)

func WithPrefix(prefix string) Option {
	return func(d *Driver) {
		d.prefix = prefix
	}
}

func WithLogger(log *slog.Logger) Option {
	return func(d *Driver) {
		d.log = log
	}
}

func WithJetStream(stream string, maxAge time.Duration) Option {
	return func(d *Driver) {
		d.stream = stream
		d.maxAge = maxAge
	}
}
