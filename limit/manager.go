package limit

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/daarxwalker/enjin/httpx"
	"github.com/daarxwalker/enjin/session"
)

var ErrNotLoaded = errors.New("limit not loaded")

type Manager struct {
	cache Counter
	log   *slog.Logger
	key   ctxKeyType
	name  string
	ttl   time.Duration
	max   int
}

func NewLimiter(
	cache Counter,
	name string,
	maximum int,
	options ...Option,
) *Manager {
	m := &Manager{
		cache: cache,
		log:   slog.Default(),
		ttl:   24 * time.Hour,
		key:   ctxKeyType{name: name},
		max:   maximum,
		name:  name,
	}
	for _, option := range options {
		option(m)
	}
	return m
}

func (m *Manager) Max() int {
	return m.max
}

type state struct {
	cacheKey string
	count    int
}

func (m *Manager) Current(ctx context.Context) int {
	s, ok := m.stateFrom(ctx)
	if !ok {
		return 0
	}
	return s.count
}

func (m *Manager) ResetIn(ctx context.Context) time.Duration {
	s, ok := m.stateFrom(ctx)
	if !ok {
		return 0
	}
	ttl, err := m.cache.TTL(ctx, s.cacheKey)
	if err != nil {
		m.log.ErrorContext(ctx, fmt.Sprintf("get limit ttl failed: %s", err))
		return 0
	}
	return ttl
}

func (m *Manager) Consume(ctx context.Context) (bool, error) {
	s, ok := m.stateFrom(ctx)
	if !ok {
		return false, ErrNotLoaded
	}
	_, allowed, err := m.consume(ctx, s.cacheKey)
	return allowed, err
}

func (m *Manager) Revert(ctx context.Context) error {
	s, ok := m.stateFrom(ctx)
	if !ok {
		return ErrNotLoaded
	}
	if _, err := m.cache.Increment(ctx, s.cacheKey, -1, m.ttl); err != nil {
		return fmt.Errorf("revert limit failed: %w", err)
	}
	return nil
}

func (m *Manager) Load(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			if session.Exists(ctx) {
				next.ServeHTTP(w, r)
				return
			}
			next.ServeHTTP(w, r.WithContext(m.load(ctx, httpx.ClientIP(r))))
		},
	)
}

func (m *Manager) Verify(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			if session.Exists(ctx) {
				next.ServeHTTP(w, r)
				return
			}
			cacheKey := m.createCacheKey(httpx.ClientIP(r))
			count, allowed, err := m.consume(ctx, cacheKey)
			if err != nil {
				m.log.ErrorContext(ctx, err.Error())
				http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
				return
			}
			if !allowed {
				ttl, ttlErr := m.cache.TTL(ctx, cacheKey)
				if ttlErr != nil {
					m.log.ErrorContext(ctx, fmt.Sprintf("get limit ttl failed: %s", ttlErr))
				}
				if ttl > 0 {
					w.Header().Set("Retry-After", strconv.Itoa(int(ttl.Round(time.Second).Seconds())))
				}
				http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
				return
			}
			ctx = context.WithValue(ctx, m.key, state{cacheKey: cacheKey, count: count})
			next.ServeHTTP(w, r.WithContext(ctx))
		},
	)
}

func (m *Manager) consume(ctx context.Context, cacheKey string) (int, bool, error) {
	count, err := m.cache.Increment(ctx, cacheKey, 1, m.ttl)
	if err != nil {
		return 0, false, fmt.Errorf("consume limit failed: %w", err)
	}
	if count <= m.max {
		return count, true, nil
	}
	count, err = m.cache.Increment(ctx, cacheKey, -1, m.ttl)
	if err != nil {
		return 0, false, fmt.Errorf("revert exceeded limit failed: %w", err)
	}
	return count, false, nil
}

func (m *Manager) load(ctx context.Context, clientIP string) context.Context {
	cacheKey := m.createCacheKey(clientIP)
	limitStr, err := m.cache.GetString(ctx, cacheKey)
	if err != nil {
		m.log.ErrorContext(ctx, fmt.Sprintf("get limit failed: %s", err))
	}
	if limitStr == "" {
		limitStr = "0"
	}
	limit, atoiErr := strconv.Atoi(limitStr)
	if atoiErr != nil {
		m.log.ErrorContext(ctx, fmt.Sprintf("limit atoi %s failed: %s", limitStr, atoiErr))
	}
	return context.WithValue(ctx, m.key, state{cacheKey: cacheKey, count: limit})
}

func (m *Manager) createCacheKey(clientIP string) string {
	return "limit:" + m.name + ":" + clientIP
}

func (m *Manager) stateFrom(ctx context.Context) (state, bool) {
	var zero state
	v := ctx.Value(m.key)
	if v == nil {
		return zero, false
	}
	s, ok := v.(state)
	return s, ok
}
