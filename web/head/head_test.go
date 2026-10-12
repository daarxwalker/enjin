package head_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/daarxwalker/enjin/web/head"

	"github.com/stretchr/testify/require"
)

func TestHead(t *testing.T) {
	t.Run(
		"attach options in order", func(t *testing.T) {
			var got *head.ContextValue
			h := head.Middleware(
				http.HandlerFunc(
					func(_ http.ResponseWriter, r *http.Request) {
						head.Attach(r.Context(), head.WithTitle("First"), head.WithDescription("About"))
						head.Attach(
							r.Context(),
							head.WithTitle("Products"),
							head.WithMeta("robots", "noindex"),
							head.WithLink(head.Link{Href: "/x.css", Type: "text/css", Rel: "stylesheet"}),
							head.WithScript(head.Script{Src: "/x.js", Defer: true}),
						)
						got = head.From(r.Context())
					},
				),
			)
			h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
			require.Equal(
				t, &head.ContextValue{
					Title:   "Products",
					Metas:   []head.Meta{{Name: "description", Content: "About"}, {Name: "robots", Content: "noindex"}},
					Links:   []head.Link{{Href: "/x.css", Type: "text/css", Rel: "stylesheet"}},
					Scripts: []head.Script{{Src: "/x.js", Defer: true}},
				}, got,
			)
		},
	)

	t.Run(
		"each request starts empty", func(t *testing.T) {
			var titles []string
			h := head.Middleware(
				http.HandlerFunc(
					func(_ http.ResponseWriter, r *http.Request) {
						titles = append(titles, head.From(r.Context()).Title)
						head.Attach(r.Context(), head.WithTitle("set"))
					},
				),
			)
			for range 2 {
				h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
			}
			require.Equal(t, []string{"", ""}, titles)
		},
	)

	t.Run(
		"without middleware", func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			head.Attach(r.Context(), head.WithTitle("x"))
			require.Equal(t, &head.ContextValue{}, head.From(r.Context()))
		},
	)
}
