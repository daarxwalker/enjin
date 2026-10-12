package client

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/daarxwalker/enjin/cookie"
	"github.com/daarxwalker/enjin/infra/cache"
	"github.com/daarxwalker/enjin/token"
)

const (
	DefaultCookie = "X-Client"
)

var DefaultTTL = 365 * 24 * time.Hour

type Manager struct {
	cache  cache.Commander
	cookie *cookie.Config
	log    *slog.Logger
}

func NewManager(cache cache.Commander, options ...Option) *Manager {
	m := new(Manager{
		cookie: cookie.NewConfig(
			cookie.WithTTL(DefaultTTL),
			cookie.WithName(DefaultCookie),
		),
		cache: cache,
		log:   slog.Default(),
	})
	for _, option := range options {
		option(m)
	}
	return m
}

func (m *Manager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			cv := &ctxVal{
				values:    make(map[string]any),
				unixMilli: make(map[string]int64),
			}
			r = r.WithContext(
				context.WithValue(ctx, ctxKey, cv),
			)
			if c, err := r.Cookie(m.cookie.Name); err == nil && token.Valid(c.Value) {
				cv.id = c.Value
			}
			if cv.id == "" {
				cv.id = token.New()
			}
			m.getValues(ctx, cv)
			m.cookie.Attach(w, cv.id)
			next.ServeHTTP(w, r)
			m.saveValues(ctx, cv)
		},
	)
}

func (m *Manager) getValues(ctx context.Context, cv *ctxVal) {
	if cv == nil || cv.id == "" {
		return
	}
	cacheKey := m.createCacheKey(cv.id)
	var doc storedValuesDoc
	if _, err := m.cache.GetJSON(ctx, cacheKey, &doc); err != nil {
		m.log.ErrorContext(ctx, fmt.Sprintf("get client values from cache failed: %s", err))
		return
	}
	now := time.Now()
	for key, unixMilli := range doc.UnixMilli {
		if time.UnixMilli(unixMilli).Before(now) {
			delete(doc.Values, key)
			delete(doc.UnixMilli, key)
			cv.changed = true
		}
	}
	for key, raw := range doc.Values {
		cv.values[key] = raw
	}
	if len(doc.UnixMilli) > 0 {
		cv.unixMilli = doc.UnixMilli
	}
}

func (m *Manager) saveValues(ctx context.Context, cv *ctxVal) {
	if cv == nil || cv.id == "" || !cv.changed {
		return
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
	defer cancel()
	cacheKey := m.createCacheKey(cv.id)
	if len(cv.values) == 0 {
		if err := m.cache.Delete(ctx, cacheKey); err != nil {
			m.log.ErrorContext(ctx, fmt.Sprintf("delete client values from cache failed: %s", err))
		}
		return
	}
	if err := m.cache.SetJSON(ctx, cacheKey, valuesDoc{
		Values:    cv.values,
		UnixMilli: cv.unixMilli,
	}, m.valuesTTL(cv)); err != nil {
		m.log.ErrorContext(ctx, fmt.Sprintf("set client values to cache failed: %s", err))
	}
}

func (m *Manager) valuesTTL(cv *ctxVal) time.Duration {
	var latest int64
	for key := range cv.values {
		unixMilli, ok := cv.unixMilli[key]
		if !ok {
			return cmp.Or(m.cookie.TTL, DefaultTTL)
		}
		latest = max(latest, unixMilli)
	}
	return max(time.Until(time.UnixMilli(latest)), time.Millisecond)
}

func (m *Manager) createCacheKey(token string) string {
	return "client:" + token
}
