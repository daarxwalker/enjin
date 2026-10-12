package theme_test

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	"github.com/daarxwalker/enjin/cookie"
	"github.com/daarxwalker/enjin/httpx"
	"github.com/daarxwalker/enjin/web/theme"

	"github.com/stretchr/testify/require"
)

func TestTheme(t *testing.T) {
	t.Run(
		"resolve value", func(t *testing.T) {
			tests := []struct {
				name    string
				options []theme.Option
				cookie  string
				want    theme.Value
			}{
				{name: "no cookie", want: theme.Light},
				{name: "dark cookie", cookie: "dark", want: theme.Dark},
				{name: "invalid cookie", cookie: "<script>", want: theme.Light},
				{name: "configured default", options: []theme.Option{theme.WithDefaultTheme(theme.Dark)}, want: theme.Dark},
				{name: "invalid default ignored", options: []theme.Option{theme.WithDefaultTheme("blue")}, want: theme.Light},
				{name: "invalid cookie with default", options: []theme.Option{theme.WithDefaultTheme(theme.Dark)}, cookie: "blue", want: theme.Dark},
			}
			for _, tt := range tests {
				t.Run(
					tt.name, func(t *testing.T) {
						r := httptest.NewRequest(http.MethodGet, "/", nil)
						if tt.cookie != "" {
							r.AddCookie(&http.Cookie{Name: theme.CookieName, Value: tt.cookie})
						}
						w, got := serve(theme.NewManager(tt.options...), r)
						require.Equal(t, http.StatusOK, w.Code)
						require.Equal(t, tt.want, got)
						cookies := w.Result().Cookies()
						require.Len(t, cookies, 1)
						require.Equal(t, tt.want.String(), cookies[0].Value)
					},
				)
			}
		},
	)

	t.Run(
		"query sets cookie and redirects", func(t *testing.T) {
			tests := []struct {
				name     string
				target   string
				options  []theme.Option
				want     theme.Value
				location string
			}{
				{name: "dark", target: "/page?theme=dark&tab=2", want: theme.Dark, location: "/page?tab=2"},
				{name: "invalid falls back to default", target: "/page?theme=blue", options: []theme.Option{theme.WithDefaultTheme(theme.Dark)}, want: theme.Dark, location: "/page"},
				{name: "foreign host", target: "//evil.com?theme=dark", want: theme.Dark, location: "/"},
				{name: "escaped backslash stays local", target: "/%5Cevil.com?theme=dark", want: theme.Dark, location: "/%5Cevil.com"},
			}
			for _, tt := range tests {
				t.Run(
					tt.name, func(t *testing.T) {
						w, _ := serve(theme.NewManager(tt.options...), httptest.NewRequest(http.MethodGet, tt.target, nil))
						require.Equal(t, http.StatusSeeOther, w.Code)
						require.Equal(t, tt.location, w.Header().Get("Location"))
						cookies := w.Result().Cookies()
						require.Len(t, cookies, 1)
						require.Equal(t, tt.want.String(), cookies[0].Value)
					},
				)
			}
		},
	)

	t.Run(
		"redirect keeps the path prefix of a trusted proxy", func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/page?theme=dark&tab=2", nil)
			r.RemoteAddr = "192.0.2.1:1234"
			r.Header.Set("X-Forwarded-Prefix", "/shop")
			var w *httptest.ResponseRecorder
			httpx.NewProxy(httpx.WithTrustedProxies(netip.MustParsePrefix("192.0.2.0/24"))).Middleware(
				http.HandlerFunc(
					func(_ http.ResponseWriter, r *http.Request) {
						w, _ = serve(theme.NewManager(), r)
					},
				),
			).ServeHTTP(httptest.NewRecorder(), r)
			require.Equal(t, http.StatusSeeOther, w.Code)
			require.Equal(t, "/shop/page?tab=2", w.Header().Get("Location"))
		},
	)

	t.Run(
		"configured cookie name", func(t *testing.T) {
			m := theme.NewManager(theme.WithConfig(cookie.WithName("X-Mode")))
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.AddCookie(&http.Cookie{Name: "X-Mode", Value: "dark"})
			w, got := serve(m, r)
			require.Equal(t, theme.Dark, got)
			require.Equal(t, "X-Mode", w.Result().Cookies()[0].Name)
		},
	)

	t.Run(
		"attach", func(t *testing.T) {
			tests := []struct {
				name  string
				value theme.Value
				want  theme.Value
			}{
				{name: "valid", value: theme.Dark, want: theme.Dark},
				{name: "invalid ignored", value: "blue", want: theme.Light},
			}
			for _, tt := range tests {
				t.Run(
					tt.name, func(t *testing.T) {
						var got theme.Value
						h := theme.NewManager().Middleware(
							http.HandlerFunc(
								func(w http.ResponseWriter, r *http.Request) {
									theme.Attach(w, r, tt.value)
									got = theme.From(r.Context())
								},
							),
						)
						w := httptest.NewRecorder()
						h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
						require.Equal(t, tt.want, got)
						cookies := w.Result().Cookies()
						require.Equal(t, tt.want.String(), cookies[len(cookies)-1].Value)
					},
				)
			}
		},
	)

	t.Run(
		"without middleware", func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			require.Equal(t, theme.Light, theme.From(r.Context()))
			require.False(t, theme.IsDark(r.Context()))
		},
	)
}

func serve(m *theme.Manager, r *http.Request) (*httptest.ResponseRecorder, theme.Value) {
	var got theme.Value
	h := m.Middleware(
		http.HandlerFunc(
			func(_ http.ResponseWriter, r *http.Request) {
				got = theme.From(r.Context())
			},
		),
	)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w, got
}
