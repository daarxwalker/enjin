package main

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/daarxwalker/enjin/app"
	"github.com/daarxwalker/enjin/app/openapi"
	"github.com/daarxwalker/enjin/httpx"
	"github.com/daarxwalker/enjin/infra/cache"
	"github.com/daarxwalker/enjin/security"
	"github.com/daarxwalker/enjin/session"
	"github.com/daarxwalker/enjin/validate"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/redis/go-redis/v9"
)

const writeProducts = security.Securable("products.write")

var editor = &security.Role{Name: "editor", Securables: []security.Securable{writeProducts}}

type api struct {
	engine *app.Engine
	spec   *openapi.Spec
}

type errorResponse struct {
	Errors  validate.Result `json:"errors,omitempty"`
	Message string          `json:"message"`
	Status  int             `json:"status"`
}

func newAPI(ctx context.Context, rdb *redis.Client, logger *slog.Logger, options ...app.Option) (*api, error) {
	store := session.NewStore(
		cache.NewClient(cache.NewRedisDriver(rdb)),
		session.WithBearer(),
		session.WithRoles(map[string]*security.Role{editor.Name: editor}),
		session.WithLogger(logger),
	)
	handler := app.NewDefaultHandler(
		app.WithHandlerLogger(logger),
		app.WithHandlerOnError(func(w http.ResponseWriter, _ *http.Request, err *httpx.Err) {
			message := http.StatusText(err.Status)
			if err.Status < http.StatusInternalServerError && err.Raw != nil {
				message = err.Raw.Error()
			}
			if writeErr := httpx.JSON(w, err.Status, errorResponse{Status: err.Status, Message: message}); writeErr != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}),
	)
	spec := openapi.New(
		openapi.WithTitle("Products API"),
		openapi.WithVersion("1.0.0"),
		openapi.WithBearer("Token from POST /login"),
	)
	engine, err := app.NewEngine(
		append(
			[]app.Option{
				app.WithContext(ctx),
				app.WithRouter(app.NewRouter(chi.NewRouter(), handler)),
				app.WithSpec(spec),
				app.WithCORS(cors.Options{
					AllowedOrigins: []string{"http://localhost:5173"},
					AllowedMethods: []string{http.MethodGet, http.MethodPost},
					AllowedHeaders: []string{"Authorization", "Content-Type"},
					MaxAge:         300,
				}),
				app.WithMiddleware(httpx.Middleware, store.Middleware),
			},
			options...,
		)...,
	)
	if err != nil {
		return nil, err
	}
	mountAuth(engine.Router, store)
	mountProducts(engine.Router, newProductRepo())
	return &api{engine: engine, spec: spec}, nil
}
