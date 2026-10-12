package abuse

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type botVerifierFakeResolver struct {
	names   map[string][]string
	addrs   map[string][]net.IPAddr
	err     error
	lookups int
}

func (r *botVerifierFakeResolver) LookupAddr(_ context.Context, addr string) ([]string, error) {
	r.lookups++
	if r.err != nil {
		return nil, r.err
	}
	names, ok := r.names[addr]
	if !ok {
		return nil, &net.DNSError{Err: "no such host", Name: addr, IsNotFound: true}
	}
	return names, nil
}

func (r *botVerifierFakeResolver) LookupIPAddr(_ context.Context, host string) ([]net.IPAddr, error) {
	return r.addrs[host], nil
}

func TestBotVerifier(t *testing.T) {
	start := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	googlebot := "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)"

	t.Run("reverse and forward dns confirm googlebot and result is cached", func(t *testing.T) {
		resolver := &botVerifierFakeResolver{
			names: map[string][]string{"66.249.66.1": {"crawl-66-249-66-1.googlebot.com."}},
			addrs: map[string][]net.IPAddr{"crawl-66-249-66-1.googlebot.com": {{IP: net.ParseIP("66.249.66.1")}}},
		}
		verifier := NewBotVerifier(NewMemoryStore(t.Context()), WithResolver(resolver))
		addr := netip.MustParseAddr("66.249.66.1")
		require.True(t, verifier.verifyBot(newBotVerifierRequest(googlebot), addr, start))
		require.True(t, verifier.verifyBot(newBotVerifierRequest(googlebot), addr, start.Add(time.Hour)))
		require.Equal(t, 1, resolver.lookups)
	})

	t.Run("spoofed user agent from other host is rejected", func(t *testing.T) {
		resolver := &botVerifierFakeResolver{
			names: map[string][]string{"203.0.113.7": {"googlebot.com.evil.example."}},
			addrs: map[string][]net.IPAddr{"googlebot.com.evil.example": {{IP: net.ParseIP("203.0.113.7")}}},
		}
		verifier := NewBotVerifier(NewMemoryStore(t.Context()), WithResolver(resolver))
		require.False(t, verifier.verifyBot(newBotVerifierRequest(googlebot), netip.MustParseAddr("203.0.113.7"), start))
	})

	t.Run("forward lookup must point back to the same address", func(t *testing.T) {
		resolver := &botVerifierFakeResolver{
			names: map[string][]string{"203.0.113.7": {"crawl.googlebot.com."}},
			addrs: map[string][]net.IPAddr{"crawl.googlebot.com": {{IP: net.ParseIP("66.249.66.1")}}},
		}
		verifier := NewBotVerifier(NewMemoryStore(t.Context()), WithResolver(resolver))
		require.False(t, verifier.verifyBot(newBotVerifierRequest(googlebot), netip.MustParseAddr("203.0.113.7"), start))
	})

	t.Run("google cloud vm host is rejected", func(t *testing.T) {
		resolver := &botVerifierFakeResolver{
			names: map[string][]string{"34.1.2.3": {"3.2.1.34.bc.googleusercontent.com."}},
			addrs: map[string][]net.IPAddr{"3.2.1.34.bc.googleusercontent.com": {{IP: net.ParseIP("34.1.2.3")}}},
		}
		verifier := NewBotVerifier(NewMemoryStore(t.Context()), WithResolver(resolver))
		require.False(t, verifier.verifyBot(newBotVerifierRequest(googlebot), netip.MustParseAddr("34.1.2.3"), start))
	})

	t.Run("missing reverse dns is cached", func(t *testing.T) {
		resolver := new(botVerifierFakeResolver)
		verifier := NewBotVerifier(NewMemoryStore(t.Context()), WithResolver(resolver))
		addr := netip.MustParseAddr("203.0.113.7")
		require.False(t, verifier.verifyBot(newBotVerifierRequest(googlebot), addr, start))
		require.False(t, verifier.verifyBot(newBotVerifierRequest(googlebot), addr, start.Add(time.Hour)))
		require.Equal(t, 1, resolver.lookups)
	})

	t.Run("failed dns lookup is cached briefly", func(t *testing.T) {
		resolver := &botVerifierFakeResolver{err: errors.New("i/o timeout")}
		verifier := NewBotVerifier(NewMemoryStore(t.Context()), WithResolver(resolver))
		addr := netip.MustParseAddr("203.0.113.7")
		require.False(t, verifier.verifyBot(newBotVerifierRequest(googlebot), addr, start))
		require.False(t, verifier.verifyBot(newBotVerifierRequest(googlebot), addr, start.Add(time.Minute)))
		require.Equal(t, 1, resolver.lookups)
		require.False(t, verifier.verifyBot(newBotVerifierRequest(googlebot), addr, start.Add(botErrorTTL)))
		require.Equal(t, 2, resolver.lookups)
	})

	t.Run("regular browsers skip dns", func(t *testing.T) {
		resolver := new(botVerifierFakeResolver)
		verifier := NewBotVerifier(NewMemoryStore(t.Context()), WithResolver(resolver))
		require.False(t, verifier.verifyBot(newBotVerifierRequest("Mozilla/5.0 Safari/605.1.15"), netip.MustParseAddr("203.0.113.7"), start))
		require.Zero(t, resolver.lookups)
	})

	t.Run("verified bot header is ignored without cloudflare proxy", func(t *testing.T) {
		resolver := new(botVerifierFakeResolver)
		verifier := NewBotVerifier(NewMemoryStore(t.Context()), WithResolver(resolver))
		r := newBotVerifierRequest(googlebot)
		r.Header.Set("X-Verified-Bot", "1")
		require.False(t, verifier.verifyBot(r, netip.MustParseAddr("203.0.113.7"), start))
	})
}

func newBotVerifierRequest(userAgent string) *http.Request {
	r := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/products", nil)
	r.Header.Set("User-Agent", userAgent)
	return r
}
