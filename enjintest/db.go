package enjintest

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
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
			// The image starts a temporary server for initdb first and then restarts, so the port
			// may already be open while the first server is shutting down.
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
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

	migrateClient := stdlib.OpenDBFromPool(pool)
	defer func() {
		if closeErr := migrateClient.Close(); closeErr != nil {
			require.NoError(t, fmt.Errorf("close migrate client failed: %w", closeErr))
		}
	}()
	if cfg.migrationsFS != nil {
		runMigrations(ctx, t, migrateClient, cfg.migrationsFS, goose.DefaultTablename)
	}
	for _, migrations := range cfg.tableMigrations {
		runMigrations(ctx, t, migrateClient, migrations.migrationsFS, migrations.versionTable)
	}

	return pool
}

func runMigrations(ctx context.Context, t testing.TB, migrateClient *sql.DB, migrationsFS fs.FS, versionTable string) {
	t.Helper()
	provider, err := goose.NewProvider(
		goose.DialectPostgres,
		migrateClient,
		migrationsFS,
		goose.WithTableName(versionTable),
	)
	require.NoError(t, err, "create goose provider for %s", versionTable)
	_, err = provider.Up(ctx)
	require.NoError(t, err, "run migrations of %s", versionTable)
}
