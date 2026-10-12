package csrf

import (
	"context"
	"crypto/subtle"
	"net/http"
	"time"

	"github.com/daarxwalker/enjin/cookie"
	"github.com/daarxwalker/enjin/token"
)

const (
	Header        = "X-CSRF-Token"
	DefaultCookie = "X-CSRF"
)

var DefaultTTL = 24 * time.Hour

type CookieManager struct {
	cookie *cookie.Config
}

func NewCookieManager(options ...cookie.Option) *CookieManager {
	return &CookieManager{
		cookie: cookie.NewConfig(
			append(
				[]cookie.Option{
					cookie.WithName(DefaultCookie),
					cookie.WithTTL(DefaultTTL),
					cookie.WithHTTPOnly(false),
				},
				options...,
			)...,
		),
	}
}

func (m *CookieManager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var value string
			if c, err := r.Cookie(m.cookie.Name); err == nil {
				value = c.Value
			}
			if !token.Valid(value) {
				value = token.New()
			}
			m.cookie.Attach(w, value)
			next.ServeHTTP(w, r)
		},
	)
}

func (m *CookieManager) Verify(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			cv := from(ctx)
			if cv == nil {
				cv = &ctxVal{}
				r = r.WithContext(context.WithValue(ctx, ctxKey, cv))
			}
			cv.rejected = !m.matches(r)
			next.ServeHTTP(w, r)
		},
	)
}

func (m *CookieManager) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if !m.matches(r) {
				forbidden.ServeHTTP(w, r)
				return
			}
			next.ServeHTTP(w, r)
		},
	)
}

func (m *CookieManager) matches(r *http.Request) bool {
	c, err := r.Cookie(m.cookie.Name)
	if err != nil {
		return false
	}
	header := r.Header.Get(Header)
	if len(c.Value) == 0 || len(header) == 0 {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(c.Value), []byte(header)) == 1
}
