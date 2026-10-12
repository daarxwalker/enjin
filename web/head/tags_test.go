package head_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/daarxwalker/enjin/web/head"

	"github.com/stretchr/testify/require"
)

func TestTags(t *testing.T) {
	t.Run(
		"render attached values", func(t *testing.T) {
			var (
				html      strings.Builder
				renderErr error
			)
			h := head.Middleware(
				http.HandlerFunc(
					func(_ http.ResponseWriter, r *http.Request) {
						head.Attach(
							r.Context(),
							head.WithTitle(`Krém <b>"A&B"</b>`),
							head.WithDescription(`Popis "s" uvozovkami`),
							head.WithLink(head.Link{Href: "/app.css", Rel: "stylesheet", Type: "text/css"}),
							head.WithScript(head.Script{Src: "/app.js", Defer: true}),
							head.WithScript(head.Script{Src: "/sync.js"}),
						)
						renderErr = head.Tags().Render(r.Context(), &html)
					},
				),
			)
			h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
			require.NoError(t, renderErr)
			require.Equal(
				t,
				`<title>Krém &lt;b&gt;&#34;A&amp;B&#34;&lt;/b&gt;</title>`+
					`<meta name="description" content="Popis &#34;s&#34; uvozovkami">`+
					`<link href="/app.css" rel="stylesheet" type="text/css">`+
					`<script src="/app.js" defer></script>`+
					`<script src="/sync.js"></script>`,
				html.String(),
			)
		},
	)

	t.Run(
		"without middleware", func(t *testing.T) {
			var html strings.Builder
			require.NoError(t, head.Tags().Render(context.Background(), &html))
			require.Empty(t, html.String())
		},
	)
}
