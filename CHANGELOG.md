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
