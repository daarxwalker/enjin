package csrf

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/daarxwalker/enjin/client"
	"github.com/daarxwalker/enjin/token"
	"github.com/daarxwalker/enjin/web/htmx"

	"github.com/a-h/templ"
)

type InputManager struct {
	driver Driver
	reject http.Handler
	log    *slog.Logger
	ttl    time.Duration
	limit  int
}

const (
	Field = "__csrf__"
)

func NewInputManager(driver Driver, options ...InputOption) *InputManager {
	m := &InputManager{
		driver: driver,
		reject: forbidden,
		log:    slog.Default(),
		ttl:    24 * time.Hour,
		limit:  20,
	}
	for _, option := range options {
		option(m)
	}
	return m
}

func (m *InputManager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), ctxKey, &ctxVal{input: m})
			next.ServeHTTP(w, r.WithContext(ctx))
		},
	)
}

func (m *InputManager) Verify(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			cv := from(ctx)
			if cv == nil {
				cv = &ctxVal{input: m}
				r = r.WithContext(context.WithValue(ctx, ctxKey, cv))
			}
			cv.rejected = !m.consume(ctx, client.IDFrom(ctx), submittedToken(r))
			next.ServeHTTP(w, r)
		},
	)
}

func (m *InputManager) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			if !m.consume(ctx, client.IDFrom(ctx), submittedToken(r)) {
				m.reject.ServeHTTP(w, r)
				return
			}
			next.ServeHTTP(w, r)
		},
	)
}

func (m *InputManager) Redirect(w http.ResponseWriter, r *http.Request, to string) {
	if strings.Contains(r.Header.Get("Accept"), "text/html") {
		if !htmx.Is(r) {
			http.Redirect(w, r, to, http.StatusSeeOther)
		} else {
			htmx.AttachRedirect(w, to)
		}
	} else {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
	}
}

func (m *InputManager) Create(ctx context.Context) string {
	clientID := client.IDFrom(ctx)
	if len(clientID) == 0 {
		return ""
	}
	created := token.New()
	expiresAt := time.Now().Add(m.ttl)
	if addErr := m.driver.Add(ctx, createCacheKey(clientID), created, expiresAt, m.limit); addErr != nil {
		m.log.ErrorContext(ctx, fmt.Sprintf("csrf: add token to cache failed: %s", addErr))
		return ""
	}
	return created
}

func (m *InputManager) CreateInput(ctx context.Context) templ.Component {
	return Input(Field, m.Create(ctx)) //nolint:contextcheck // templ passes ctx to Render
}

func (m *InputManager) consume(ctx context.Context, clientID, submitted string) bool {
	if len(clientID) == 0 || !token.Valid(submitted) {
		return false
	}
	taken, err := m.driver.Take(ctx, createCacheKey(clientID), submitted, time.Now())
	if err != nil {
		m.log.ErrorContext(ctx, fmt.Sprintf("csrf: take token from cache failed: %s", err))
		return false
	}
	return taken
}
