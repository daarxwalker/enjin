package lock

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

var unlockScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("DEL", KEYS[1])
end
return 0
`)

type RedisDriver struct {
	client *redis.Client
}

func NewRedisDriver(client *redis.Client) *RedisDriver {
	return &RedisDriver{client: client}
}

func (d *RedisDriver) TryLock(
	ctx context.Context,
	key, token string,
	ttl time.Duration,
) (bool, error) {
	ok, err := d.client.SetNX(ctx, key, token, ttl).Result()
	if err != nil {
		return false, err
	}
	return ok, nil
}

func (d *RedisDriver) Unlock(ctx context.Context, key, token string) (bool, error) {
	deleted, err := unlockScript.Run(ctx, d.client, []string{key}, token).Int()
	if err != nil {
		return false, err
	}
	return deleted == 1, nil
}
