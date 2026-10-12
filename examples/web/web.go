package main

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/daarxwalker/enjin/app"
	"github.com/daarxwalker/enjin/client"
	"github.com/daarxwalker/enjin/cookie"
	"github.com/daarxwalker/enjin/httpx"
	"github.com/daarxwalker/enjin/infra/cache"
	"github.com/daarxwalker/enjin/validate"
	"github.com/daarxwalker/enjin/web"
	"github.com/daarxwalker/enjin/web/csrf"
	"github.com/daarxwalker/enjin/web/flash"
	"github.com/daarxwalker/enjin/web/render"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
)

func newWeb(ctx context.Context, rdb *redis.Client, logger *slog.Logger, secureCookies bool, options ...app.Option) (*app.Engine, error) {
	cacheClient := cache.NewClient(cache.NewRedisDriver(rdb))
	csrfManager := csrf.NewInputManager(
		csrf.NewRedisDriver(rdb),
		csrf.WithLogger(logger),
		csrf.WithRejectHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			flash.Attach(r.Context(), flash.WithError("The form has expired. Please try again."))
			if err := httpx.Refresh(w, r); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})),
	)
	handler := app.NewDefaultHandler(
		app.WithHandlerLogger(logger),
		app.WithHandlerOnError(func(w http.ResponseWriter, r *http.Request, err *httpx.Err) {
			if renderErr := render.Templ(w, r, err.Status, errorPage(http.StatusText(err.Status)), layout()); renderErr != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}),
	)
	engine, err := app.NewEngine(
		append(
			[]app.Option{
				app.WithContext(ctx),
				app.WithRouter(app.NewRouter(chi.NewRouter(), handler)),
				app.WithMiddleware(web.CreateMiddlewareManager(
					web.WithClient(client.NewManager(cacheClient,
						client.WithConfig(cookie.WithSecure(secureCookies)),
						client.WithLogger(logger))),
					web.WithFlash(flash.NewManager(cacheClient, flash.WithLogger(logger))),
					web.WithValidate(validate.NewManager(cacheClient, validate.WithLogger(logger))),
					web.WithCSRF(csrfManager),
				)...),
			},
			options...,
		)...,
	)
	if err != nil {
		return nil, err
	}
	mountGuestbook(engine.Router, csrfManager, newGuestbook())
	engine.Router.NotFound(func(http.ResponseWriter, *http.Request) error {
		return httpx.StatusError(http.StatusNotFound)
	})
	return engine, nil
}
