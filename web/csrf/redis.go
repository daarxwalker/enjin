package csrf

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisDriver struct {
	client *redis.Client
}

func NewRedisDriver(client *redis.Client) *RedisDriver {
	return &RedisDriver{client: client}
}

func (d *RedisDriver) Add(
	ctx context.Context,
	key, token string,
	expiresAt time.Time,
	limit int,
) error {
	return addScript.Run(
		ctx, d.client, []string{key},
		token, expiresAt.UnixMilli(), time.Now().UnixMilli(), limit,
	).Err()
}

func (d *RedisDriver) Take(ctx context.Context, key, token string, now time.Time) (bool, error) {
	taken, err := takeScript.Run(ctx, d.client, []string{key}, token, now.UnixMilli()).Int()
	if err != nil {
		return false, err
	}
	return taken == 1, nil
}
