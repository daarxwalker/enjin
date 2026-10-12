package redirect

import (
	"context"
	"net/http"
	"time"

	"github.com/daarxwalker/enjin/cookie"
)

type Manager struct {
	cookie *cookie.Config
}

func NewManager(options ...Option) *Manager {
	m := &Manager{
		cookie: cookie.NewConfig(
			cookie.WithName("X-Redirect"),
			cookie.WithHTTPOnly(false),
			cookie.WithTTL(10*time.Minute),
		),
	}
	for _, opt := range options {
		opt(m)
	}
	return m
}

func (m *Manager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), key, &ctxVal{manager: m})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
