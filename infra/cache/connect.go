package cache

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type ConnectOption func(*connectOptions)

type connectOptions struct {
	password   string
	db         int
	useTLS     bool
	ctxTimeout time.Duration
}

func Connect(ctx context.Context, addr string, options ...ConnectOption) (*redis.Client, error) {
	opts := &connectOptions{
		ctxTimeout: 10 * time.Second,
	}
	for _, opt := range options {
		opt(opts)
	}
	c, cancel := context.WithTimeout(ctx, opts.ctxTimeout)
	defer cancel()
	clientOpts := &redis.Options{
		Addr:     addr,
		Password: opts.password,
		DB:       opts.db,
	}
	if opts.useTLS {
		clientOpts.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}
	client := redis.NewClient(clientOpts)
	if err := client.Ping(c).Err(); err != nil {
		return nil, fmt.Errorf("cache: connect failed: %w", err)
	}
	return client, nil
}

func WithPassword(password string) ConnectOption {
	return func(opts *connectOptions) {
		opts.password = password
	}
}

func WithDB(db int) ConnectOption {
	return func(opts *connectOptions) {
		opts.db = db
	}
}

func WithTLS(use bool) ConnectOption {
	return func(opts *connectOptions) {
		opts.useTLS = use
	}
}

func WithContextTimeout(timeout time.Duration) ConnectOption {
	return func(opts *connectOptions) {
		opts.ctxTimeout = timeout
	}
}
