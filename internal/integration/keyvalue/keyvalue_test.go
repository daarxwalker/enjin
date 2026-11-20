package keyvalue_test

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/daarxwalker/enjin/enjintest"
	"github.com/daarxwalker/enjin/infra/db/migrations/keyvalue"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestMigrations(t *testing.T) {
	appMigrations := fstest.MapFS{
		"00001_users.sql": {Data: []byte("-- +goose Up\nCREATE TABLE users (id text PRIMARY KEY);\n")},
	}
	pool := enjintest.NewPostgres(
		context.Background(),
		t,
		enjintest.WithMigrationsFS(appMigrations),
		enjintest.WithMigrationsFSTable(keyvalue.FS, "enjin_keyvalue_version"),
	)

	t.Run("tables are unlogged", func(t *testing.T) {
		rows, err := pool.Query(
			t.Context(),
			`SELECT relname FROM pg_class WHERE relname = ANY($1) AND relkind = 'r' AND relpersistence = 'u'`,
			keyvalue.Tables(),
		)
		require.NoError(t, err)
		unlogged, err := pgx.CollectRows(rows, pgx.RowTo[string])
		require.NoError(t, err)
		require.ElementsMatch(t, keyvalue.Tables(), unlogged)
	})

	t.Run("application migrations keep their own version table", func(t *testing.T) {
		var users, appVersions, keyValueVersions int
		err := pool.QueryRow(
			t.Context(),
			`SELECT (SELECT count(*) FROM users),
       (SELECT count(*) FROM goose_db_version WHERE version_id > 0),
       (SELECT count(*) FROM enjin_keyvalue_version WHERE version_id > 0)`,
		).Scan(&users, &appVersions, &keyValueVersions)
		require.NoError(t, err)
		require.Zero(t, users)
		require.Equal(t, 1, appVersions)
		require.Equal(t, 1, keyValueVersions)
	})
}
