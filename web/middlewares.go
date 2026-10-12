package web

import (
	"net/http"

	"github.com/daarxwalker/enjin/client"
	"github.com/daarxwalker/enjin/httpx"
	"github.com/daarxwalker/enjin/localize"
	"github.com/daarxwalker/enjin/session"
	"github.com/daarxwalker/enjin/validate"
	"github.com/daarxwalker/enjin/web/consent"
	"github.com/daarxwalker/enjin/web/csrf"
	"github.com/daarxwalker/enjin/web/flash"
	"github.com/daarxwalker/enjin/web/head"
	"github.com/daarxwalker/enjin/web/redirect"
	"github.com/daarxwalker/enjin/web/theme"
)

type managers struct {
	client      *client.Manager
	redirect    *redirect.Manager
	localizer   *localize.Localizer
	session     *session.Store
	flash       *flash.Manager
	validate    *validate.Manager
	theme       *theme.Manager
	consent     *consent.Manager
	csrf        *csrf.InputManager
	assets      func(http.Handler) http.Handler
	middlewares []func(http.Handler) http.Handler
}

func CreateMiddlewareManager(options ...Option) []func(http.Handler) http.Handler {
	s := new(managers)
	for _, option := range options {
		option(s)
	}
	middlewares := append([]func(http.Handler) http.Handler{httpx.Middleware}, s.middlewares...)
	for _, middleware := range []func(http.Handler) http.Handler{
		handleMiddleware(s.client),
		handleMiddleware(s.redirect),
		handleMiddleware(s.localizer),
		handleMiddleware(s.session),
		handleMiddleware(s.flash),
		handleMiddleware(s.validate),
		handleMiddleware(s.theme),
		handleMiddleware(s.consent),
		handleMiddleware(s.csrf),
		head.Middleware,
		s.assets,
	} {
		if middleware != nil {
			middlewares = append(middlewares, middleware)
		}
	}
	return middlewares
}

type middlewareHandler interface {
	comparable
	Middleware(next http.Handler) http.Handler
}

func handleMiddleware[M middlewareHandler](m M) func(http.Handler) http.Handler {
	var zero M
	if m == zero {
		return nil
	}
	return m.Middleware
}
