package httpx_test

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	"github.com/daarxwalker/enjin/httpx"
)

func TestUtil(t *testing.T) {
	t.Run("host from", func(t *testing.T) {
		tests := []struct {
			name      string
			target    string
			forwarded string
			proxied   bool
			want      string
		}{
			{"request host", "http://example.com/", "", false, "example.com"},
			{"localhost with port", "http://localhost:8080/", "", false, "localhost:8080"},
			{"forwarded host", "http://localhost/", "example.com", true, "example.com"},
			{"forwarded host without proxy", "http://localhost/", "evil.example", false, "localhost"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				r := httptest.NewRequest(http.MethodGet, tt.target, nil)
				if tt.forwarded != "" {
					r.Header.Set("X-Forwarded-Host", tt.forwarded)
				}
				if tt.proxied {
					r = viaTrustedProxy(r)
				}
				if got := httpx.HostFrom(r); got != tt.want {
					t.Fatalf("HostFrom() = %q, want %q", got, tt.want)
				}
			})
		}
	})

	t.Run("create url", func(t *testing.T) {
		tests := []struct {
			name    string
			proxied bool
			want    string
		}{
			{"trusted proxy", true, "https://example.com/products"},
			{"without proxy", false, "http://localhost/products"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				r := httptest.NewRequest(http.MethodGet, "http://localhost/", nil)
				r.Header.Set("X-Forwarded-Proto", "https")
				r.Header.Set("X-Forwarded-Host", "example.com")
				if tt.proxied {
					r = viaTrustedProxy(r)
				}
				if got := httpx.CreateURL(r, "/products"); got != tt.want {
					t.Fatalf("CreateURL() = %q, want %q", got, tt.want)
				}
			})
		}
	})

	t.Run("create url without request host", func(t *testing.T) {
		tests := []struct {
			name        string
			defaultHost string
			path        string
			want        string
		}{
			{"default host", "example.com", "/products", "http://example.com/products"},
			{"relative path", "", "/products", "/products"},
			{"relative root", "", "", "/"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Host = ""
				var got string
				httpx.NewProxy(httpx.WithDefaultHost(tt.defaultHost)).Middleware(
					http.HandlerFunc(
						func(_ http.ResponseWriter, r *http.Request) {
							got = httpx.CreateURL(r, tt.path)
						},
					),
				).ServeHTTP(httptest.NewRecorder(), r)
				if got != tt.want {
					t.Fatalf("CreateURL() = %q, want %q", got, tt.want)
				}
			})
		}
	})

	t.Run("path prefix", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("X-Forwarded-Prefix", "/shop")
		if got := httpx.PathPrefix(r); got != "" {
			t.Fatalf("PathPrefix() without proxy = %q, want empty", got)
		}
		if got := httpx.PathPrefix(viaTrustedProxy(r)); got != "/shop" {
			t.Fatalf("PathPrefix() = %q, want /shop", got)
		}
	})

	t.Run("parse prefixes", func(t *testing.T) {
		got, err := httpx.ParsePrefixes(" 10.0.0.0/8", "", "192.0.2.7", "::ffff:198.51.100.1", "2001:db8::/32", "172.16.5.4/12")
		if err != nil {
			t.Fatalf("ParsePrefixes() error = %v", err)
		}
		want := []netip.Prefix{
			netip.MustParsePrefix("10.0.0.0/8"),
			netip.MustParsePrefix("192.0.2.7/32"),
			netip.MustParsePrefix("198.51.100.1/32"),
			netip.MustParsePrefix("2001:db8::/32"),
			netip.MustParsePrefix("172.16.0.0/12"),
		}
		if len(got) != len(want) {
			t.Fatalf("ParsePrefixes() = %v, want %v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("ParsePrefixes()[%d] = %v, want %v", i, got[i], want[i])
			}
		}
		for _, invalid := range []string{"10.0.0.0/33", "localhost", "10.0.0"} {
			if _, err := httpx.ParsePrefixes(invalid); err == nil {
				t.Fatalf("ParsePrefixes(%q) error = nil, want error", invalid)
			}
		}
	})
	t.Run("client ip", func(t *testing.T) {
		tests := []struct {
			name       string
			remoteAddr string
			want       string
		}{
			{"ipv4 with port", "192.0.2.1:1234", "192.0.2.1"},
			{"ipv4", "192.0.2.1", "192.0.2.1"},
			{"ipv6 with port", "[2001:db8::1]:443", "2001:db8::1"},
			{"ipv6", "2001:db8::1", "2001:db8::1"},
			{"ipv4 mapped ipv6", "::ffff:192.0.2.1", "192.0.2.1"},
			{"invalid", "not-an-ip", "not-an-ip"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.RemoteAddr = tt.remoteAddr
				if got := httpx.ClientIP(r); got != tt.want {
					t.Fatalf("ClientIP() = %q, want %q", got, tt.want)
				}
			})
		}
	})
	t.Run("is local path", func(t *testing.T) {
		tests := []struct {
			name  string
			value string
			want  bool
		}{
			{"root", "/", true},
			{"path with query", "/account?tab=persons#top", true},
			{"empty", "", false},
			{"relative", "account", false},
			{"absolute url", "https://evil.com", false},
			{"scheme relative", "//evil.com", false},
			{"backslash after slash", "/\\evil.com", false},
			{"backslash prefix", "\\\\evil.com", false},
			{"tab inside", "/\t/evil.com", false},
			{"newline inside", "/\n/evil.com", false},
			{"javascript", "javascript:alert(1)", false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := httpx.IsLocalPath(tt.value); got != tt.want {
					t.Fatalf("IsLocalPath(%q) = %v, want %v", tt.value, got, tt.want)
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
