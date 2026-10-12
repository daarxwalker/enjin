package security

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireBearer(t *testing.T) {
	tests := []struct {
		name          string
		token         string
		authorization string
		want          int
	}{
		{"valid token", "secret", "Bearer secret", http.StatusOK},
		{"missing header", "secret", "", http.StatusUnauthorized},
		{"wrong token", "secret", "Bearer other", http.StatusUnauthorized},
		{"missing scheme", "secret", "secret", http.StatusUnauthorized},
		{"basic scheme", "secret", "Basic secret", http.StatusUnauthorized},
		{"empty token", "", "Bearer ", http.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.authorization != "" {
				r.Header.Set("Authorization", tt.authorization)
			}
			w := httptest.NewRecorder()
			RequireBearer(tt.token)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})).ServeHTTP(w, r)
			if w.Code != tt.want {
				t.Fatalf("status = %d, want %d", w.Code, tt.want)
			}
			if tt.want == http.StatusUnauthorized && w.Header().Get("WWW-Authenticate") != "Bearer" {
				t.Fatalf("WWW-Authenticate = %q, want Bearer", w.Header().Get("WWW-Authenticate"))
			}
		})
	}
}
