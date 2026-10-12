package antibot

import (
	"log/slog"
	"net/http"
)

type Option func(v *Verifier)

func WithLogger(log *slog.Logger) Option {
	return func(v *Verifier) {
		if log != nil {
			v.log = log
		}
	}
}

func WithRejectHandler(reject http.Handler) Option {
	return func(v *Verifier) {
		if reject != nil {
			v.reject = reject
		}
	}
}
