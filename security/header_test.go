package security

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHeaders(t *testing.T) {
	tests := []struct {
		name    string
		options []HeadersOption
		want    map[string]string
	}{
		{
			name: "defaults",
			want: map[string]string{
				"X-Content-Type-Options":     "nosniff",
				"X-Frame-Options":            "DENY",
				"Referrer-Policy":            "strict-origin-when-cross-origin",
				"Cross-Origin-Opener-Policy": "same-origin",
				"Permissions-Policy":         "camera=(), microphone=(), geolocation=(), payment=(), usb=(), browsing-topics=()",
				"Strict-Transport-Security":  "",
				"Content-Security-Policy":    "",
			},
		},
		{
			name:    "hsts",
			options: []HeadersOption{WithHSTS(365 * 24 * time.Hour)},
			want:    map[string]string{"Strict-Transport-Security": "max-age=31536000"},
		},
		{
			name:    "csp",
			options: []HeadersOption{WithCSP("default-src 'self'"), WithCSPReportOnly("img-src 'self'")},
			want: map[string]string{
				"Content-Security-Policy":             "default-src 'self'",
				"Content-Security-Policy-Report-Only": "img-src 'self'",
			},
		},
		{
			name:    "permissions policy override",
			options: []HeadersOption{WithPermissionsPolicy("camera=(self)")},
			want:    map[string]string{"Permissions-Policy": "camera=(self)"},
		},
		{
			name:    "header override and removal",
			options: []HeadersOption{WithHeader("x-frame-options", "SAMEORIGIN"), WithHeader("Cross-Origin-Opener-Policy", "")},
			want: map[string]string{
				"X-Frame-Options":            "SAMEORIGIN",
				"Cross-Origin-Opener-Policy": "",
			},
		},
		{
			name:    "custom header",
			options: []HeadersOption{WithHeader("X-Custom", "1")},
			want:    map[string]string{"X-Custom": "1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			NewHeaders(tt.options...).Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
			for name, want := range tt.want {
				if got := w.Header().Get(name); got != want {
					t.Errorf("%s = %q, want %q", name, got, want)
				}
			}
		})
	}
}
