package consent_test

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"strings"
	"testing"

	"github.com/daarxwalker/enjin/app"
	"github.com/daarxwalker/enjin/httpx"
	"github.com/daarxwalker/enjin/web/consent"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func TestMiddleware(t *testing.T) {
	tests := []struct {
		name    string
		cookie  string
		exists  bool
		cookies *consent.Cookies
	}{
		{name: "no cookie"},
		{name: "invalid cookie", cookie: "%zz"},
		{name: "all", cookie: "analytic=true&marketing=true&preference=true", exists: true, cookies: &consent.Cookies{Analytic: true, Marketing: true, Preference: true}},
		{name: "necessary", cookie: "analytic=false&marketing=false&preference=false", exists: true, cookies: &consent.Cookies{}},
		{name: "unknown values are false", cookie: "analytic=yes&marketing=1", exists: true, cookies: &consent.Cookies{}},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				var got *consent.ContextValue
				h := consent.NewManager().Middleware(
					http.HandlerFunc(
						func(_ http.ResponseWriter, r *http.Request) {
							got = consent.From(r.Context())
						},
					),
				)
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				if tt.cookie != "" {
					r.AddCookie(&http.Cookie{Name: consent.DefaultCookieName, Value: tt.cookie})
				}
				w := httptest.NewRecorder()
				h.ServeHTTP(w, r)
				require.NotNil(t, got)
				require.Equal(t, tt.exists, got.Exists)
				require.Equal(t, tt.cookies, got.Cookies)
				if tt.exists {
					require.Len(t, w.Result().Cookies(), 1)
				} else {
					require.Empty(t, w.Result().Cookies())
				}
			},
		)
	}

	t.Run(
		"without middleware", func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			consent.Attach(w, r, consent.Cookies{Analytic: true})
			require.Nil(t, consent.From(r.Context()))
			require.Empty(t, w.Result().Cookies())
		},
	)
}

func TestParseSaveConsent(t *testing.T) {
	tests := []struct {
		name   string
		form   url.Values
		prefix string
		want   consent.SaveConsent
	}{
		{
			name: "all",
			form: url.Values{"action": {"all"}, "redirect": {"/products"}},
			want: consent.SaveConsent{Redirect: "/products", Analytic: true, Preference: true, Marketing: true},
		},
		{
			name: "necessary ignores categories",
			form: url.Values{"action": {"necessary"}, "redirect": {"/"}, "analytic": {"true"}},
			want: consent.SaveConsent{Redirect: "/"},
		},
		{
			name: "selection",
			form: url.Values{"action": {"selection"}, "redirect": {"/"}, "analytic": {"true"}, "marketing": {"false"}},
			want: consent.SaveConsent{Redirect: "/", Analytic: true},
		},
		{
			name: "unknown action",
			form: url.Values{"action": {"other"}, "redirect": {"/"}, "analytic": {"true"}},
			want: consent.SaveConsent{Redirect: "/"},
		},
		{
			name: "foreign redirect",
			form: url.Values{"action": {"all"}, "redirect": {"//evil.com"}},
			want: consent.SaveConsent{Redirect: "/", Analytic: true, Preference: true, Marketing: true},
		},
		{
			name:   "foreign redirect with prefix",
			form:   url.Values{"action": {"necessary"}, "redirect": {"https://evil.com"}},
			prefix: "/shop",
			want:   consent.SaveConsent{Redirect: "/shop"},
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				r := newFormRequest(tt.form)
				if tt.prefix != "" {
					r.Header.Set("X-Forwarded-Prefix", tt.prefix)
					r = viaTrustedProxy(r)
				}
				require.Equal(t, tt.want, consent.ParseSaveConsent(r))
			},
		)
	}
}

func TestMount(t *testing.T) {
	router := app.NewRouter(chi.NewRouter(), app.NewDefaultHandler())
	consent.Mount(router, "consent")
	h := httpx.Middleware(consent.NewManager().Middleware(router.Raw()))

	t.Run(
		"save and redirect", func(t *testing.T) {
			w := httptest.NewRecorder()
			h.ServeHTTP(w, newFormRequest(url.Values{"action": {"all"}, "redirect": {"/products"}}))
			require.Equal(t, http.StatusSeeOther, w.Code)
			require.Equal(t, "/products", w.Header().Get("Location"))
			cookies := w.Result().Cookies()
			require.Len(t, cookies, 1)
			require.Equal(t, consent.DefaultCookieName, cookies[0].Name)
			values, err := url.ParseQuery(cookies[0].Value)
			require.NoError(t, err)
			require.Equal(t, "true", values.Get(consent.Analytic))
		},
	)

	t.Run(
		"reject cross origin", func(t *testing.T) {
			r := newFormRequest(url.Values{"action": {"all"}})
			r.Header.Set("Sec-Fetch-Site", "cross-site")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			require.Equal(t, http.StatusForbidden, w.Code)
			require.Empty(t, w.Result().Cookies())
		},
	)
}

func newFormRequest(form url.Values) *http.Request {
	r := httptest.NewRequest(http.MethodPost, consent.PathPattern, strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return r
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
