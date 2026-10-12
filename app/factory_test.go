package app

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/daarxwalker/enjin/httpx"

	"github.com/go-chi/chi/v5"
)

func TestURL(t *testing.T) {
	t.Run("builds the path of a named route", func(t *testing.T) {
		got, err := serveURL(t, "product", map[string]any{"slug": "cream", "page": 2})
		if err != nil {
			t.Fatalf("URL() error = %v", err)
		}
		if got != "/products/cream?page=2" {
			t.Fatalf("URL() = %q, want /products/cream?page=2", got)
		}
	})

	t.Run("unknown route", func(t *testing.T) {
		if _, err := serveURL(t, "missing", nil); !errors.Is(err, ErrRouteNotFound) {
			t.Fatalf("URL() error = %v, want ErrRouteNotFound", err)
		}
	})

	t.Run("without router", func(t *testing.T) {
		if _, err := URL(context.Background(), "product"); !errors.Is(err, ErrNoRouter) {
			t.Fatalf("URL() error = %v, want ErrNoRouter", err)
		}
	})

	t.Run("must url panics on error", func(t *testing.T) {
		defer func() {
			err, ok := recover().(error)
			if !ok || !errors.Is(err, ErrNoRouter) {
				t.Fatalf("recovered = %v, want ErrNoRouter", err)
			}
		}()
		MustURL(context.Background(), "product")
	})
}

func serveURL(t *testing.T, name string, params map[string]any) (string, error) {
	t.Helper()
	var (
		got string
		err error
	)
	mux := chi.NewRouter()
	r := NewRouter(mux, NewDefaultHandler())
	r.Get("product", "/products/{slug}", func(w http.ResponseWriter, req *http.Request) error {
		got, err = URL(req.Context(), name, params)
		w.WriteHeader(http.StatusNoContent)
		return nil
	})
	httpx.Middleware(mux).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/products/x", nil))
	return got, err
}
