package lock_test

import (
	"context"
	"testing"
	"time"

	"github.com/daarxwalker/enjin/enjintest"
	"github.com/daarxwalker/enjin/infra/db/migrations/keyvalue"
	"github.com/daarxwalker/enjin/lock"
	"github.com/daarxwalker/enjin/lock/locktest"
)

func TestPostgresDriver(t *testing.T) {
	pool := enjintest.NewPostgres(context.Background(), t, enjintest.WithMigrationsFS(keyvalue.FS))
	locktest.RunDriverSuite(t, locktest.Subject{
		Driver:       lock.NewPostgresDriver(pool),
		AdvanceClock: time.Sleep,
	})
}
