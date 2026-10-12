package head

import (
	"context"
	"net/http"
)

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			ctxv := new(ContextValue)
			ctx := r.Context()
			ctx = context.WithValue(ctx, key, ctxv)
			next.ServeHTTP(w, r.WithContext(ctx))
		},
	)
}
