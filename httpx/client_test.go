package httpx_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/daarxwalker/enjin/httpx"
)

func TestClient(t *testing.T) {
	t.Run("error response", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(strings.Repeat("x", 1<<20)))
		}))
		defer srv.Close()

		req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
		if err != nil {
			t.Fatalf("NewRequest() error = %v", err)
		}
		res, err := httpx.NewClient().Do(req)
		if res != nil {
			defer func() { _ = res.Body.Close() }()
		}
		if err == nil {
			t.Fatal("Do() error = nil, want error")
		}
		if got := len(err.Error()); got > 65<<10 {
			t.Errorf("error length = %d, want body limited to 64 KiB", got)
		}
		if res == nil {
			t.Fatal("Do() response = nil, want response")
		}
		if res.StatusCode != http.StatusServiceUnavailable || res.Header.Get("Retry-After") != "1" {
			t.Errorf("response = %d %v", res.StatusCode, res.Header)
		}
		if res.Body != http.NoBody {
			t.Errorf("Body = %T, want http.NoBody", res.Body)
		}
	})
}
