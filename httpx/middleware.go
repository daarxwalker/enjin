package httpx

import (
	"context"
	"maps"
	"net/http"
	"net/url"
)

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			ctxv := new(ContextValue{
				PathPrefix: PathPrefix(r),
				Host:       HostFrom(r),
				Query:      make(url.Values),
			})
			{
				currentURL, err := url.JoinPath("/", ctxv.PathPrefix, r.URL.Path)
				if err != nil {
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					return
				}
				if r.URL.RawQuery != "" {
					currentURL += "?" + r.URL.RawQuery
				}
				ctxv.URL = currentURL
			}
			{
				query := r.URL.Query()
				if len(query) > 0 {
					maps.Copy(ctxv.Query, query)
				}
			}
			ctx = context.WithValue(ctx, key, ctxv)
			next.ServeHTTP(w, r.WithContext(ctx))
		},
	)
}
