package app

import (
	"context"
	"net/http"
)

func attachRoute(route *Route) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				ctxVal := &RouteContextValue{
					Route: route,
				}
				ctx := r.Context()
				ctx = context.WithValue(ctx, routeCtxKey, ctxVal)
				next.ServeHTTP(w, r.WithContext(ctx))
			},
		)
	}
}

func limitRequestBody(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				if r.ContentLength > maxBytes {
					http.Error(w, http.StatusText(http.StatusRequestEntityTooLarge), http.StatusRequestEntityTooLarge)
					return
				}
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
				next.ServeHTTP(w, r)
			},
		)
	}
}
