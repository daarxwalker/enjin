package csrftest

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/daarxwalker/enjin/web/csrf"

	"github.com/stretchr/testify/require"
)

const concurrency = 20

// RunDriverSuite checks that a Driver behaves like the Redis one. Every case
// uses its own keys, so the store may be shared with other tests.
func RunDriverSuite(t *testing.T, driver csrf.Driver) {
	t.Run("take returns a token once", func(t *testing.T) {
		checkTakeOnce(t, driver)
	})
	t.Run("unknown token", func(t *testing.T) {
		checkUnknownToken(t, driver)
	})
	t.Run("expired token is removed", func(t *testing.T) {
		checkExpiredToken(t, driver)
	})
	t.Run("limit keeps the newest tokens", func(t *testing.T) {
		checkLimit(t, driver)
	})
	t.Run("limit below one keeps one token", func(t *testing.T) {
		checkLimitBelowOne(t, driver)
	})
	t.Run("keys are independent", func(t *testing.T) {
		checkIndependentKeys(t, driver)
	})
	t.Run("concurrent take", func(t *testing.T) {
		checkConcurrentTake(t, driver)
	})
	t.Run("concurrent add keeps the limit", func(t *testing.T) {
		checkConcurrentAdd(t, driver)
	})
}

func checkTakeOnce(t *testing.T, driver csrf.Driver) {
	key := "suite:once"
	now := time.Now()
	require.NoError(t, driver.Add(t.Context(), key, "token", now.Add(time.Hour), 5))

	requireTake(t, driver, key, "token", now, true)
	requireTake(t, driver, key, "token", now, false)
}

func checkUnknownToken(t *testing.T, driver csrf.Driver) {
	key := "suite:unknown"
	now := time.Now()
	requireTake(t, driver, key, "token", now, false)

	require.NoError(t, driver.Add(t.Context(), key, "token", now.Add(time.Hour), 5))
	requireTake(t, driver, key, "other", now, false)
	requireTake(t, driver, key, "token", now, true)
}

func checkExpiredToken(t *testing.T, driver csrf.Driver) {
	key := "suite:expired"
	now := time.Now()
	require.NoError(t, driver.Add(t.Context(), key, "token", now.Add(time.Hour), 5))

	requireTake(t, driver, key, "token", now.Add(2*time.Hour), false)
	requireTake(t, driver, key, "token", now, false)
}

func checkLimit(t *testing.T, driver csrf.Driver) {
	key := "suite:limit"
	now := time.Now()
	for i, token := range []string{"oldest", "older", "newest"} {
		require.NoError(t, driver.Add(t.Context(), key, token, now.Add(time.Duration(i+1)*time.Hour), 2))
	}

	requireTake(t, driver, key, "oldest", now, false)
	requireTake(t, driver, key, "older", now, true)
	requireTake(t, driver, key, "newest", now, true)
}

func checkLimitBelowOne(t *testing.T, driver csrf.Driver) {
	key := "suite:limit-zero"
	now := time.Now()
	require.NoError(t, driver.Add(t.Context(), key, "first", now.Add(time.Hour), 0))
	require.NoError(t, driver.Add(t.Context(), key, "second", now.Add(2*time.Hour), 0))

	requireTake(t, driver, key, "first", now, false)
	requireTake(t, driver, key, "second", now, true)
}

func checkIndependentKeys(t *testing.T, driver csrf.Driver) {
	now := time.Now()
	require.NoError(t, driver.Add(t.Context(), "suite:key-a", "token", now.Add(time.Hour), 1))
	require.NoError(t, driver.Add(t.Context(), "suite:key-b", "token", now.Add(time.Hour), 1))

	requireTake(t, driver, "suite:key-a", "token", now, true)
	requireTake(t, driver, "suite:key-b", "token", now, true)
}

func checkConcurrentTake(t *testing.T, driver csrf.Driver) {
	ctx := t.Context()
	key := "suite:take-concurrent"
	now := time.Now()
	require.NoError(t, driver.Add(ctx, key, "token", now.Add(time.Hour), 5))

	var (
		taken    atomic.Int32
		failures atomic.Int32
		wg       sync.WaitGroup
	)
	for range concurrency {
		wg.Go(func() {
			ok, err := driver.Take(ctx, key, "token", now)
			if err != nil {
				failures.Add(1)
			}
			if ok {
				taken.Add(1)
			}
		})
	}
	wg.Wait()
	require.Zero(t, failures.Load())
	require.Equal(t, int32(1), taken.Load())
}

func checkConcurrentAdd(t *testing.T, driver csrf.Driver) {
	ctx := t.Context()
	key := "suite:add-concurrent"
	limit := 5
	now := time.Now()

	var (
		failures atomic.Int32
		wg       sync.WaitGroup
	)
	for i := range concurrency {
		wg.Go(func() {
			expiresAt := now.Add(time.Hour + time.Duration(i)*time.Second)
			if err := driver.Add(ctx, key, createToken(i), expiresAt, limit); err != nil {
				failures.Add(1)
			}
		})
	}
	wg.Wait()
	require.Zero(t, failures.Load())

	for i := range concurrency {
		requireTake(t, driver, key, createToken(i), now, i >= concurrency-limit)
	}
}

func createToken(i int) string {
	return fmt.Sprintf("token-%02d", i)
}

func requireTake(t *testing.T, driver csrf.Driver, key, token string, now time.Time, expected bool) {
	t.Helper()
	taken, err := driver.Take(t.Context(), key, token, now)
	require.NoError(t, err)
	require.Equal(t, expected, taken, "%s %s", key, token)
}
