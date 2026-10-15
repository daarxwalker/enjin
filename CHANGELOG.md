# Changelog

Changes since the last release, grouped under `### Added`, `### Changed`,
`### Fixed` and `### Security`. `task release` publishes this section as the
release notes (annotation of the core tag and the GitHub release) and empties
it; notes of past releases live on the GitHub releases page.

## [Unreleased]

### Added

- `mail.NewSMTPDriver` takes options: `WithSMTPEncryption` (`SMTPEncryptionAuto`,
  `None`, `StartTLS`, `ImplicitTLS`), `WithSMTPHelo`, `WithSMTPTLSConfig`,
  `WithSMTPOAuth2` (AUTH XOAUTH2 with a token fetched per send),
  `WithSMTPTimeout` and `WithSMTPLogger`.
- The SMTP driver authenticates with AUTH LOGIN and CRAM-MD5 besides PLAIN and
  picks what the server offers.

### Changed

- The SMTP driver skips authentication when the username is empty (IP relays,
  Mailpit, MailHog).
- The SMTP driver greets with `EHLO <os.Hostname()>` instead of `localhost`.
- Port 465 uses implicit TLS by default.
- The SMTP timeout covers the whole send, dial and TLS handshake included.

### Fixed

- A failed QUIT after the server accepted the message no longer fails the
  send, so a retry does not deliver the mail twice; the failure is logged.
