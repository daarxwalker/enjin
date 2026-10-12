package assets_test

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
	"testing/fstest"

	"github.com/daarxwalker/enjin/httpx"
	"github.com/daarxwalker/enjin/web/assets"
	"github.com/daarxwalker/enjin/web/head"

	"github.com/stretchr/testify/require"
)

func TestAssets(t *testing.T) {
	manifest := map[string]string{
		"vendor.js":  "vendor-3c4d.js",
		"app.css":    "app-1a2b.css",
		"app.js":     "app-5e6f.js",
		"print.css":  "print-7a8b.css",
		"admin.js":   "admin-9c0d.js",
		"font.woff2": "font-1e2f.woff2",
	}

	t.Run(
		"links and scripts in manifest key order", func(t *testing.T) {
			for range 20 {
				got := serve(t, manifest, "/static", httptest.NewRequest(http.MethodGet, "/", nil))
				require.Equal(
					t, []head.Link{
						{Href: "/static/app-1a2b.css", Type: "text/css", Rel: "stylesheet"},
						{Href: "/static/print-7a8b.css", Type: "text/css", Rel: "stylesheet"},
					}, got.Links,
				)
				require.Equal(
					t, []head.Script{
						{Src: "/static/admin-9c0d.js", Defer: true},
						{Src: "/static/app-5e6f.js", Defer: true},
						{Src: "/static/vendor-3c4d.js", Defer: true},
					}, got.Scripts,
				)
			}
		},
	)

	t.Run(
		"path prefix", func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("X-Forwarded-Prefix", "/shop")
			got := serve(t, map[string]string{"app.css": "app.css"}, "/", viaTrustedProxy(r))
			require.Equal(t, "/shop/app.css", got.Links[0].Href)
		},
	)

	t.Run(
		"load manifest", func(t *testing.T) {
			fsys := fstest.MapFS{
				"manifest.json": {Data: []byte(`{"app.js":"app-5e6f.js"}`)},
				"broken.json":   {Data: []byte(`{`)},
			}
			m, err := assets.LoadManifest(fsys, "manifest.json")
			require.NoError(t, err)
			require.Equal(t, map[string]string{"app.js": "app-5e6f.js"}, m)
			_, err = assets.LoadManifest(fsys, "broken.json")
			require.Error(t, err)
			_, err = assets.LoadManifest(fsys, "missing.json")
			require.Error(t, err)
		},
	)
}

func serve(t *testing.T, manifest map[string]string, basepath string, r *http.Request) *head.ContextValue {
	t.Helper()
	var got *head.ContextValue
	h := head.Middleware(
		assets.Middleware(manifest, basepath)(
			http.HandlerFunc(
				func(_ http.ResponseWriter, r *http.Request) {
					got = head.From(r.Context())
				},
			),
		),
	)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, got)
	return got
}

func viaTrustedProxy(r *http.Request) *http.Request {
	var proxied *http.Request
	httpx.NewProxy(httpx.WithTrustedProxies(netip.MustParsePrefix("192.0.2.0/24"))).Middleware(
		http.HandlerFunc(
			func(_ http.ResponseWriter, r *http.Request) {
				proxied = r
			},
		),
	).ServeHTTP(httptest.NewRecorder(), r)
	return proxied
}
