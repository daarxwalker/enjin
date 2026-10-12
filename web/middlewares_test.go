package web_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"testing/fstest"

	"github.com/daarxwalker/enjin/client"
	"github.com/daarxwalker/enjin/httpx"
	"github.com/daarxwalker/enjin/infra/cache"
	"github.com/daarxwalker/enjin/internal/testredis"
	"github.com/daarxwalker/enjin/localize"
	"github.com/daarxwalker/enjin/session"
	"github.com/daarxwalker/enjin/validate"
	"github.com/daarxwalker/enjin/web"
	"github.com/daarxwalker/enjin/web/consent"
	"github.com/daarxwalker/enjin/web/csrf"
	"github.com/daarxwalker/enjin/web/flash"
	"github.com/daarxwalker/enjin/web/head"
	"github.com/daarxwalker/enjin/web/redirect"
	"github.com/daarxwalker/enjin/web/theme"

	"github.com/stretchr/testify/require"
	"golang.org/x/text/language"
)

func TestMiddlewares(t *testing.T) {
	t.Run(
		"without managers", func(t *testing.T) {
			var title string
			serve(
				web.CreateMiddlewareManager(), func(r *http.Request) {
					require.NotNil(t, httpx.From(r.Context()))
					head.Attach(r.Context(), head.WithTitle("x"))
					title = head.From(r.Context()).Title
				},
			)
			require.Equal(t, "x", title)
			require.Len(t, web.CreateMiddlewareManager(), 2)
		},
	)

	t.Run(
		"nil managers are skipped", func(t *testing.T) {
			require.Len(t, web.CreateMiddlewareManager(web.WithClient(nil), web.WithSession(nil), web.WithCSRF(nil)), 2)
		},
	)

	t.Run(
		"all managers", func(t *testing.T) {
			rdb := testredis.New(t)
			cacheClient := cache.NewClient(cache.NewRedisDriver(rdb))
			localizer, err := localize.NewLocalizer(
				localize.WithFS(fstest.MapFS{"active.en.json": {Data: []byte(`{}`)}}),
				localize.WithDefaultLang(language.English),
				localize.WithLangs(language.English),
				localize.WithExtension("json", json.Unmarshal),
			)
			require.NoError(t, err)

			var extraSawHTTPX, extraSawClient bool
			extra := func(next http.Handler) http.Handler {
				return http.HandlerFunc(
					func(w http.ResponseWriter, r *http.Request) {
						extraSawHTTPX = httpx.From(r.Context()) != nil
						extraSawClient = client.IDFrom(r.Context()) != ""
						next.ServeHTTP(w, r)
					},
				)
			}
			middlewares := web.CreateMiddlewareManager(
				web.WithAssets(map[string]string{"app.css": "app-1a2b.css"}, "/"),
				web.WithCSRF(csrf.NewInputManager(csrf.NewRedisDriver(rdb))),
				web.WithConsent(consent.NewManager()),
				web.WithTheme(theme.NewManager()),
				web.WithValidate(validate.NewManager(cacheClient)),
				web.WithFlash(flash.NewManager(cacheClient)),
				web.WithSession(session.NewStore(cacheClient)),
				web.WithLocalizer(localizer),
				web.WithRedirect(redirect.NewManager()),
				web.WithClient(client.NewManager(cacheClient)),
				web.WithMiddleware(extra),
			)
			require.Len(t, middlewares, 13)

			var checked bool
			serve(
				middlewares, func(r *http.Request) {
					ctx := r.Context()
					require.NotEmpty(t, client.IDFrom(ctx))
					require.NotNil(t, localize.TranslateFrom(ctx))
					require.NotNil(t, consent.From(ctx))
					require.NotNil(t, csrf.NewInputFrom(ctx))
					require.Equal(t, []head.Link{{Href: "/app-1a2b.css", Type: "text/css", Rel: "stylesheet"}}, head.From(ctx).Links)
					checked = true
				},
			)
			require.True(t, checked)
			require.True(t, extraSawHTTPX)
			require.False(t, extraSawClient)
		},
	)
}

func serve(middlewares []func(http.Handler) http.Handler, check func(r *http.Request)) {
	var h http.Handler = http.HandlerFunc(
		func(_ http.ResponseWriter, r *http.Request) {
			check(r)
		},
	)
	for _, middleware := range slices.Backward(middlewares) {
		h = middleware(h)
	}
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
}
