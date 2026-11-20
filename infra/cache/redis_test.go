package cache_test

import (
	"testing"

	"github.com/daarxwalker/enjin/infra/cache"
	"github.com/daarxwalker/enjin/infra/cache/cachetest"
	"github.com/daarxwalker/enjin/internal/testredis"
)

func TestRedisDriver(t *testing.T) {
	rdb, server := testredis.NewWithServer(t)
	cachetest.RunDriverSuite(t, cachetest.Subject{
		Driver:       cache.NewRedisDriver(rdb),
		AdvanceClock: server.FastForward,
	})
}
