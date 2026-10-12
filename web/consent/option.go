package consent

import (
	"log/slog"

	"github.com/daarxwalker/enjin/cookie"
)

type Option func(m *Manager)

func WithLogger(log *slog.Logger) Option {
	return func(m *Manager) {
		if log != nil {
			m.log = log
		}
	}
}

func WithConfig(options ...cookie.Option) Option {
	return func(m *Manager) {
		m.cookie.Apply(options...)
	}
}
