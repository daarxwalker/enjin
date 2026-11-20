package csrf

import (
	"context"
	"fmt"
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
	err := addScript.Run(
		ctx, d.client, []string{key},
		token, expiresAt.UnixMilli(), time.Now().UnixMilli(), limit,
	).Err()
	if err != nil {
		return fmt.Errorf("add csrf token for %q failed: %w", key, err)
	}
	return nil
}

func (d *RedisDriver) Take(ctx context.Context, key, token string, now time.Time) (bool, error) {
	var zero bool

	taken, err := takeScript.Run(ctx, d.client, []string{key}, token, now.UnixMilli()).Int()
	if err != nil {
		return zero, fmt.Errorf("take csrf token for %q failed: %w", key, err)
	}
	return taken == 1, nil
}
