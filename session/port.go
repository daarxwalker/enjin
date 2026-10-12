package session

import (
	"context"
	"time"
)

type cache interface {
	GetJSON(ctx context.Context, key string, t any) (bool, error)
	SetJSON(ctx context.Context, key string, v any, ttl time.Duration) error
	UpdateExpiration(ctx context.Context, key string, ttl time.Duration) (bool, error)
	Delete(ctx context.Context, key string) error
}
