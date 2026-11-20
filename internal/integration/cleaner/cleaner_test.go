package cleaner_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/daarxwalker/enjin/enjintest"
	"github.com/daarxwalker/enjin/infra/db/cleaner"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestCleaner(t *testing.T) {
	ctx := context.Background()
	pool := enjintest.NewPostgres(ctx, t)
	_, err := pool.Exec(ctx, `
CREATE SCHEMA app;
CREATE TABLE app.sessions (id text PRIMARY KEY, expires_at timestamptz);
CREATE TABLE tokens (id text PRIMARY KEY, expires_at timestamptz NOT NULL);
CREATE TABLE untouched (id text PRIMARY KEY, expires_at timestamptz);
CREATE TABLE later (id text PRIMARY KEY, expires_at timestamptz);
INSERT INTO app.sessions VALUES
    ('expired-1', now() - interval '1 minute'),
    ('expired-2', now() - interval '1 minute'),
    ('expired-3', now() - interval '1 second'),
    ('live', now() + interval '1 hour'),
    ('forever', NULL);
INSERT INTO tokens VALUES
    ('expired', now() - interval '1 minute'),
    ('live', now() + interval '1 hour');
INSERT INTO untouched VALUES ('expired', now() - interval '1 minute');
INSERT INTO later VALUES ('expired', now() - interval '1 minute');
`)
	require.NoError(t, err)

	t.Run("deletes expired rows of the given tables", func(t *testing.T) {
		deleted, deleteErr := cleaner.NewCleaner(pool, []string{"app.sessions", "tokens"}, cleaner.WithBatchSize(2)).
			DeleteExpiredRows(t.Context())
		require.NoError(t, deleteErr)
		require.Equal(t, int64(4), deleted)

		require.ElementsMatch(t, []string{"live", "forever"}, loadIDs(t, pool, "app.sessions"))
		require.ElementsMatch(t, []string{"live"}, loadIDs(t, pool, "tokens"))
		require.ElementsMatch(t, []string{"expired"}, loadIDs(t, pool, "untouched"))
	})

	t.Run("a failing table does not stop the others", func(t *testing.T) {
		deleted, deleteErr := cleaner.NewCleaner(pool, []string{"missing", "later"}).DeleteExpiredRows(t.Context())
		require.ErrorContains(t, deleteErr, "missing")
		require.Equal(t, int64(1), deleted)
		require.Empty(t, loadIDs(t, pool, "later"))
	})

	t.Run("keeps its own copy of the tables", func(t *testing.T) {
		tables := []string{"untouched"}
		tableCleaner := cleaner.NewCleaner(pool, tables)
		tables[0] = "missing"

		_, deleteErr := tableCleaner.DeleteExpiredRows(t.Context())
		require.NoError(t, deleteErr)
		require.Empty(t, loadIDs(t, pool, "untouched"))
	})

	t.Run("run cleanups until the context ends", func(t *testing.T) {
		_, insertErr := pool.Exec(t.Context(), `INSERT INTO tokens VALUES ('expired-again', now() - interval '1 minute')`)
		require.NoError(t, insertErr)
		cleanupCtx, cancel := context.WithTimeout(t.Context(), 300*time.Millisecond)
		defer cancel()
		cleaner.NewCleaner(pool, []string{"tokens"}).RunCleanups(cleanupCtx, 50*time.Millisecond)
		require.ElementsMatch(t, []string{"live"}, loadIDs(t, pool, "tokens"))
	})
}

func loadIDs(t *testing.T, pool *pgxpool.Pool, table string) []string {
	t.Helper()
	identifier := pgx.Identifier(strings.Split(table, "."))
	rows, err := pool.Query(t.Context(), `SELECT id FROM `+identifier.Sanitize())
	require.NoError(t, err)
	ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
	require.NoError(t, err)
	return ids
}
