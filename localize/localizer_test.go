package localize_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/daarxwalker/enjin/localize"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/language"
)

func TestLocalizer(t *testing.T) {
	localizer, err := localize.NewLocalizer(
		localize.WithFS(fstest.MapFS{
			"active.cs.json": {Data: []byte(`{"hello": "Ahoj"}`)},
			"active.en.json": {Data: []byte(`{"hello": "Hello", "products": {"one": "{{.Count}} product", "other": "{{.Count}} products"}}`)},
		}),
		localize.WithDefaultLang(language.Czech),
		localize.WithLangs(language.Czech, language.English),
		localize.WithExtension("json", json.Unmarshal),
	)
	require.NoError(t, err)

	t.Run(
		"lang", func(t *testing.T) {
			tests := []struct {
				name   string
				query  string
				accept string
				want   string
			}{
				{"default", "", "", "cs"},
				{"accept language", "", "en", "en"},
				{"accept language with weights", "", "en,cs;q=0.9", "en"},
				{"accept language by weight", "", "cs;q=0.5,en;q=0.9", "en"},
				{"accept language with region", "", "en-GB,en;q=0.9", "en"},
				{"accept language skips unsupported", "", "de-DE,de;q=0.9,en;q=0.8", "en"},
				{"accept language unsupported", "", "de", "cs"},
				{"accept language invalid", "", "@@@", "cs"},
				{"query", "en", "cs", "en"},
				{"query unsupported", "de", "en", "en"},
				{"query invalid", "xx-!!", "", "cs"},
			}
			for _, tt := range tests {
				t.Run(
					tt.name, func(t *testing.T) {
						var got string
						handler := localizer.Middleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
							got = localize.LangFrom(r.Context())
						}))
						r := httptest.NewRequest(http.MethodGet, "/?lang="+tt.query, nil)
						if tt.accept != "" {
							r.Header.Set("Accept-Language", tt.accept)
						}
						handler.ServeHTTP(httptest.NewRecorder(), r)
						require.Equal(t, tt.want, got)
					},
				)
			}
		},
	)

	t.Run(
		"plural", func(t *testing.T) {
			products := &i18n.Message{
				ID:    "products",
				One:   "{{.Count}} produkt",
				Few:   "{{.Count}} produkty",
				Other: "{{.Count}} produktů",
			}
			more := &i18n.Message{ID: "more", Other: "a dalších {{.Count}}"}
			tests := []struct {
				name    string
				lang    string
				message *i18n.Message
				count   int
				want    string
			}{
				{"czech one", "cs", products, 1, "1 produkt"},
				{"czech few", "cs", products, 3, "3 produkty"},
				{"czech other", "cs", products, 5, "5 produktů"},
				{"czech zero", "cs", products, 0, "0 produktů"},
				{"english one", "en", products, 1, "1 product"},
				{"english other", "en", products, 3, "3 products"},
				{"count without plural forms", "cs", more, 2, "a dalších 2"},
			}
			for _, tt := range tests {
				t.Run(
					tt.name, func(t *testing.T) {
						got := localizer.GetTranslate(tt.lang)(tt.message, map[string]any{"Count": tt.count})
						require.Equal(t, tt.want, got)
					},
				)
			}
		},
	)
}
