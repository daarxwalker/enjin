package lock

import (
	"context"
	"testing"
	"time"

	"github.com/daarxwalker/enjin/internal/testredis"

	"github.com/stretchr/testify/require"
)

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
