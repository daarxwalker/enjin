package httpx

import (
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"unicode"
)

func CreateURL(r *http.Request, path string) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	if proto := forwardedHeader(r, "X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}

	host := HostFrom(r)
	if host == "" {
		if path == "" {
			return "/"
		}
		return path
	}

	u := url.URL{
		Scheme: scheme,
		Host:   host,
		Path:   path,
	}

	return u.String()
}

func HostFrom(r *http.Request) string {
	if h := forwardedHeader(r, "X-Forwarded-Host"); h != "" {
		return h
	}
	if r.Host != "" {
		return r.Host
	}
	return defaultHost(r)
}

func defaultHost(r *http.Request) string {
	val := r.Context().Value(defaultHostKey{})
	if val == nil {
		return ""
	}
	host, ok := val.(string)
	if !ok {
		return ""
	}
	return host
}

func forwardedHeader(r *http.Request, name string) string {
	val := r.Context().Value(forwardedKey{})
	if val == nil {
		return ""
	}
	forwarded, ok := val.(bool)
	if !ok || !forwarded {
		return ""
	}
	return r.Header.Get(name)
}

func IsLocalPath(value string) bool {
	if !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") {
		return false
	}
	return !strings.ContainsFunc(value, func(char rune) bool {
		return char == '\\' || unicode.IsControl(char)
	})
}

func ClientIP(r *http.Request) string {
	if addr, ok := parseAddr(r.RemoteAddr); ok {
		return addr.String()
	}
	return r.RemoteAddr
}

func parseAddr(value string) (netip.Addr, bool) {
	if addrPort, err := netip.ParseAddrPort(value); err == nil {
		return addrPort.Addr().Unmap(), true
	}
	if addr, err := netip.ParseAddr(value); err == nil {
		return addr.Unmap(), true
	}
	return netip.Addr{}, false
}
