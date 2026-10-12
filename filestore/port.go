package filestore

import (
	"context"
	"io"
)

type Driver interface {
	List(ctx context.Context, prefix string) ([]File, error)
	Stat(ctx context.Context, key string) (File, bool, error)
	Get(ctx context.Context, key string) (*Object, bool, error)
	Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) error
	Delete(ctx context.Context, key string) error
}

type Commander interface {
	List(ctx context.Context, prefix string) ([]File, error)
	Stat(ctx context.Context, key string) (File, bool, error)
	Get(ctx context.Context, key string) (*Object, bool, error)
	GetBytes(ctx context.Context, key string) ([]byte, bool, error)
	Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) error
	PutBytes(ctx context.Context, key string, value []byte, contentType string) error
	Delete(ctx context.Context, key string) error
}
