# Changelog

Changes since the last release, grouped under `### Added`, `### Changed`,
`### Fixed` and `### Security`. `task release` publishes this section as the
release notes (annotation of the core tag and the GitHub release) and empties
it; notes of past releases live on the GitHub releases page.

## [Unreleased]

### Added

- Shared driver test suites `infra/cache/cachetest`, `web/csrf/csrftest`,
  `lock/locktest` and `filestore/filestoretest` (`RunDriverSuite`); the Redis
  and MinIO drivers run them.
- `internal/testredis.NewWithServer` returns the miniredis server for
  `FastForward`.
- Postgres drivers next to the Redis ones, so an app runs without
  Redis/Valkey: `cache.NewPostgresDriver` (with `DeleteByPattern`),
  `csrf.NewPostgresDriver` and `lock.NewPostgresDriver` on UNLOGGED tables
  from the migration `infra/db/migrations/keyvalue.FS`.
- `infra/db/cleaner` deletes expired rows from any tables with an
  `expires_at` column (`keyvalue.Tables()` for the drivers above).
- `enjintest.WithMigrationsFSTable` runs another set of goose migrations with
  its own version table, e.g. `keyvalue.FS` next to the application ones.
- `cachetest` checks that `Increment` rejects values Redis rejects (`" 5"`,
  `"5.0"`, empty).
