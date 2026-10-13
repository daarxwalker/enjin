package session

import (
	"net/http"
	"net/url"

	"github.com/daarxwalker/enjin/app"
	"github.com/daarxwalker/enjin/httpx"
	"github.com/daarxwalker/enjin/security"
)

const redirectQueryKey = "redirect"

type MiddlewareOption func(*middlewareConfig)

func Restrict(policy security.Policy, options ...MiddlewareOption) func(http.Handler) http.Handler {
	cfg := new(middlewareConfig)
	for _, opt := range options {
		opt(cfg)
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				session := From(r.Context())
				if session == nil ||
					session.ID == "" ||
					(session.Role != nil && session.Role.Name == "") {
					if cfg.redirect != nil {
						http.Redirect(w, r, cfg.location(r), http.StatusSeeOther) //nolint:gosec // the request URL only goes into the query of the configured target
						return
					}
					http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
					return
				}
				if !policy(session.Role) {
					if cfg.redirect != nil {
						http.Redirect(w, r, cfg.location(r), http.StatusSeeOther) //nolint:gosec // the request URL only goes into the query of the configured target
						return
					}
					http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
					return
				}
				next.ServeHTTP(w, r)
			},
		)
	}
}

func Forbid(options ...MiddlewareOption) func(http.Handler) http.Handler {
	cfg := new(middlewareConfig)
	for _, opt := range options {
		opt(cfg)
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				if Exists(r.Context()) {
					if cfg.redirect != nil {
						http.Redirect(w, r, cfg.location(r), http.StatusSeeOther) //nolint:gosec // the request URL only goes into the query of the configured target
						return
					}
					http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
					return
				}
				next.ServeHTTP(w, r)
			},
		)
	}
}

func WithRedirectName(name string, params ...map[string]any) MiddlewareOption {
	return func(cfg *middlewareConfig) {
		cfg.redirect = func(r *http.Request) string {
			return app.MustURL(r.Context(), name, params...)
		}
	}
}

func WithRedirectBack() MiddlewareOption {
	return func(cfg *middlewareConfig) {
		cfg.back = true
	}
}

type middlewareConfig struct {
	redirect func(r *http.Request) string
	back     bool
}

func (c *middlewareConfig) location(r *http.Request) string {
	location := c.redirect(r)
	if !c.back || (r.Method != http.MethodGet && r.Method != http.MethodHead) {
		return location
	}
	back := r.URL.RequestURI()
	if ctxv := httpx.From(r.Context()); ctxv != nil {
		back = ctxv.URL
	}
	if !httpx.IsLocalPath(back) {
		return location
	}
	target, err := url.Parse(location)
	if err != nil {
		return location
	}
	query := target.Query()
	query.Set(redirectQueryKey, back)
	target.RawQuery = query.Encode()
	return target.String()
}
