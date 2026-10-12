package httpx

import "net/url"

type ctxKey struct{}

type forwardedKey struct{}

type cloudflareKey struct{}

type defaultHostKey struct{}

var key = ctxKey{}

type ContextValue struct {
	Query      url.Values
	URL        string
	PathPrefix string
	Host       string
}
