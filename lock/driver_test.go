package lock_test

import (
	"testing"

	"github.com/daarxwalker/enjin/internal/testredis"
	"github.com/daarxwalker/enjin/lock"
	"github.com/daarxwalker/enjin/lock/locktest"
)

func TestRedisDriver(t *testing.T) {
	rdb, server := testredis.NewWithServer(t)
	locktest.RunDriverSuite(t, locktest.Subject{
		Driver:       lock.NewRedisDriver(rdb),
		AdvanceClock: server.FastForward,
	})
}
