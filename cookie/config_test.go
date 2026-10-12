package cookie_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/daarxwalker/enjin/cookie"
)

func written(t *testing.T, write func(w http.ResponseWriter)) *http.Cookie {
	t.Helper()
	rec := httptest.NewRecorder()
	write(rec)
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %d, want 1", len(cookies))
	}
	return cookies[0]
}

func TestConfig(t *testing.T) {
	t.Run("secure by default", func(t *testing.T) {
		cfg := cookie.NewConfig(cookie.WithName("x"))
		c := written(t, func(w http.ResponseWriter) { cfg.Attach(w, "v") })
		if !c.Secure || !c.HttpOnly {
			t.Fatalf("Secure = %v, HttpOnly = %v, want both true", c.Secure, c.HttpOnly)
		}
		if c.SameSite != http.SameSiteLaxMode || c.Path != "/" {
			t.Fatalf("SameSite = %v, Path = %q, want Lax and /", c.SameSite, c.Path)
		}
	})

	t.Run("apply disables flags", func(t *testing.T) {
		cfg := cookie.NewConfig(cookie.WithName("x"))
		cfg.Apply(cookie.WithSecure(false), cookie.WithHTTPOnly(false))
		c := written(t, func(w http.ResponseWriter) { cfg.Attach(w, "v") })
		if c.Secure || c.HttpOnly {
			t.Fatalf("Secure = %v, HttpOnly = %v, want both false", c.Secure, c.HttpOnly)
		}
	})

	t.Run("apply keeps earlier values", func(t *testing.T) {
		cfg := cookie.NewConfig(
			cookie.WithName("x"),
			cookie.WithHTTPOnly(false),
			cookie.WithSameSite(http.SameSiteStrictMode),
			cookie.WithTTL(time.Hour),
		)
		cfg.Apply(cookie.WithName("y"))
		c := written(t, func(w http.ResponseWriter) { cfg.Attach(w, "v") })
		if c.Name != "y" || c.HttpOnly || c.SameSite != http.SameSiteStrictMode || c.MaxAge != 3600 {
			t.Fatalf("got %+v, want name y, HttpOnly false, Strict, MaxAge 3600", c)
		}
	})

	t.Run("domain on attach and destroy", func(t *testing.T) {
		cfg := cookie.NewConfig(cookie.WithName("x"), cookie.WithDomain("example.com"), cookie.WithPath("/app"))
		attached := written(t, func(w http.ResponseWriter) { cfg.Attach(w, "v") })
		destroyed := written(t, func(w http.ResponseWriter) { cfg.Destroy(w) })
		for _, c := range []*http.Cookie{attached, destroyed} {
			if c.Domain != "example.com" || c.Path != "/app" {
				t.Fatalf("Domain = %q, Path = %q, want example.com and /app", c.Domain, c.Path)
			}
		}
		if destroyed.MaxAge >= 0 || destroyed.Value != "" {
			t.Fatalf("destroy MaxAge = %d, Value = %q, want negative and empty", destroyed.MaxAge, destroyed.Value)
		}
	})
}
