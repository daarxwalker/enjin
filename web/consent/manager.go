package consent

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/daarxwalker/enjin/cookie"
)

type Manager struct {
	cookie *cookie.Config
	log    *slog.Logger
}

const (
	DefaultCookieName = "X-Consent"
	DefaultTTL        = 365 * 24 * time.Hour
)

func NewManager(options ...Option) *Manager {
	m := &Manager{
		cookie: cookie.NewConfig(
			cookie.WithName(DefaultCookieName),
			cookie.WithTTL(DefaultTTL),
		),
		log: slog.Default(),
	}
	for _, option := range options {
		option(m)
	}
	return m
}

func (m *Manager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			ctxVal := &ContextValue{manager: m}
			ctx := context.WithValue(r.Context(), key, ctxVal)
			r = r.WithContext(ctx)
			consentCookie, err := r.Cookie(m.cookie.Name)
			if err != nil && !errors.Is(err, http.ErrNoCookie) {
				m.log.ErrorContext(ctx, fmt.Sprintf("get cookie failed: %s", err))
				next.ServeHTTP(w, r)
				return
			}
			if consentCookie == nil || consentCookie.Value == "" {
				next.ServeHTTP(w, r)
				return
			}
			v, parseErr := url.ParseQuery(consentCookie.Value)
			if parseErr != nil {
				m.log.ErrorContext(ctx, fmt.Sprintf("parse consent cookie failed: %s", parseErr))
				next.ServeHTTP(w, r)
				return
			}
			ctxVal.Cookies = new(Cookies)
			ctxVal.Cookies.Analytic = v.Get(Analytic) == "true"
			ctxVal.Cookies.Preference = v.Get(Preference) == "true"
			ctxVal.Cookies.Marketing = v.Get(Marketing) == "true"
			ctxVal.Exists = true
			Attach(w, r, *ctxVal.Cookies)
			next.ServeHTTP(w, r)
		},
	)
}
