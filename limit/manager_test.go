package limit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"
)

type fakeEntry struct {
	value int
	ttl   time.Duration
}

type fakeCounter struct {
	mu      sync.Mutex
	entries map[string]fakeEntry
}

func newFakeCounter() *fakeCounter {
	return &fakeCounter{entries: make(map[string]fakeEntry)}
}

func (c *fakeCounter) GetString(_ context.Context, key string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[key]
	if !ok {
		return "", nil
	}
	return strconv.Itoa(e.value), nil
}

func (c *fakeCounter) TTL(_ context.Context, key string) (time.Duration, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.entries[key].ttl, nil
}

func (c *fakeCounter) Increment(_ context.Context, key string, delta int, ttl time.Duration) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[key]
	if delta < 0 && !ok {
		return 0, nil
	}
	if !ok {
		e.ttl = ttl
	}
	e.value += delta
	c.entries[key] = e
	return e.value, nil
}

func serve(middleware func(http.Handler) http.Handler, remoteAddr string, handler func(ctx context.Context)) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.RemoteAddr = remoteAddr
	w := httptest.NewRecorder()
	middleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		handler(r.Context())
	})).ServeHTTP(w, r)
	return w
}

func TestManager(t *testing.T) {
	key := "limit:analyze:192.0.2.1"

	t.Run("load consume revert", func(t *testing.T) {
		counter := newFakeCounter()
		m := NewLimiter(counter, "analyze", 1)

		serve(m.Load, "192.0.2.1:1234", func(ctx context.Context) {
			if got := m.Current(ctx); got != 0 {
				t.Errorf("Current = %d, want 0", got)
			}
		})
		if len(counter.entries) != 0 {
			t.Fatalf("Load wrote the counter: %+v", counter.entries)
		}

		serve(m.Load, "192.0.2.1:1234", func(ctx context.Context) {
			if allowed, err := m.Consume(ctx); err != nil || !allowed {
				t.Fatalf("Consume = %v, %v, want allowed", allowed, err)
			}
			if allowed, err := m.Consume(ctx); err != nil || allowed {
				t.Fatalf("Consume over max = %v, %v, want rejected", allowed, err)
			}
		})
		if got := counter.entries[key]; got != (fakeEntry{value: 1, ttl: 24 * time.Hour}) {
			t.Fatalf("after consume = %+v, want 1 for 24h", got)
		}

		serve(m.Load, "192.0.2.1:1234", func(ctx context.Context) {
			if got := m.Current(ctx); got != 1 {
				t.Errorf("Current = %d, want 1", got)
			}
			if got := m.ResetIn(ctx); got != 24*time.Hour {
				t.Errorf("ResetIn = %v, want 24h", got)
			}
			if err := m.Revert(ctx); err != nil {
				t.Fatal(err)
			}
		})
		if got := counter.entries[key]; got.value != 0 {
			t.Fatalf("after revert = %+v, want 0", got)
		}
	})

	t.Run("consume without load is rejected", func(t *testing.T) {
		m := NewLimiter(newFakeCounter(), "analyze", 1)
		if allowed, err := m.Consume(context.Background()); err == nil || allowed {
			t.Fatalf("Consume = %v, %v, want ErrNotLoaded", allowed, err)
		}
	})

	t.Run("concurrent consume allows max", func(t *testing.T) {
		counter := newFakeCounter()
		m := NewLimiter(counter, "analyze", 3)
		ctx := m.load(context.Background(), "192.0.2.1")

		var (
			wg      sync.WaitGroup
			mu      sync.Mutex
			allowed int
		)
		for range 50 {
			wg.Go(func() {
				ok, err := m.Consume(ctx)
				if err != nil {
					t.Error(err)
				}
				if ok {
					mu.Lock()
					allowed++
					mu.Unlock()
				}
			})
		}
		wg.Wait()
		if allowed != 3 || counter.entries[key].value != 3 {
			t.Fatalf("allowed = %d, counter = %d, want 3 and 3", allowed, counter.entries[key].value)
		}
	})

	t.Run("verify rejects over max", func(t *testing.T) {
		counter := newFakeCounter()
		m := NewLimiter(counter, "analyze", 1, WithTTL(time.Hour))

		calls := 0
		handler := func(ctx context.Context) {
			calls++
			if got := m.Current(ctx); got != 1 {
				t.Errorf("Current = %d, want 1", got)
			}
		}
		if w := serve(m.Verify, "192.0.2.1:1234", handler); w.Code != http.StatusOK {
			t.Fatalf("first status = %d, want 200", w.Code)
		}
		w := serve(m.Verify, "192.0.2.1:5678", handler)
		if w.Code != http.StatusTooManyRequests {
			t.Fatalf("second status = %d, want 429", w.Code)
		}
		if got := w.Header().Get("Retry-After"); got != "3600" {
			t.Errorf("Retry-After = %q, want 3600", got)
		}
		if calls != 1 {
			t.Fatalf("handler called %d times, want 1", calls)
		}
		if got := counter.entries[key]; got != (fakeEntry{value: 1, ttl: time.Hour}) || len(counter.entries) != 1 {
			t.Fatalf("entries = %+v, want one counter %s at 1", counter.entries, key)
		}
	})
}
