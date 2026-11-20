package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/daarxwalker/enjin/enjintest"
	"github.com/daarxwalker/enjin/infra/cache"
	"github.com/daarxwalker/enjin/infra/cache/cachetest"
	"github.com/daarxwalker/enjin/infra/db/migrations/keyvalue"

	"github.com/stretchr/testify/require"
)

func TestPostgresDriver(t *testing.T) {
	pool := enjintest.NewPostgres(context.Background(), t, enjintest.WithMigrationsFS(keyvalue.FS))
	driver := cache.NewPostgresDriver(pool)

	cachetest.RunDriverSuite(t, cachetest.Subject{
		Driver:       driver,
		AdvanceClock: time.Sleep,
	})

	t.Run("delete by pattern", func(t *testing.T) {
		ctx := t.Context()
		keys := []string{"scan:a", "scan:b", "scan:nested:c", "scan_x", "scanner", "keep:scan:a"}
		for _, key := range keys {
			require.NoError(t, driver.Set(ctx, key, []byte("x"), time.Minute))
		}

		require.NoError(t, driver.DeleteByPattern(ctx, "scan:*"))
		for _, key := range keys {
			exists, err := driver.Exists(ctx, key)
			require.NoError(t, err)
			deleted := key == "scan:a" || key == "scan:b" || key == "scan:nested:c"
			require.Equal(t, !deleted, exists, key)
		}

		require.NoError(t, driver.DeleteByPattern(ctx, `scan\_?`))
		exists, err := driver.Exists(ctx, "scan_x")
		require.NoError(t, err)
		require.False(t, exists)
		exists, err = driver.Exists(ctx, "scanner")
		require.NoError(t, err)
		require.True(t, exists)
	})
}
