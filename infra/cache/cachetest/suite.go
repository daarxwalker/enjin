package cachetest

import (
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/daarxwalker/enjin/infra/cache"

	"github.com/stretchr/testify/require"
)

const (
	longTTL     = time.Minute
	shortTTL    = time.Second
	concurrency = 20
)

// ttlTolerance absorbs stores that count TTL in real time, such as Postgres,
// where a few milliseconds pass between Set and TTL.
const ttlTolerance = time.Second

type Subject struct {
	Driver cache.Driver
	// AdvanceClock moves the time the driver sees forward. A store with its
	// own clock (a database server) implements it by sleeping.
	AdvanceClock func(time.Duration)
}

// RunDriverSuite checks that a Driver behaves like the Redis one. Every case
// uses its own keys, so the store may be shared with other tests.
func RunDriverSuite(t *testing.T, subject Subject) {
	t.Run("missing key", func(t *testing.T) { checkMissingKey(t, subject.Driver) })
	t.Run("set and get", func(t *testing.T) { checkSetAndGet(t, subject.Driver) })
	t.Run("set replaces value and ttl", func(t *testing.T) { checkSetReplaces(t, subject.Driver) })
	t.Run("expired key is missing", func(t *testing.T) { checkExpiredKey(t, subject) })
	t.Run("take returns value once", func(t *testing.T) { checkTake(t, subject.Driver) })
	t.Run("update expiration", func(t *testing.T) { checkUpdateExpiration(t, subject.Driver) })
	t.Run("delete", func(t *testing.T) { checkDelete(t, subject.Driver) })
	t.Run("increment", func(t *testing.T) { checkIncrement(t, subject.Driver) })
	t.Run("increment after expiry starts again", func(t *testing.T) { checkIncrementAfterExpiry(t, subject) })
	t.Run("increment rejects a non integer", func(t *testing.T) { checkIncrementNonInteger(t, subject.Driver) })
	t.Run("concurrent increment", func(t *testing.T) { checkConcurrentIncrement(t, subject.Driver) })
	t.Run("concurrent take", func(t *testing.T) { checkConcurrentTake(t, subject.Driver) })
}

func checkMissingKey(t *testing.T, driver cache.Driver) {
	ctx := t.Context()
	key := "suite:missing"

	value, exists, err := driver.Get(ctx, key)
	require.NoError(t, err)
	require.False(t, exists)
	require.Nil(t, value)

	value, exists, err = driver.Take(ctx, key)
	require.NoError(t, err)
	require.False(t, exists)
	require.Nil(t, value)

	exists, err = driver.Exists(ctx, key)
	require.NoError(t, err)
	require.False(t, exists)

	ttl, hasTTL, err := driver.TTL(ctx, key)
	require.NoError(t, err)
	require.False(t, hasTTL)
	require.Zero(t, ttl)

	updated, err := driver.UpdateExpiration(ctx, key, longTTL)
	require.NoError(t, err)
	require.False(t, updated)

	require.NoError(t, driver.Delete(ctx, key))
}

func checkSetAndGet(t *testing.T, driver cache.Driver) {
	ctx := t.Context()
	values := map[string][]byte{
		"suite:set:text":           []byte("value"),
		"suite:set:binary":         {0, 0xff, 0x80, '\n', 0},
		"suite:set:empty":          {},
		"suite:set:key/with space": []byte("žluťoučký kůň"),
	}
	for key, value := range values {
		require.NoError(t, driver.Set(ctx, key, value, longTTL))

		got, exists, err := driver.Get(ctx, key)
		require.NoError(t, err)
		require.True(t, exists, key)
		require.Equal(t, string(value), string(got), key)

		exists, err = driver.Exists(ctx, key)
		require.NoError(t, err)
		require.True(t, exists, key)
		requireTTL(t, driver, key, longTTL)
	}
}

func checkSetReplaces(t *testing.T, driver cache.Driver) {
	ctx := t.Context()
	key := "suite:replace"

	require.NoError(t, driver.Set(ctx, key, []byte("first"), longTTL))
	require.NoError(t, driver.Set(ctx, key, []byte("second"), 0))

	value, exists, err := driver.Get(ctx, key)
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, "second", string(value))
	requireNoTTL(t, driver, key)

	require.NoError(t, driver.Set(ctx, key, []byte("third"), 2*longTTL))
	requireTTL(t, driver, key, 2*longTTL)
}

func checkExpiredKey(t *testing.T, subject Subject) {
	ctx := t.Context()
	driver := subject.Driver
	keys := []string{"suite:expire:get", "suite:expire:take", "suite:expire:update"}
	for _, key := range keys {
		require.NoError(t, driver.Set(ctx, key, []byte("value"), shortTTL))
	}
	require.NoError(t, driver.Set(ctx, "suite:expire:forever", []byte("value"), 0))

	subject.AdvanceClock(shortTTL + 100*time.Millisecond)

	_, exists, err := driver.Get(ctx, "suite:expire:get")
	require.NoError(t, err)
	require.False(t, exists)
	exists, err = driver.Exists(ctx, "suite:expire:get")
	require.NoError(t, err)
	require.False(t, exists)
	requireNoTTL(t, driver, "suite:expire:get")

	_, exists, err = driver.Take(ctx, "suite:expire:take")
	require.NoError(t, err)
	require.False(t, exists)

	updated, err := driver.UpdateExpiration(ctx, "suite:expire:update", longTTL)
	require.NoError(t, err)
	require.False(t, updated)

	exists, err = driver.Exists(ctx, "suite:expire:forever")
	require.NoError(t, err)
	require.True(t, exists)
}

func checkTake(t *testing.T, driver cache.Driver) {
	ctx := t.Context()
	key := "suite:take"
	require.NoError(t, driver.Set(ctx, key, []byte("once"), longTTL))

	value, exists, err := driver.Take(ctx, key)
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, "once", string(value))

	value, exists, err = driver.Take(ctx, key)
	require.NoError(t, err)
	require.False(t, exists)
	require.Nil(t, value)

	exists, err = driver.Exists(ctx, key)
	require.NoError(t, err)
	require.False(t, exists)
}

func checkUpdateExpiration(t *testing.T, driver cache.Driver) {
	ctx := t.Context()
	key := "suite:update"

	require.NoError(t, driver.Set(ctx, key, []byte("value"), longTTL))
	updated, err := driver.UpdateExpiration(ctx, key, 3*longTTL)
	require.NoError(t, err)
	require.True(t, updated)
	requireTTL(t, driver, key, 3*longTTL)

	require.NoError(t, driver.Set(ctx, key, []byte("value"), 0))
	updated, err = driver.UpdateExpiration(ctx, key, longTTL)
	require.NoError(t, err)
	require.True(t, updated)
	requireTTL(t, driver, key, longTTL)

	value, exists, err := driver.Get(ctx, key)
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, "value", string(value))
}

func checkDelete(t *testing.T, driver cache.Driver) {
	ctx := t.Context()
	key := "suite:delete"
	require.NoError(t, driver.Set(ctx, key, []byte("value"), longTTL))
	require.NoError(t, driver.Set(ctx, "suite:delete:kept", []byte("value"), longTTL))

	require.NoError(t, driver.Delete(ctx, key))
	exists, err := driver.Exists(ctx, key)
	require.NoError(t, err)
	require.False(t, exists)
	require.NoError(t, driver.Delete(ctx, key))

	exists, err = driver.Exists(ctx, "suite:delete:kept")
	require.NoError(t, err)
	require.True(t, exists)
}

func checkIncrement(t *testing.T, driver cache.Driver) {
	ctx := t.Context()

	t.Run("keeps the first ttl", func(t *testing.T) {
		key := "suite:incr:ttl"
		requireIncrement(t, driver, key, 1, longTTL, 1)
		requireIncrement(t, driver, key, 2, 3*longTTL, 3)
		requireTTL(t, driver, key, longTTL)
		requireIncrement(t, driver, key, -1, 3*longTTL, 2)
		requireTTL(t, driver, key, longTTL)
	})

	t.Run("stores a decimal value", func(t *testing.T) {
		key := "suite:incr:text"
		requireIncrement(t, driver, key, 42, longTTL, 42)
		value, exists, err := driver.Get(ctx, key)
		require.NoError(t, err)
		require.True(t, exists)
		require.Equal(t, "42", string(value))
	})

	t.Run("continues from a set value", func(t *testing.T) {
		key := "suite:incr:set"
		require.NoError(t, driver.Set(ctx, key, []byte("10"), 0))
		requireIncrement(t, driver, key, 5, longTTL, 15)
		requireTTL(t, driver, key, longTTL)
	})

	t.Run("negative delta does not create a key", func(t *testing.T) {
		key := "suite:incr:absent"
		requireIncrement(t, driver, key, -1, longTTL, 0)
		exists, err := driver.Exists(ctx, key)
		require.NoError(t, err)
		require.False(t, exists)
	})

	t.Run("zero ttl keeps the key", func(t *testing.T) {
		key := "suite:incr:forever"
		requireIncrement(t, driver, key, 1, 0, 1)
		requireNoTTL(t, driver, key)
	})
}

func checkIncrementAfterExpiry(t *testing.T, subject Subject) {
	key := "suite:incr:expire"
	requireIncrement(t, subject.Driver, key, 5, shortTTL, 5)

	subject.AdvanceClock(shortTTL + 100*time.Millisecond)

	requireIncrement(t, subject.Driver, key, 1, longTTL, 1)
	requireTTL(t, subject.Driver, key, longTTL)
}

func checkIncrementNonInteger(t *testing.T, driver cache.Driver) {
	ctx := t.Context()
	key := "suite:incr:text-value"
	for _, stored := range []string{"abc", " 5", "5.0", ""} {
		require.NoError(t, driver.Set(ctx, key, []byte(stored), longTTL))

		for _, delta := range []int64{1, -1} {
			_, err := driver.Increment(ctx, key, delta, longTTL)
			require.Error(t, err, "%q %+d", stored, delta)
		}

		value, exists, err := driver.Get(ctx, key)
		require.NoError(t, err)
		require.True(t, exists)
		require.Equal(t, stored, string(value))
	}
}

func checkConcurrentIncrement(t *testing.T, driver cache.Driver) {
	ctx := t.Context()
	key := "suite:incr:concurrent"
	var failures atomic.Int32
	var wg sync.WaitGroup
	for range concurrency {
		wg.Go(func() {
			if _, err := driver.Increment(ctx, key, 1, longTTL); err != nil {
				failures.Add(1)
			}
		})
	}
	wg.Wait()
	require.Zero(t, failures.Load())

	value, exists, err := driver.Get(ctx, key)
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, strconv.Itoa(concurrency), string(value))
}

func checkConcurrentTake(t *testing.T, driver cache.Driver) {
	ctx := t.Context()
	key := "suite:take:concurrent"
	require.NoError(t, driver.Set(ctx, key, []byte("once"), longTTL))

	var taken, failures atomic.Int32
	var wg sync.WaitGroup
	for range concurrency {
		wg.Go(func() {
			_, exists, err := driver.Take(ctx, key)
			if err != nil {
				failures.Add(1)
			}
			if exists {
				taken.Add(1)
			}
		})
	}
	wg.Wait()
	require.Zero(t, failures.Load())
	require.Equal(t, int32(1), taken.Load())
}

func requireIncrement(
	t *testing.T,
	driver cache.Driver,
	key string,
	delta int64,
	ttl time.Duration,
	expected int64,
) {
	t.Helper()
	value, err := driver.Increment(t.Context(), key, delta, ttl)
	require.NoError(t, err)
	require.Equal(t, expected, value)
}

func requireTTL(t *testing.T, driver cache.Driver, key string, expected time.Duration) {
	t.Helper()
	ttl, hasTTL, err := driver.TTL(t.Context(), key)
	require.NoError(t, err)
	require.True(t, hasTTL, key)
	require.InDelta(t, expected, ttl, float64(ttlTolerance), key)
}

func requireNoTTL(t *testing.T, driver cache.Driver, key string) {
	t.Helper()
	ttl, hasTTL, err := driver.TTL(t.Context(), key)
	require.NoError(t, err)
	require.False(t, hasTTL, key)
	require.Zero(t, ttl, key)
}
