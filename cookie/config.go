package cookie

import (
	"net/http"
	"time"
)

type Config struct {
	Name     string
	Path     string
	Domain   string
	SameSite http.SameSite
	TTL      time.Duration
	Secure   bool
	HTTPOnly bool
}

func NewConfig(options ...Option) *Config {
	cfg := &Config{
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		Secure:   true,
		HTTPOnly: true,
	}
	cfg.Apply(options...)
	return cfg
}

func (c *Config) Apply(options ...Option) {
	for _, opt := range options {
		opt(c)
	}
}

func (c *Config) Attach(w http.ResponseWriter, v string) {
	http.SetCookie(w, c.cookie(v, int(c.TTL.Seconds())))
}

func (c *Config) Destroy(w http.ResponseWriter) {
	http.SetCookie(w, c.cookie("", -1))
}

func (c *Config) cookie(value string, maxAge int) *http.Cookie {
	return &http.Cookie{ //nolint:gosec // flags come from Config, secure by default
		Name:     c.Name,
		Value:    value,
		Path:     c.Path,
		Domain:   c.Domain,
		MaxAge:   maxAge,
		HttpOnly: c.HTTPOnly,
		SameSite: c.SameSite,
		Secure:   c.Secure,
	}
}

type Option func(*Config)

func WithName(name string) Option {
	return func(cfg *Config) {
		cfg.Name = name
	}
}

func WithPath(path string) Option {
	return func(cfg *Config) {
		cfg.Path = path
	}
}

func WithDomain(domain string) Option {
	return func(cfg *Config) {
		cfg.Domain = domain
	}
}

func WithSameSite(sameSite http.SameSite) Option {
	return func(cfg *Config) {
		cfg.SameSite = sameSite
	}
}

func WithTTL(ttl time.Duration) Option {
	return func(cfg *Config) {
		cfg.TTL = ttl
	}
}

func WithSecure(secure bool) Option {
	return func(cfg *Config) {
		cfg.Secure = secure
	}
}

func WithHTTPOnly(httpOnly bool) Option {
	return func(cfg *Config) {
		cfg.HTTPOnly = httpOnly
	}
}
