package openapi

import (
	openapigo "github.com/swaggest/openapi-go"
	"github.com/swaggest/openapi-go/openapi3"
)

type Option func(*openapi3.Reflector)

func WithURL(url string) Option {
	return func(r *openapi3.Reflector) {
		r.Spec.Servers = []openapi3.Server{{URL: url}}
	}
}

func WithTitle(title string) Option {
	return func(r *openapi3.Reflector) {
		r.Spec.Info.WithTitle(title)
	}
}

func WithDescription(description string) Option {
	return func(r *openapi3.Reflector) {
		r.Spec.Info.WithDescription(description)
	}
}

func WithVersion(version string) Option {
	return func(r *openapi3.Reflector) {
		r.Spec.Info.WithVersion(version)
	}
}

func WithBearer(description string) Option {
	return func(r *openapi3.Reflector) {
		r.Spec.SetHTTPBearerTokenSecurity("bearer", "", description)
	}
}

func WithCookieSession(cookieName string, description string) Option {
	return func(r *openapi3.Reflector) {
		r.Spec.SetAPIKeySecurity("session", cookieName, openapigo.InCookie, description)
	}
}
