package assets

import (
	"maps"
	"net/http"
	"net/url"
	"path"
	"slices"

	"github.com/daarxwalker/enjin/httpx"
	"github.com/daarxwalker/enjin/web/head"
)

func Middleware(manifest map[string]string, basepath string) func(http.Handler) http.Handler {
	keys := slices.Sorted(maps.Keys(manifest))
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				var options []head.Option
				for _, key := range keys {
					name := manifest[key]
					href, joinErr := url.JoinPath("/", httpx.PathPrefix(r), basepath, name)
					if joinErr != nil {
						http.Error(
							w,
							"failed to resolve asset path: "+joinErr.Error(),
							http.StatusInternalServerError,
						)
						return
					}
					switch path.Ext(name) {
					case ".css":
						options = append(
							options, head.WithLink(
								head.Link{
									Href: href,
									Type: "text/css",
									Rel:  "stylesheet",
								},
							),
						)
					case ".js":
						options = append(
							options, head.WithScript(
								head.Script{
									Src:   href,
									Defer: true,
								},
							),
						)
					}
				}
				if len(options) == 0 {
					next.ServeHTTP(w, r)
					return
				}
				head.Attach(r.Context(), options...)
				next.ServeHTTP(w, r)
			},
		)
	}
}
