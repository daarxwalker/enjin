package lock

import (
	"context"
	"time"
)

type Driver interface {
	TryLock(ctx context.Context, key, token string, ttl time.Duration) (bool, error)
	Unlock(ctx context.Context, key, token string) (bool, error)
}
