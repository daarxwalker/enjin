# Changelog

Changes since the last release, grouped under `### Added`, `### Changed`,
`### Fixed` and `### Security`. `task release` publishes this section as the
release notes (annotation of the core tag and the GitHub release) and empties
it; notes of past releases live on the GitHub releases page.

## [Unreleased]

### Added

- `mail/brevo`: Brevo transactional API driver (`brevo.NewDriver(apiKey)`, `WithEndpoint`, `WithHTTPClient`). Brevo has no inline images and one Reply-To, so messages with embeds or several Reply-To addresses fail with `brevo.ErrEmbedsUnsupported` / `brevo.ErrMultipleReplyTo`; send those through the Brevo SMTP relay with `mail.NewSMTPDriver`.
