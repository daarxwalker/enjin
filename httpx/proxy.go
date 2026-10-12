package httpx

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"slices"
	"strings"
)

type Proxy struct {
	trusted     []netip.Prefix
	hosts       map[string]struct{}
	defaultHost string
	exemptPaths []string
	cloudflare  bool
}

type ProxyOption func(*Proxy)

func NewProxy(options ...ProxyOption) *Proxy {
	p := new(Proxy)
	for _, option := range options {
		option(p)
	}
	return p
}

func (p *Proxy) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			r.Header.Del("True-Client-IP")
			r.Header.Del("X-Real-IP")
			r.Header.Del("Forwarded")
			if p.defaultHost != "" {
				r = r.WithContext(context.WithValue(r.Context(), defaultHostKey{}, p.defaultHost))
			}
			peer, ok := parseAddr(r.RemoteAddr)
			viaCloudflare := false
			if ok && p.isTrusted(peer) {
				r = r.WithContext(context.WithValue(r.Context(), forwardedKey{}, true))
				if client, hop, found := p.clientFromForwardedFor(r.Header.Values("X-Forwarded-For"), peer); found {
					r.RemoteAddr = client.String()
					viaCloudflare = p.cloudflare && isCloudflareAddr(hop)
				}
				if proto := r.Header.Get("X-Forwarded-Proto"); proto != "http" && proto != "https" {
					r.Header.Del("X-Forwarded-Proto")
				}
			} else {
				r.Header.Del("X-Forwarded-For")
				r.Header.Del("X-Forwarded-Host")
				r.Header.Del("X-Forwarded-Proto")
				r.Header.Del("X-Forwarded-Prefix")
			}
			if viaCloudflare {
				r = r.WithContext(context.WithValue(r.Context(), cloudflareKey{}, true))
			} else {
				r.Header.Del("CF-Connecting-IP")
				r.Header.Del("X-Verified-Bot")
			}
			if !p.isHostAllowed(r) {
				http.Error(w, http.StatusText(http.StatusMisdirectedRequest), http.StatusMisdirectedRequest)
				return
			}
			next.ServeHTTP(w, r)
		},
	)
}

func (p *Proxy) isTrusted(addr netip.Addr) bool {
	for _, prefix := range p.trusted {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

func (p *Proxy) clientFromForwardedFor(values []string, peer netip.Addr) (netip.Addr, netip.Addr, bool) {
	var entries []string
	for _, value := range values {
		entries = append(entries, strings.Split(value, ",")...)
	}
	hop := peer
	for _, entry := range slices.Backward(entries) {
		addr, ok := parseAddr(strings.TrimSpace(entry))
		if !ok {
			return netip.Addr{}, netip.Addr{}, false
		}
		if !p.isTrusted(addr) {
			return addr, hop, true
		}
		hop = addr
	}
	return netip.Addr{}, netip.Addr{}, false
}

func (p *Proxy) isHostAllowed(r *http.Request) bool {
	if len(p.hosts) == 0 {
		return true
	}
	for _, path := range p.exemptPaths {
		if r.URL.Path == path || strings.HasPrefix(r.URL.Path, path+"/") {
			return true
		}
	}
	host := HostFrom(r)
	if name, _, err := net.SplitHostPort(host); err == nil {
		host = name
	}
	_, ok := p.hosts[strings.ToLower(host)]
	return ok
}

func ParsePrefixes(values ...string) ([]netip.Prefix, error) {
	prefixes := make([]netip.Prefix, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if !strings.Contains(value, "/") {
			addr, err := netip.ParseAddr(value)
			if err != nil {
				return nil, fmt.Errorf("parse prefix %q failed: %w", value, err)
			}
			addr = addr.Unmap()
			prefixes = append(prefixes, netip.PrefixFrom(addr, addr.BitLen()))
			continue
		}
		prefix, err := netip.ParsePrefix(value)
		if err != nil {
			return nil, fmt.Errorf("parse prefix %q failed: %w", value, err)
		}
		prefixes = append(prefixes, prefix.Masked())
	}
	return prefixes, nil
}

func WithTrustedProxies(prefixes ...netip.Prefix) ProxyOption {
	return func(p *Proxy) {
		p.trusted = append(p.trusted, prefixes...)
	}
}

func WithAllowedHosts(hosts ...string) ProxyOption {
	return func(p *Proxy) {
		if p.hosts == nil {
			p.hosts = make(map[string]struct{}, len(hosts))
		}
		for _, host := range hosts {
			p.hosts[strings.ToLower(host)] = struct{}{}
		}
	}
}

func WithDefaultHost(host string) ProxyOption {
	return func(p *Proxy) {
		p.defaultHost = host
	}
}

func WithCloudflare(enabled bool) ProxyOption {
	return func(p *Proxy) {
		p.cloudflare = enabled
		if enabled {
			p.trusted = append(p.trusted, cloudflarePrefixes...)
		}
	}
}

func WithHostExemptPaths(paths ...string) ProxyOption {
	return func(p *Proxy) {
		p.exemptPaths = append(p.exemptPaths, paths...)
	}
}
