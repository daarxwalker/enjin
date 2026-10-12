package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ConnectOption func(*connectOptions)

type connectOptions struct {
	maxConns          int32
	minConns          int32
	maxConnLifetime   time.Duration
	maxConnIdleTime   time.Duration
	healthCheckPeriod time.Duration
	connectTimeout    time.Duration
	ctxTimeout        time.Duration
}

func Connect(ctx context.Context, uri string, options ...ConnectOption) (*pgxpool.Pool, error) {
	opts := &connectOptions{
		maxConns:          20,
		minConns:          2,
		maxConnLifetime:   30 * time.Minute,
		maxConnIdleTime:   5 * time.Minute,
		healthCheckPeriod: 1 * time.Minute,
		connectTimeout:    10 * time.Second,
		ctxTimeout:        10 * time.Second,
	}
	for _, opt := range options {
		opt(opts)
	}
	c, cancel := context.WithTimeout(ctx, opts.ctxTimeout)
	defer cancel()
	config, parseURIErr := pgxpool.ParseConfig(uri)
	if parseURIErr != nil {
		return nil, fmt.Errorf("db: parse uri failed: %w", parseURIErr)
	}
	config.MaxConns = opts.maxConns
	config.MinConns = opts.minConns
	config.MaxConnLifetime = opts.maxConnLifetime
	config.MaxConnIdleTime = opts.maxConnIdleTime
	config.HealthCheckPeriod = opts.healthCheckPeriod
	config.ConnConfig.ConnectTimeout = opts.connectTimeout
	pool, connectErr := pgxpool.NewWithConfig(c, config)
	if connectErr != nil {
		return nil, fmt.Errorf("db: connect failed: %w", connectErr)
	}
	if pingErr := pool.Ping(c); pingErr != nil {
		return nil, fmt.Errorf("db: ping failed: %w", pingErr)
	}
	return pool, nil
}

func WithMaxConns(maxConns int32) ConnectOption {
	return func(opts *connectOptions) {
		opts.maxConns = maxConns
	}
}

func WithMinConns(minConns int32) ConnectOption {
	return func(opts *connectOptions) {
		opts.minConns = minConns
	}
}

func WithMaxConnLifetime(maxConnLifetime time.Duration) ConnectOption {
	return func(opts *connectOptions) {
		opts.maxConnLifetime = maxConnLifetime
	}
}

func WithMaxConnIdleTime(maxConnIdleTime time.Duration) ConnectOption {
	return func(opts *connectOptions) {
		opts.maxConnIdleTime = maxConnIdleTime
	}
}

func WithHealthCheckPeriod(healthCheckPeriod time.Duration) ConnectOption {
	return func(opts *connectOptions) {
		opts.healthCheckPeriod = healthCheckPeriod
	}
}

func WithConnectTimeout(connectTimeout time.Duration) ConnectOption {
	return func(opts *connectOptions) {
		opts.connectTimeout = connectTimeout
	}
}

func WithContextTimeout(ctxTimeout time.Duration) ConnectOption {
	return func(opts *connectOptions) {
		opts.ctxTimeout = ctxTimeout
	}
}
