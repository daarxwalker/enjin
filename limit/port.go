package limit

import (
	"context"
	"time"
)

type Limiter interface {
	Current(ctx context.Context) int
	Max() int
	Consume(ctx context.Context) (bool, error)
	Revert(ctx context.Context) error
	ResetIn(ctx context.Context) time.Duration
}

type Counter interface {
	GetString(ctx context.Context, key string) (string, error)
	TTL(ctx context.Context, key string) (time.Duration, error)
	Increment(ctx context.Context, key string, delta int, ttl time.Duration) (int, error)
}
