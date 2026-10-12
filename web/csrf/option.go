package csrf

import (
	"log/slog"
	"net/http"
	"time"
)

type InputOption func(manager *InputManager)

func WithLogger(log *slog.Logger) InputOption {
	return func(c *InputManager) {
		if log != nil {
			c.log = log
		}
	}
}

func WithRejectHandler(reject http.Handler) InputOption {
	return func(c *InputManager) {
		if reject != nil {
			c.reject = reject
		}
	}
}

func WithTTL(ttl time.Duration) InputOption {
	return func(c *InputManager) {
		if ttl > 0 {
			c.ttl = ttl
		}
	}
}

func WithLimit(limit int) InputOption {
	return func(c *InputManager) {
		if limit > 0 {
			c.limit = limit
		}
	}
}
