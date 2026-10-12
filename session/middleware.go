package session

import (
	"net/http"

	"github.com/daarxwalker/enjin/app"
	"github.com/daarxwalker/enjin/security"
)

type middlewareConfig struct {
	redirect func(r *http.Request) string
}

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
						http.Redirect(w, r, cfg.redirect(r), http.StatusSeeOther)
						return
					}
					http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
					return
				}
				if !policy(session.Role) {
					if cfg.redirect != nil {
						http.Redirect(w, r, cfg.redirect(r), http.StatusSeeOther)
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
						http.Redirect(w, r, cfg.redirect(r), http.StatusSeeOther)
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
