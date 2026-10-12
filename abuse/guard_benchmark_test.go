package abuse

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

func BenchmarkGuard(b *testing.B) {
	store := NewMemoryStore(b.Context())
	guard := NewGuard(store, WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))))
	policy := Policy{
		Name: "product-page",
		Rules: []Rule{
			CreateRateLimitRule(CountedByIP, math.MaxInt32, time.Minute, VerdictDelay),
			CreateDistinctURLParamRule(CountedByClientID, "slug", 150, time.Hour, VerdictDelay),
			CreateDistinctURLParamRule(CountedByIP, "slug", 300, time.Hour, VerdictBlock),
			CreateDistinctURLParamRule(CountedByUserID, "slug", 200, 24*time.Hour, VerdictDelay),
			CreateUserAgentRule(VerdictBlock),
		},
	}
	handler := guard.Protect(policy)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	requests := make([]*http.Request, 1024)
	for i := range requests {
		r := httptest.NewRequestWithContext(context.Background(), http.MethodGet, fmt.Sprintf("/products/p-%d", i%50), nil)
		r.RemoteAddr = fmt.Sprintf("198.51.%d.%d:1234", i/256, i%256)
		r.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_6) Safari/605.1.15")
		routeContext := chi.NewRouteContext()
		routeContext.URLParams.Add("slug", fmt.Sprintf("p-%d", i%50))
		requests[i] = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, routeContext))
	}
	w := httptest.NewRecorder()
	b.Run("many ips", func(b *testing.B) {
		b.ReportAllocs()
		i := 0
		for b.Loop() {
			handler.ServeHTTP(w, requests[i%len(requests)])
			i++
		}
	})
	b.Run("one ip", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			handler.ServeHTTP(w, requests[0])
		}
	})
	b.Run("parallel", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			recorder := httptest.NewRecorder()
			i := 0
			for pb.Next() {
				handler.ServeHTTP(recorder, requests[i%len(requests)])
				i++
			}
		})
	})
}
