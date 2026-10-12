package cache

import (
	"context"
	"errors"
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
		return nil, false, err
	}
	return value, true, nil
}

func (d *RedisDriver) Take(ctx context.Context, key string) ([]byte, bool, error) {
	value, err := d.client.GetDel(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return value, true, nil
}

func (d *RedisDriver) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return d.client.Set(ctx, key, value, ttl).Err()
}

func (d *RedisDriver) UpdateExpiration(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return d.client.PExpire(ctx, key, ttl).Result()
}

func (d *RedisDriver) Exists(ctx context.Context, key string) (bool, error) {
	n, err := d.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return n == 1, nil
}

func (d *RedisDriver) TTL(ctx context.Context, key string) (time.Duration, bool, error) {
	ttl, err := d.client.PTTL(ctx, key).Result()
	if err != nil {
		return 0, false, err
	}
	if ttl < 0 {
		return 0, false, nil
	}
	return ttl, true, nil
}

func (d *RedisDriver) Increment(ctx context.Context, key string, delta int64, ttl time.Duration) (int64, error) {
	return incrementScript.Run(ctx, d.client, []string{key}, delta, ttl.Milliseconds()).Int64()
}

func (d *RedisDriver) Delete(ctx context.Context, key string) error {
	return d.client.Del(ctx, key).Err()
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
			return err
		}
		if len(keys) > 0 {
			if delErr := d.client.Del(ctx, keys...).Err(); delErr != nil {
				return delErr
			}
		}
		if next == 0 {
			return nil
		}
		cursor = next
	}
}
