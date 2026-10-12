package flash

import "log/slog"

type Option func(m *Manager)

func WithLogger(log *slog.Logger) Option {
	return func(m *Manager) {
		if log != nil {
			m.log = log
		}
	}
}
