package web

import (
	"net/http"

	"github.com/daarxwalker/enjin/client"
	"github.com/daarxwalker/enjin/localize"
	"github.com/daarxwalker/enjin/session"
	"github.com/daarxwalker/enjin/validate"
	"github.com/daarxwalker/enjin/web/assets"
	"github.com/daarxwalker/enjin/web/consent"
	"github.com/daarxwalker/enjin/web/csrf"
	"github.com/daarxwalker/enjin/web/flash"
	"github.com/daarxwalker/enjin/web/redirect"
	"github.com/daarxwalker/enjin/web/theme"
)

type Option func(*managers)

func WithClient(m *client.Manager) Option {
	return func(s *managers) {
		s.client = m
	}
}

func WithRedirect(m *redirect.Manager) Option {
	return func(s *managers) {
		s.redirect = m
	}
}

func WithLocalizer(l *localize.Localizer) Option {
	return func(s *managers) {
		s.localizer = l
	}
}

func WithSession(store *session.Store) Option {
	return func(s *managers) {
		s.session = store
	}
}

func WithFlash(m *flash.Manager) Option {
	return func(s *managers) {
		s.flash = m
	}
}

func WithValidate(m *validate.Manager) Option {
	return func(s *managers) {
		s.validate = m
	}
}

func WithTheme(m *theme.Manager) Option {
	return func(s *managers) {
		s.theme = m
	}
}

func WithConsent(m *consent.Manager) Option {
	return func(s *managers) {
		s.consent = m
	}
}

func WithCSRF(m *csrf.InputManager) Option {
	return func(s *managers) {
		s.csrf = m
	}
}

func WithAssets(manifest map[string]string, basepath string) Option {
	return func(s *managers) {
		s.assets = assets.Middleware(manifest, basepath)
	}
}

func WithMiddleware(middlewares ...func(http.Handler) http.Handler) Option {
	return func(s *managers) {
		s.middlewares = append(s.middlewares, middlewares...)
	}
}
