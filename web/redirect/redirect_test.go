package redirect_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/daarxwalker/enjin/web/redirect"

	"github.com/stretchr/testify/require"
)

func TestRedirect(t *testing.T) {
	t.Run(
		"attach and read cookie", func(t *testing.T) {
			h := newTestHandler()
			attached := h.serve(httptest.NewRequest(http.MethodGet, "/attach?redirect=/account", nil))
			cookies := attached.Result().Cookies()
			require.Len(t, cookies, 1)
			require.Equal(t, "/account", cookies[0].Value)

			r := httptest.NewRequest(http.MethodGet, "/read", nil)
			r.AddCookie(cookies[0])
			read := h.serve(r)
			require.Equal(t, "/account", read.Body.String())
			destroyed := read.Result().Cookies()
			require.Len(t, destroyed, 1)
			require.Negative(t, destroyed[0].MaxAge)
		},
	)

	t.Run(
		"reject foreign query", func(t *testing.T) {
			h := newTestHandler()
			attached := h.serve(httptest.NewRequest(http.MethodGet, "/attach?redirect=//evil.com", nil))
			require.Empty(t, attached.Result().Cookies())
		},
	)

	t.Run(
		"reject foreign cookie", func(t *testing.T) {
			h := newTestHandler()
			r := httptest.NewRequest(http.MethodGet, "/read", nil)
			r.AddCookie(&http.Cookie{Name: "X-Redirect", Value: "https://evil.com"})
			require.Empty(t, h.serve(r).Body.String())
		},
	)

	t.Run(
		"without middleware", func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			redirect.Attach(w, r, "/account")
			require.Empty(t, w.Result().Cookies())
			require.Empty(t, redirect.FromCookie(w, r))
		},
	)
}

type testHandler struct {
	handler http.Handler
}

func newTestHandler() *testHandler {
	mux := http.NewServeMux()
	mux.HandleFunc(
		"/attach", func(w http.ResponseWriter, r *http.Request) {
			if to := redirect.FromQuery(r); to != "" {
				redirect.Attach(w, r, to)
			}
		},
	)
	mux.HandleFunc(
		"/read", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(redirect.FromCookie(w, r)))
		},
	)
	return &testHandler{handler: redirect.NewManager().Middleware(mux)}
}

func (h *testHandler) serve(r *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	h.handler.ServeHTTP(w, r)
	return w
}
