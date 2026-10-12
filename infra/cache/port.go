package cache

import (
	"context"
	"time"
)

type Driver interface {
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Take(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	UpdateExpiration(ctx context.Context, key string, ttl time.Duration) (bool, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	TTL(ctx context.Context, key string) (time.Duration, bool, error)
	Increment(ctx context.Context, key string, delta int64, ttl time.Duration) (int64, error)
}

type Commander interface {
	Exists(ctx context.Context, key string) (bool, error)
	TTL(ctx context.Context, key string) (time.Duration, error)
	GetString(ctx context.Context, key string) (string, error)
	GetJSON(ctx context.Context, key string, target any) (bool, error)
	TakeString(ctx context.Context, key string) (string, error)
	TakeJSON(ctx context.Context, key string, target any) (bool, error)
	SetString(ctx context.Context, key string, value string, ttl time.Duration) error
	SetJSON(ctx context.Context, key string, value any, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}
