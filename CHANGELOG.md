# Changelog

Changes since the last release, grouped under `### Added`, `### Changed`,
`### Fixed` and `### Security`. `task release` publishes this section as the
release notes (annotation of the core tag and the GitHub release) and empties
it; notes of past releases live on the GitHub releases page.

## [Unreleased]

### Added

- `limit.WithSessions`: `Load` and `Verify` count requests with a session too.
  Without it they still skip them, so `Consume` after `Load` returns
  `limit.ErrNotLoaded` for a request with a session.
