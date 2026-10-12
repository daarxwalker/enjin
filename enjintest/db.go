package enjintest

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func NewPostgres(ctx context.Context, t testing.TB, options ...Option) *pgxpool.Pool {
	skipShort(t)
	cfg := config{
		version: "18.4-alpine",
		port:    5432,
	}
	for _, opt := range options {
		opt(&cfg)
	}
	container, runErr := postgres.Run(
		ctx,
		"postgres:"+cfg.version,
		postgres.WithDatabase("test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort(strconv.Itoa(cfg.port)+"/tcp"),
		),
	)
	require.NoError(t, runErr)

	t.Cleanup(
		func() {
			require.NoError(t, container.Terminate(context.WithoutCancel(ctx)))
			t.Log("cleanup db done")
		},
	)

	endpoint, dsnErr := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, dsnErr)

	config, parseURIErr := pgxpool.ParseConfig(endpoint)
	require.NoError(t, parseURIErr)

	pool, connectErr := pgxpool.NewWithConfig(ctx, config)
	require.NoError(t, connectErr)

	t.Cleanup(pool.Close)

	if cfg.migrationsFS != nil {
		migrateClient := stdlib.OpenDBFromPool(pool)
		defer func() {
			if closeErr := migrateClient.Close(); closeErr != nil {
				require.NoError(t, fmt.Errorf("close migrate client failed: %w", closeErr))
			}
		}()
		provider, err := goose.NewProvider(goose.DialectPostgres, migrateClient, cfg.migrationsFS)
		if err != nil {
			require.NoError(t, fmt.Errorf("set goose dialect failed: %w", err))
			return nil
		}
		if _, err = provider.Up(ctx); err != nil {
			require.NoError(t, fmt.Errorf("run migrations failed: %w", err))
			return nil
		}
	}

	return pool
}
