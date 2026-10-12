package lock

import (
	"context"
	"testing"
	"time"

	"github.com/daarxwalker/enjin/internal/testredis"

	"github.com/stretchr/testify/require"
)

func TestDriver(t *testing.T) {
	ctx := context.Background()
	cache := testredis.New(t)
	driver := NewRedisDriver(cache)
	require.NotNil(t, driver)

	t.Run(
		"try lock", func(t *testing.T) {
			key := "test-key"
			locked, lockErr := driver.TryLock(ctx, key, "owner", time.Minute)
			require.NoError(t, lockErr)
			require.True(t, locked)
			released, unlockErr := driver.Unlock(ctx, key, "owner")
			require.NoError(t, unlockErr)
			require.True(t, released)
		},
	)

	t.Run(
		"try lock held", func(t *testing.T) {
			key := "test-key-held"
			locked, lockErr := driver.TryLock(ctx, key, "owner", time.Minute)
			require.NoError(t, lockErr)
			require.True(t, locked)
			lockedAgain, lockAgainErr := driver.TryLock(ctx, key, "other", time.Minute)
			require.NoError(t, lockAgainErr)
			require.False(t, lockedAgain)
		},
	)

	t.Run(
		"unlock foreign token", func(t *testing.T) {
			key := "test-key-foreign"
			locked, lockErr := driver.TryLock(ctx, key, "owner", time.Minute)
			require.NoError(t, lockErr)
			require.True(t, locked)
			released, unlockErr := driver.Unlock(ctx, key, "other")
			require.NoError(t, unlockErr)
			require.False(t, released)
			value, getErr := cache.Get(ctx, key).Result()
			require.NoError(t, getErr)
			require.Equal(t, "owner", value)
		},
	)
}

func TestManager(t *testing.T) {
	ctx := context.Background()
	cache := testredis.New(t)
	locker := NewManager(NewRedisDriver(cache), time.Minute)

	t.Run(
		"lock unlock", func(t *testing.T) {
			lease, lockErr := locker.Lock(ctx, "sync", "1")
			require.NoError(t, lockErr)
			require.NoError(t, locker.Unlock(ctx, lease))
		},
	)

	t.Run(
		"stale lease does not release new holder", func(t *testing.T) {
			staleLease, lockErr := locker.Lock(ctx, "sync", "2")
			require.NoError(t, lockErr)
			require.NoError(t, cache.Del(ctx, staleLease.key).Err())

			currentLease, relockErr := locker.Lock(ctx, "sync", "2")
			require.NoError(t, relockErr)

			staleErr := locker.Unlock(ctx, staleLease)
			require.ErrorIs(t, staleErr, ErrNotHeld)
			require.NoError(t, locker.Unlock(ctx, currentLease))
		},
	)
}
