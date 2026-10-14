# Changelog

Changes since the last release, grouped under `### Added`, `### Changed`,
`### Fixed` and `### Security`. `task release` publishes this section as the
release notes (annotation of the core tag and the GitHub release) and empties
it; notes of past releases live on the GitHub releases page.

## [Unreleased]

### Added

- `abuse.WithRecorder`: the guard hands every delay and block to the app as an `abuse.Event` (IP, client, user, request, policy, rule, retry after); `abuse.RulePenalty` marks an escalated IP block.
