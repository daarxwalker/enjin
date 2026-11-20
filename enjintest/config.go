package enjintest

import (
	"io/fs"
	"testing"
)

type config struct {
	migrationsFS    fs.FS
	version         string
	tableMigrations []tableMigrations
	port            int
}

type tableMigrations struct {
	migrationsFS fs.FS
	versionTable string
}

type Option func(*config)

func WithVersion(version string) Option {
	return func(c *config) {
		c.version = version
	}
}

func WithPort(port int) Option {
	return func(c *config) {
		c.port = port
	}
}

func WithMigrationsFS(migrationsFS fs.FS) Option {
	return func(c *config) {
		c.migrationsFS = migrationsFS
	}
}

// WithMigrationsFSTable runs one more set of goose migrations, tracked in its
// own version table, after the WithMigrationsFS ones. Library migrations such
// as infra/db/migrations/keyvalue need it: in the application version table
// their timestamp versions would block every later application migration.
func WithMigrationsFSTable(migrationsFS fs.FS, versionTable string) Option {
	return func(c *config) {
		c.tableMigrations = append(c.tableMigrations, tableMigrations{
			migrationsFS: migrationsFS,
			versionTable: versionTable,
		})
	}
}

func skipShort(t testing.TB) {
	t.Helper()
	if testing.Short() {
		t.Skip("enjintest: docker skipped in short mode")
	}
}
