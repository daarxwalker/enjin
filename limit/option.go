package limit

import (
	"log/slog"
	"time"
)

type Option func(*Manager)

func WithLogger(log *slog.Logger) Option {
	return func(m *Manager) {
		if log != nil {
			m.log = log
		}
	}
}

func WithTTL(ttl time.Duration) Option {
	return func(m *Manager) {
		m.ttl = ttl
	}
}

// WithSessions counts requests with a session too; by default Load and Verify skip them.
func WithSessions() Option {
	return func(m *Manager) {
		m.sessions = true
	}
}
