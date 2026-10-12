package security

import (
	"net/http"
	"slices"
	"strconv"
	"time"
)

type Headers struct {
	values []header
}

type header struct {
	name, value string
}

type HeadersOption func(*Headers)

func NewHeaders(options ...HeadersOption) *Headers {
	h := &Headers{
		values: []header{
			{"X-Content-Type-Options", "nosniff"},
			{"X-Frame-Options", "DENY"},
			{"Referrer-Policy", "strict-origin-when-cross-origin"},
			{"Cross-Origin-Opener-Policy", "same-origin"},
			{"Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=(), browsing-topics=()"},
		},
	}
	for _, option := range options {
		option(h)
	}
	return h
}

func (h *Headers) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			responseHeaders := w.Header()
			for _, value := range h.values {
				responseHeaders.Set(value.name, value.value)
			}
			next.ServeHTTP(w, r)
		},
	)
}

func (h *Headers) set(name, value string) {
	index := slices.IndexFunc(h.values, func(existing header) bool {
		return http.CanonicalHeaderKey(existing.name) == http.CanonicalHeaderKey(name)
	})
	switch {
	case index >= 0 && value == "":
		h.values = slices.Delete(h.values, index, index+1)
	case index >= 0:
		h.values[index].value = value
	case value != "":
		h.values = append(h.values, header{name, value})
	}
}

func WithHSTS(maxAge time.Duration) HeadersOption {
	return func(h *Headers) {
		h.set("Strict-Transport-Security", "max-age="+strconv.FormatInt(int64(maxAge.Seconds()), 10))
	}
}

func WithCSP(policy string) HeadersOption {
	return func(h *Headers) {
		h.set("Content-Security-Policy", policy)
	}
}

func WithCSPReportOnly(policy string) HeadersOption {
	return func(h *Headers) {
		h.set("Content-Security-Policy-Report-Only", policy)
	}
}

func WithPermissionsPolicy(policy string) HeadersOption {
	return func(h *Headers) {
		h.set("Permissions-Policy", policy)
	}
}

func WithHeader(name, value string) HeadersOption {
	return func(h *Headers) {
		h.set(name, value)
	}
}
