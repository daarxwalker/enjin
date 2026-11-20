package locktest

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/daarxwalker/enjin/lock"

	"github.com/stretchr/testify/require"
)

const (
	longTTL     = time.Minute
	shortTTL    = time.Second
	concurrency = 20
)

type Subject struct {
	Driver lock.Driver
	// AdvanceClock moves the time the driver sees forward. A store with its
	// own clock (a database server) implements it by sleeping.
	AdvanceClock func(time.Duration)
}

// RunDriverSuite checks that a Driver behaves like the Redis one. Every case
// uses its own keys, so the store may be shared with other tests.
func RunDriverSuite(t *testing.T, subject Subject) {
	t.Run("lock and unlock", func(t *testing.T) { checkLockAndUnlock(t, subject.Driver) })
	t.Run("held lock rejects everyone", func(t *testing.T) { checkHeldLock(t, subject.Driver) })
	t.Run("unlock needs the owner token", func(t *testing.T) { checkForeignUnlock(t, subject.Driver) })
	t.Run("unlock of a free key", func(t *testing.T) { checkUnlockFree(t, subject.Driver) })
	t.Run("expired lock is free", func(t *testing.T) { checkExpiredLock(t, subject) })
	t.Run("concurrent lock has one winner", func(t *testing.T) { checkConcurrentLock(t, subject.Driver) })
}

func checkLockAndUnlock(t *testing.T, driver lock.Driver) {
	key := "suite:lock"
	requireLock(t, driver, key, "owner", longTTL, true)
	requireUnlock(t, driver, key, "owner", true)
	requireLock(t, driver, key, "other", longTTL, true)
}

func checkHeldLock(t *testing.T, driver lock.Driver) {
	key := "suite:held"
	requireLock(t, driver, key, "owner", longTTL, true)
	requireLock(t, driver, key, "other", longTTL, false)
	requireLock(t, driver, key, "owner", longTTL, false)
}

func checkForeignUnlock(t *testing.T, driver lock.Driver) {
	key := "suite:foreign"
	requireLock(t, driver, key, "owner", longTTL, true)
	requireUnlock(t, driver, key, "other", false)
	requireLock(t, driver, key, "other", longTTL, false)
	requireUnlock(t, driver, key, "owner", true)
}

func checkUnlockFree(t *testing.T, driver lock.Driver) {
	requireUnlock(t, driver, "suite:free", "owner", false)
}

func checkExpiredLock(t *testing.T, subject Subject) {
	key := "suite:expired"
	requireLock(t, subject.Driver, key, "owner", shortTTL, true)

	subject.AdvanceClock(shortTTL + 100*time.Millisecond)

	requireLock(t, subject.Driver, key, "other", longTTL, true)
	requireUnlock(t, subject.Driver, key, "owner", false)
	requireUnlock(t, subject.Driver, key, "other", true)
}

func checkConcurrentLock(t *testing.T, driver lock.Driver) {
	ctx := t.Context()
	key := "suite:concurrent"
	var locked, failures atomic.Int32
	var wg sync.WaitGroup
	for i := range concurrency {
		wg.Go(func() {
			ok, err := driver.TryLock(ctx, key, fmt.Sprintf("owner-%d", i), longTTL)
			if err != nil {
				failures.Add(1)
			}
			if ok {
				locked.Add(1)
			}
		})
	}
	wg.Wait()
	require.Zero(t, failures.Load())
	require.Equal(t, int32(1), locked.Load())
}

func requireLock(t *testing.T, driver lock.Driver, key, token string, ttl time.Duration, expected bool) {
	t.Helper()
	locked, err := driver.TryLock(t.Context(), key, token, ttl)
	require.NoError(t, err)
	require.Equal(t, expected, locked, "lock %s by %s", key, token)
}

func requireUnlock(t *testing.T, driver lock.Driver, key, token string, expected bool) {
	t.Helper()
	released, err := driver.Unlock(t.Context(), key, token)
	require.NoError(t, err)
	require.Equal(t, expected, released, "unlock %s by %s", key, token)
}
