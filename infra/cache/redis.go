package cache

import (
	"context"
	"errors"
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

func (d *RedisDriver) Get(ctx context.Context, key string) ([]byte, bool, error) {
	value, err := d.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("get cache value %q failed: %w", key, err)
	}
	return value, true, nil
}

func (d *RedisDriver) Take(ctx context.Context, key string) ([]byte, bool, error) {
	value, err := d.client.GetDel(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("take cache value %q failed: %w", key, err)
	}
	return value, true, nil
}

func (d *RedisDriver) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if err := d.client.Set(ctx, key, value, ttl).Err(); err != nil {
		return fmt.Errorf("set cache value %q failed: %w", key, err)
	}
	return nil
}

func (d *RedisDriver) UpdateExpiration(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	var zero bool

	updated, err := d.client.PExpire(ctx, key, ttl).Result()
	if err != nil {
		return zero, fmt.Errorf("update cache expiration of %q failed: %w", key, err)
	}
	return updated, nil
}

func (d *RedisDriver) Exists(ctx context.Context, key string) (bool, error) {
	var zero bool

	n, err := d.client.Exists(ctx, key).Result()
	if err != nil {
		return zero, fmt.Errorf("check cache existence of %q failed: %w", key, err)
	}
	return n == 1, nil
}

func (d *RedisDriver) TTL(ctx context.Context, key string) (time.Duration, bool, error) {
	var zero time.Duration

	ttl, err := d.client.PTTL(ctx, key).Result()
	if err != nil {
		return zero, false, fmt.Errorf("get cache ttl of %q failed: %w", key, err)
	}
	if ttl < 0 {
		return 0, false, nil
	}
	return ttl, true, nil
}

func (d *RedisDriver) Increment(ctx context.Context, key string, delta int64, ttl time.Duration) (int64, error) {
	var zero int64

	value, err := incrementScript.Run(ctx, d.client, []string{key}, delta, ttl.Milliseconds()).Int64()
	if err != nil {
		return zero, fmt.Errorf("increment cache value %q by %d failed: %w", key, delta, err)
	}
	return value, nil
}

func (d *RedisDriver) Delete(ctx context.Context, key string) error {
	if err := d.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("delete cache value %q failed: %w", key, err)
	}
	return nil
}

func (d *RedisDriver) DeleteByPattern(
	ctx context.Context,
	pattern string,
	batchSize int64,
) error {
	var cursor uint64
	for {
		keys, next, err := d.client.Scan(ctx, cursor, pattern, batchSize).Result()
		if err != nil {
			return fmt.Errorf("scan cache keys by pattern %q failed: %w", pattern, err)
		}
		if len(keys) > 0 {
			if delErr := d.client.Del(ctx, keys...).Err(); delErr != nil {
				return fmt.Errorf("delete cache keys by pattern %q failed: %w", pattern, delErr)
			}
		}
		if next == 0 {
			return nil
		}
		cursor = next
	}
}
