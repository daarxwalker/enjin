# Changelog

Changes since the last release, grouped under `### Added`, `### Changed`,
`### Fixed` and `### Security`. `task release` publishes this section as the
release notes (annotation of the core tag and the GitHub release) and empties
it; notes of past releases live on the GitHub releases page.

## [Unreleased]

### Added

- `app.WithLogger` sets the logger of the engine; it logs start, stop and a
  missing ops token through it instead of the standard `log` package.
- `cleaner.WithInterval` sets how often `RunCleanups` runs (default one
  minute; zero or negative is ignored).
- `filestoretest` checks that the key of a directory reads as missing and that
  deleting it removes nothing.

### Changed

- `cleaner.RunCleanups(ctx)` no longer takes the interval; use
  `cleaner.WithInterval`. A non-positive interval used to panic.
- `webdav.WithHTTPClient(nil)` keeps the default client instead of panicking.

### Fixed

- WebDAV Put waits until net/http releases the body before it seeks back for a
  retry; the transport may still read it after the response.
- WebDAV Get of a key that is a collection reports a missing file instead of
  an error (405) or, after a redirect, the HTML listing as the file.
- WebDAV Stat and Delete follow a redirect to the address with a slash as a
  collection; a followed redirect used to turn PROPFIND into GET.
- MinIO Delete of a key that is only a directory prefix of other objects
  succeeds on stores that refuse to remove it (SeaweedFS answered with an
  internal error).
- `google.NewSAClient` returns an error instead of panicking when
  `http.DefaultTransport` is no `*http.Transport`.
- `web/head` reads its context value with a nil check and a checked type
  assertion; `mail/brevo` keeps the JSON error when an error body cannot be
  decoded; every early return with an error returns `zero` instead of a
  literal; tests no longer discard errors.
