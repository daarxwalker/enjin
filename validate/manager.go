package validate

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

func (v *Manager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			clientID := client.IDFrom(ctx)
			if clientID == "" {
				next.ServeHTTP(w, r)
				return
			}
			cacheKey := createCacheKey(clientID)
			var storedResult Result
			if _, err := v.cache.GetJSON(ctx, cacheKey, &storedResult); err != nil {
				v.log.ErrorContext(ctx, fmt.Sprintf("get validation messages from cache failed: %s", err))
			}
			cv := &ctxVal{
				manager: v,
				result:  storedResult,
			}
			r = r.WithContext(
				context.WithValue(
					ctx, key, cv,
				),
			)
			next.ServeHTTP(w, r)
		},
	)
}

func (v *Manager) save(ctx context.Context, result Result) {
	clientID := client.IDFrom(ctx)
	cacheKey := createCacheKey(clientID)
	cacheOpCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	if err := v.cache.SetJSON(cacheOpCtx, cacheKey, result, 5*time.Minute); err != nil {
		v.log.ErrorContext(cacheOpCtx, fmt.Sprintf("set validation result to cache failed: %s", err))
	}
}

func (v *Manager) destroy(ctx context.Context) {
	clientID := client.IDFrom(ctx)
	cacheKey := createCacheKey(clientID)
	cacheOpCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	if err := v.cache.Delete(cacheOpCtx, cacheKey); err != nil {
		v.log.ErrorContext(cacheOpCtx, fmt.Sprintf("delete validation result in cache failed: %s", err))
	}
}
