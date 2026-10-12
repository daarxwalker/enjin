package backlink_test

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	"github.com/daarxwalker/enjin/httpx"
	"github.com/daarxwalker/enjin/web/backlink"
)

func TestAccessor(t *testing.T) {
	t.Run("query", func(t *testing.T) {
		tests := []struct {
			name      string
			referer   string
			forwarded string
			path      string
			want      string
			ok        bool
		}{
			{"matching referer", "http://example.com/products?search=krem&sort=price", "", "/products", "search=krem&sort=price", true},
			{"matching referer without query", "http://example.com/products", "", "/products", "", true},
			{"forwarded host", "https://example.org/products?search=krem", "example.org", "/products", "search=krem", true},
			{"path prefix", "http://example.com/cs/products?search=krem", "", "/cs/products", "search=krem", true},
			{"other host", "http://evil.com/products?search=krem", "", "/products", "", false},
			{"other host than forwarded", "http://example.com/products?search=krem", "example.org", "/products", "", false},
			{"other path", "http://example.com/products/krem", "", "/products", "", false},
			{"missing referer", "", "", "/products", "", false},
			{"relative referer", "/products?search=krem", "", "/products", "", false},
			{"invalid referer", "http://example.com/%zz", "", "/products", "", false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				r := httptest.NewRequest(http.MethodGet, "/products/detail", nil)
				if tt.referer != "" {
					r.Header.Set("Referer", tt.referer)
				}
				if tt.forwarded != "" {
					r.Header.Set("X-Forwarded-Host", tt.forwarded)
					r = viaTrustedProxy(r)
				}
				got, ok := backlink.Query(r, tt.path)
				if ok != tt.ok {
					t.Fatalf("ok = %v, want %v", ok, tt.ok)
				}
				if got.Encode() != tt.want {
					t.Errorf("query = %q, want %q", got.Encode(), tt.want)
				}
			})
		}
	})
}

func viaTrustedProxy(r *http.Request) *http.Request {
	var proxied *http.Request
	httpx.NewProxy(httpx.WithTrustedProxies(netip.MustParsePrefix("192.0.2.0/24"))).Middleware(
		http.HandlerFunc(
			func(_ http.ResponseWriter, r *http.Request) {
				proxied = r
			},
		),
	).ServeHTTP(httptest.NewRecorder(), r)
	return proxied
}
