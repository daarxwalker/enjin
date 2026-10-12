package flash

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/daarxwalker/enjin/client"
	"github.com/daarxwalker/enjin/infra/cache"
)

type Manager struct {
	cache cache.Commander
	log   *slog.Logger
}

func NewManager(cache cache.Commander, options ...Option) *Manager {
	m := &Manager{
		cache: cache,
		log:   slog.Default(),
	}
	for _, option := range options {
		option(m)
	}
	return m
}

func (m *Manager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			clientID := client.IDFrom(r.Context())
			if clientID == "" {
				next.ServeHTTP(w, r)
				return
			}
			cacheKey := createCacheKey(clientID)
			var storedMessages []Message
			if _, err := m.cache.GetJSON(r.Context(), cacheKey, &storedMessages); err != nil {
				m.log.ErrorContext(r.Context(), fmt.Sprintf("get flash messages from cache failed: %s", err))
			}
			ctxVal := &ContextValue{
				Messages: storedMessages,
			}
			ctx := context.WithValue(r.Context(), key, ctxVal)
			next.ServeHTTP(w, r.WithContext(ctx))
			m.saveMessages(ctx, cacheKey, ctxVal, storedMessages)
		},
	)
}

func (m *Manager) saveMessages(ctx context.Context, cacheKey string, ctxVal *ContextValue, storedMessages []Message) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
	defer cancel()
	switch {
	case ctxVal.read && len(storedMessages) > 0:
		if err := m.cache.Delete(ctx, cacheKey); err != nil {
			m.log.ErrorContext(ctx, fmt.Sprintf("delete flash messages after obtain failed: %s", err))
		}
	case !ctxVal.read && len(ctxVal.Messages) > len(storedMessages):
		if err := m.cache.SetJSON(ctx, cacheKey, ctxVal.Messages, 5*time.Minute); err != nil {
			m.log.ErrorContext(ctx, fmt.Sprintf("set flash messages to cache failed: %s", err))
		}
	}
}
