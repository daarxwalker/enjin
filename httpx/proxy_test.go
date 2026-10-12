package httpx_test

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	"github.com/daarxwalker/enjin/httpx"
)

type proxied struct {
	header        http.Header
	remoteAddr    string
	status        int
	viaCloudflare bool
}

func serveProxy(proxy *httpx.Proxy, r *http.Request) proxied {
	var got proxied
	w := httptest.NewRecorder()
	proxy.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.remoteAddr = r.RemoteAddr
		got.header = r.Header.Clone()
		got.viaCloudflare = httpx.IsFromCloudflare(r)
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(w, r)
	got.status = w.Code
	return got
}

func TestProxy(t *testing.T) {
	trusted := httpx.WithTrustedProxies(netip.MustParsePrefix("10.0.0.0/8"))

	t.Run("client ip", func(t *testing.T) {
		tests := []struct {
			name       string
			remoteAddr string
			forwarded  []string
			want       string
		}{
			{"untrusted peer ignores forwarded for", "203.0.113.9:1234", []string{"198.51.100.1"}, "203.0.113.9:1234"},
			{"trusted peer", "10.0.0.2:1234", []string{"203.0.113.7"}, "203.0.113.7"},
			{"rightmost untrusted", "10.0.0.2:1234", []string{"198.51.100.1, 203.0.113.7, 10.0.0.5"}, "203.0.113.7"},
			{"multiple headers", "10.0.0.2:1234", []string{"198.51.100.1", "203.0.113.7"}, "203.0.113.7"},
			{"invalid entry keeps peer", "10.0.0.2:1234", []string{"203.0.113.7, nonsense"}, "10.0.0.2:1234"},
			{"only trusted entries keep peer", "10.0.0.2:1234", []string{"10.0.0.5"}, "10.0.0.2:1234"},
			{"trusted peer without header", "10.0.0.2:1234", nil, "10.0.0.2:1234"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.RemoteAddr = tt.remoteAddr
				for _, value := range tt.forwarded {
					r.Header.Add("X-Forwarded-For", value)
				}
				if got := serveProxy(httpx.NewProxy(trusted), r); got.remoteAddr != tt.want {
					t.Fatalf("RemoteAddr = %q, want %q", got.remoteAddr, tt.want)
				}
			})
		}
	})

	t.Run("headers", func(t *testing.T) {
		spoofed := map[string]string{
			"X-Forwarded-For":    "198.51.100.1",
			"X-Forwarded-Host":   "evil.example",
			"X-Forwarded-Proto":  "https",
			"X-Forwarded-Prefix": "/evil",
			"X-Real-IP":          "198.51.100.2",
			"True-Client-IP":     "198.51.100.3",
			"Forwarded":          "for=198.51.100.4",
		}
		tests := []struct {
			name       string
			remoteAddr string
			kept       []string
		}{
			{"untrusted peer", "203.0.113.9:1234", nil},
			{"trusted peer", "10.0.0.2:1234", []string{"X-Forwarded-For", "X-Forwarded-Host", "X-Forwarded-Proto", "X-Forwarded-Prefix"}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.RemoteAddr = tt.remoteAddr
				for name, value := range spoofed {
					r.Header.Set(name, value)
				}
				got := serveProxy(httpx.NewProxy(trusted), r)
				keptSet := make(map[string]bool, len(tt.kept))
				for _, name := range tt.kept {
					keptSet[name] = true
				}
				for name := range spoofed {
					if has := got.header.Get(name) != ""; has != keptSet[name] {
						t.Errorf("header %s present = %v, want %v", name, has, keptSet[name])
					}
				}
			})
		}
	})

	t.Run("forwarded proto", func(t *testing.T) {
		tests := []struct {
			proto string
			want  string
		}{
			{"https", "https"},
			{"http", "http"},
			{"javascript", ""},
		}
		for _, tt := range tests {
			t.Run(tt.proto, func(t *testing.T) {
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.RemoteAddr = "10.0.0.2:1234"
				r.Header.Set("X-Forwarded-Proto", tt.proto)
				if got := serveProxy(httpx.NewProxy(trusted), r).header.Get("X-Forwarded-Proto"); got != tt.want {
					t.Fatalf("X-Forwarded-Proto = %q, want %q", got, tt.want)
				}
			})
		}
	})

	t.Run("allowed hosts", func(t *testing.T) {
		proxy := httpx.NewProxy(
			trusted,
			httpx.WithAllowedHosts("example.com", "Example.org"),
			httpx.WithHostExemptPaths("/metrics"),
		)
		tests := []struct {
			name       string
			remoteAddr string
			host       string
			forwarded  string
			path       string
			want       int
		}{
			{"allowed host", "203.0.113.9:1234", "example.com", "", "/", http.StatusOK},
			{"allowed host with port", "203.0.113.9:1234", "example.com:8080", "", "/", http.StatusOK},
			{"allowed host other case", "203.0.113.9:1234", "EXAMPLE.ORG", "", "/", http.StatusOK},
			{"other host", "203.0.113.9:1234", "evil.example", "", "/", http.StatusMisdirectedRequest},
			{"untrusted forwarded host ignored", "203.0.113.9:1234", "example.com", "evil.example", "/", http.StatusOK},
			{"trusted forwarded host", "10.0.0.2:1234", "app.example:80", "example.org", "/", http.StatusOK},
			{"trusted forwarded other host", "10.0.0.2:1234", "example.com", "evil.example", "/", http.StatusMisdirectedRequest},
			{"exempt path", "10.0.0.2:1234", "app.example:80", "", "/metrics", http.StatusOK},
			{"exempt subpath", "10.0.0.2:1234", "app.example:80", "", "/metrics/", http.StatusOK},
			{"not exempt prefix", "10.0.0.2:1234", "app.example:80", "", "/metricsx", http.StatusMisdirectedRequest},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				r := httptest.NewRequest(http.MethodGet, tt.path, nil)
				r.RemoteAddr = tt.remoteAddr
				r.Host = tt.host
				if tt.forwarded != "" {
					r.Header.Set("X-Forwarded-Host", tt.forwarded)
				}
				if got := serveProxy(proxy, r).status; got != tt.want {
					t.Fatalf("status = %d, want %d", got, tt.want)
				}
			})
		}
	})

	t.Run("no allowed hosts", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Host = "anything.example"
		if got := serveProxy(httpx.NewProxy(), r).status; got != http.StatusOK {
			t.Fatalf("status = %d, want %d", got, http.StatusOK)
		}
	})

	t.Run("cloudflare", func(t *testing.T) {
		tests := []struct {
			name           string
			remoteAddr     string
			forwarded      string
			wantRemoteAddr string
			enabled        bool
			wantVia        bool
		}{
			{"disabled strips headers", "10.0.0.2:1234", "203.0.113.7, 104.16.0.1", "104.16.0.1", false, false},
			{"enabled through cloudflare hop", "10.0.0.2:1234", "203.0.113.7, 104.16.0.1", "203.0.113.7", true, true},
			{"enabled without cloudflare hop", "10.0.0.2:1234", "203.0.113.7", "203.0.113.7", true, false},
			{"enabled direct untrusted peer", "203.0.113.9:1234", "198.51.100.1, 104.16.0.1", "203.0.113.9:1234", true, false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.RemoteAddr = tt.remoteAddr
				r.Header.Set("X-Forwarded-For", tt.forwarded)
				r.Header.Set("CF-Connecting-IP", "198.51.100.66")
				r.Header.Set("X-Verified-Bot", "1")
				got := serveProxy(httpx.NewProxy(trusted, httpx.WithCloudflare(tt.enabled)), r)
				if got.remoteAddr != tt.wantRemoteAddr {
					t.Fatalf("RemoteAddr = %q, want %q", got.remoteAddr, tt.wantRemoteAddr)
				}
				if got.viaCloudflare != tt.wantVia {
					t.Fatalf("ViaCloudflare = %v, want %v", got.viaCloudflare, tt.wantVia)
				}
				kept := got.header.Get("CF-Connecting-IP") != "" && got.header.Get("X-Verified-Bot") != ""
				if kept != tt.wantVia {
					t.Fatalf("cloudflare headers kept = %v, want %v", kept, tt.wantVia)
				}
			})
		}
	})
}
