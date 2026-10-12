package enjintest

import (
	"io/fs"
	"testing"
)

type config struct {
	migrationsFS fs.FS
	version      string
	port         int
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

func skipShort(t testing.TB) {
	t.Helper()
	if testing.Short() {
		t.Skip("enjintest: docker skipped in short mode")
	}
}
