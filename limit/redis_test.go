package limit_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/daarxwalker/enjin/infra/cache"
	"github.com/daarxwalker/enjin/internal/testredis"
	"github.com/daarxwalker/enjin/limit"

	"github.com/stretchr/testify/require"
)

func TestRedisLimiter(t *testing.T) {
	ctx := context.Background()
	rdb := testredis.New(t)
	counter := cache.NewClient(cache.NewRedisDriver(rdb))

	t.Run(
		"verify rejects over the limit", func(t *testing.T) {
			limiter := limit.NewLimiter(counter, "verify", 2, limit.WithTTL(time.Hour))
			for range 2 {
				require.Equal(t, http.StatusNoContent, serve(limiter.Verify, "192.0.2.1:1000", nil).Code)
			}
			rejected := serve(limiter.Verify, "192.0.2.1:2000", nil)
			require.Equal(t, http.StatusTooManyRequests, rejected.Code)
			require.Equal(t, "3600", rejected.Header().Get("Retry-After"))
			require.Equal(t, http.StatusNoContent, serve(limiter.Verify, "192.0.2.2:1000", nil).Code)
			value, err := rdb.Get(ctx, "limit:verify:192.0.2.1").Result()
			require.NoError(t, err)
			require.Equal(t, "2", value)
			require.Equal(t, time.Hour, rdb.PTTL(ctx, "limit:verify:192.0.2.1").Val())
		},
	)

	t.Run(
		"verify is atomic", func(t *testing.T) {
			limiter := limit.NewLimiter(counter, "atomic", 5, limit.WithTTL(time.Hour))
			var allowed atomic.Int32
			var wg sync.WaitGroup
			for range 30 {
				wg.Go(func() {
					if serve(limiter.Verify, "192.0.2.3:1000", nil).Code == http.StatusNoContent {
						allowed.Add(1)
					}
				})
			}
			wg.Wait()
			require.Equal(t, int32(5), allowed.Load())
		},
	)

	t.Run(
		"load consume revert", func(t *testing.T) {
			limiter := limit.NewLimiter(counter, "consume", 1, limit.WithTTL(time.Hour))
			var steps []string
			step := func(ctx context.Context) {
				current := limiter.Current(ctx)
				ok, err := limiter.Consume(ctx)
				require.NoError(t, err)
				if ok {
					steps = append(steps, "consumed")
					return
				}
				steps = append(steps, "blocked")
				if current == 1 {
					require.NoError(t, limiter.Revert(ctx))
					steps = append(steps, "reverted")
				}
			}
			serve(limiter.Load, "192.0.2.4:1000", step)
			serve(limiter.Load, "192.0.2.4:1000", step)
			serve(limiter.Load, "192.0.2.4:1000", step)
			require.Equal(t, []string{"consumed", "blocked", "reverted", "consumed"}, steps)
		},
	)

	t.Run(
		"reset in", func(t *testing.T) {
			limiter := limit.NewLimiter(counter, "reset", 3, limit.WithTTL(time.Hour))
			serve(limiter.Verify, "192.0.2.5:1000", nil)
			var resetIn time.Duration
			serve(limiter.Load, "192.0.2.5:1000", func(ctx context.Context) {
				resetIn = limiter.ResetIn(ctx)
			})
			require.Equal(t, time.Hour, resetIn)
		},
	)
}

func serve(middleware func(http.Handler) http.Handler, remoteAddr string, handler func(ctx context.Context)) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.RemoteAddr = remoteAddr
	w := httptest.NewRecorder()
	middleware(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				if handler != nil {
					handler(r.Context())
				}
				w.WriteHeader(http.StatusNoContent)
			},
		),
	).ServeHTTP(w, r)
	return w
}
