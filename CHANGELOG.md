# Changelog

Changes since the last release, grouped under `### Added`, `### Changed`,
`### Fixed` and `### Security`. `task release` publishes this section as the
release notes (annotation of the core tag and the GitHub release) and empties
it; notes of past releases live on the GitHub releases page.

## [Unreleased]

### Added

- `filestore/disk`: filestore Driver on the local disk through `os.Root`,
  with atomic writes, metadata in `.meta/`, `ErrInvalidKey` for keys that
  would leave the directory and `ErrSizeMismatch` when the body differs from
  the given size.
- `filestore/webdav`: filestore Driver for WebDAV servers, a client over
  `net/http` (PROPFIND, GET, PUT, MKCOL, DELETE) with basic or bearer auth,
  parent collections created on Put and `ErrInvalidKey` /
  `ErrUnexpectedStatus` sentinels.
- `filestoretest.Subject.DerivesContentType` also relaxes the put-and-get and
  stat checks to the media type, since such stores add parameters
  (`text/plain; charset=utf-8`).
- `abuse.WithBotVerifierLogger` sets the logger of `BotVerifier`.

### Changed

- Every returned error is wrapped with the operation (and key, subject or
  path) that failed: the Redis and Postgres drivers of cache, csrf and lock,
  `infra/db/cleaner`, `filestore` and `filestore/minio`, `bus` and `bus/nats`,
  `app`, `app/openapi`, `abuse`, `limit`, `mail`, `mail/brevo`, `session`,
  `sse`, `web/antibot`, `web/assets` and `google`.
- Sentinel errors (`bus.ErrClosed`, `bus.ErrFull`, `bus.ErrInUse`,
  `app.ErrNoRouter`, `abuse.ErrInvalidWindow`, `limit.ErrNotLoaded`,
  `brevo.ErrMultipleReplyTo`, `brevo.ErrEmbedsUnsupported`) are returned
  wrapped; match them with `errors.Is`, not `==`.
- A failing Close no longer overwrites the error of `assets.LoadManifest` and
  the Turnstile verification; both are joined.
- No error is discarded any more: the SMTP driver logs a failed Close of the
  connection or client (other than an already closed one), `BotVerifier` logs
  when its verdict cannot be loaded from or saved to the store, and the abuse
  identity keys by the full address when the network prefix cannot be built.
- `cleaner.RunCleanups` logs a failed cleanup with the error in the message and
  no longer logs every cleanup at debug level.

### Security

- `enjintest`, `bus/nats`, `filestore/minio` and `internal/integration`
  require `github.com/klauspost/compress` v1.20.2 (GO-2026-6865, panic on
  two-block frames).
