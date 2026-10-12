package csrf

import (
	"context"
	"time"
)

type Driver interface {
	Add(ctx context.Context, key, token string, expiresAt time.Time, limit int) error
	Take(ctx context.Context, key, token string, now time.Time) (bool, error)
}
