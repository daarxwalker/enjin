package theme

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/daarxwalker/enjin/cookie"
	"github.com/daarxwalker/enjin/httpx"
)

type Manager struct {
	cookie       *cookie.Config
	defaultTheme Value
}

const (
	CookieName = "X-Theme"
)

const (
	queryKey = "theme"
)

func NewManager(options ...Option) *Manager {
	m := &Manager{
		cookie: cookie.NewConfig(
			cookie.WithName(CookieName),
			cookie.WithTTL(365*24*time.Hour),
		),
		defaultTheme: Light,
	}
	for _, opt := range options {
		opt(m)
	}
	return m
}

func (m *Manager) resolve(r *http.Request, queryValue string) Value {
	raw := queryValue
	if raw == "" {
		if themeCookie, err := r.Cookie(m.cookie.Name); err == nil {
			raw = themeCookie.Value
		}
	}
	if value, ok := parseValue(raw); ok {
		return value
	}
	return m.defaultTheme
}

func (m *Manager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			queryValue := r.URL.Query().Get(queryKey)
			value := m.resolve(r, queryValue)
			m.cookie.Attach(w, value.String())
			if queryValue != "" {
				q := r.URL.Query()
				q.Del(queryKey)
				target := url.URL{Path: r.URL.Path, RawPath: r.URL.RawPath, RawQuery: q.Encode()}
				location := target.String()
				if !httpx.IsLocalPath(location) {
					location = "/"
				}
				location = strings.TrimSuffix(httpx.PathPrefix(r), "/") + location
				if !httpx.IsLocalPath(location) {
					location = "/"
				}
				http.Redirect(w, r, location, http.StatusSeeOther) //nolint:gosec // location is checked by httpx.IsLocalPath
				return
			}
			r = r.WithContext(
				context.WithValue(
					r.Context(), key, &ContextValue{
						manager: m,
						value:   value,
					},
				),
			)
			next.ServeHTTP(w, r)
		},
	)
}
