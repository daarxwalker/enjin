package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestRouter(t *testing.T) {
	t.Run("group", func(t *testing.T) {
		ok := func(w http.ResponseWriter, _ *http.Request) error {
			w.WriteHeader(http.StatusNoContent)
			return nil
		}
		mux := chi.NewRouter()
		r := NewRouter(mux, NewDefaultHandler())
		admin := r.Group("/admin", func(r *Router) {
			r.Get("admin", "/", ok)
			r.Group("/users", func(r *Router) {
				r.Get("admin.users", "/{id}", ok)
			})
		})
		admin.Get("admin.settings", "/settings", ok)

		tests := []struct {
			name    string
			pattern string
			path    string
		}{
			{"admin", "/admin/", "/admin/"},
			{"admin.users", "/admin/users/{id}", "/admin/users/1"},
			{"admin.settings", "/admin/settings", "/admin/settings"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				route, exists := r.routeSet[tt.name]
				if !exists {
					t.Fatalf("route %q not registered", tt.name)
				}
				if route.pattern != tt.pattern {
					t.Fatalf("pattern = %q, want %q", route.pattern, tt.pattern)
				}
				w := httptest.NewRecorder()
				mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tt.path, nil))
				if w.Code != http.StatusNoContent {
					t.Fatalf("GET %s = %d, want %d", tt.path, w.Code, http.StatusNoContent)
				}
			})
		}
	})
}
