package security

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

func RestrictUUIDv7(params ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				for _, param := range params {
					paramValue := chi.URLParam(r, param)
					if !uuidV7Matcher.MatchString(paramValue) {
						http.Error(w, "invalid uuid format", http.StatusBadRequest)
						return
					}
				}
				next.ServeHTTP(w, r)
			},
		)
	}
}

func RequireBearer(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				credentials, found := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
				if !found || token == "" || subtle.ConstantTimeCompare([]byte(credentials), []byte(token)) != 1 {
					w.Header().Set("WWW-Authenticate", "Bearer")
					http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
					return
				}
				next.ServeHTTP(w, r)
			},
		)
	}
}
