# Changelog

Changes since the last release, grouped under `### Added`, `### Changed`,
`### Fixed` and `### Security`. `task release` publishes this section as the
release notes (annotation of the core tag and the GitHub release) and empties
it; notes of past releases live on the GitHub releases page.

## [Unreleased]

### Added

- `theme.System`: a third theme that leaves the choice to the browser
  (`prefers-color-scheme`). The cookie, `?theme=system`, `theme.Attach` and
  `theme.WithDefaultTheme` accept it; `theme.IsSystem` reports it.

### Changed

- `theme.IsDark` is false for `theme.System`; the server cannot tell which
  scheme the browser picks.
- The Turnstile widget gets the theme `auto` when the theme is
  `theme.System`.
