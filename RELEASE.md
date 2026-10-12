# Releasing

Enjin is one repository with several Go modules: the core at the root and the
nested modules `app/openapi`, `bus/nats`, `enjintest`, `filestore/minio`,
`google` and `mail/ses`. Each is versioned with its own tag; `examples` is never
tagged.

| Module | Tag | Requires |
|---|---|---|
| core | `vX.Y.Z` | — |
| `enjintest` | `enjintest/vX.Y.Z` | core |
| `app/openapi` | `app/openapi/vX.Y.Z` | core |
| `mail/ses` | `mail/ses/vX.Y.Z` | core |
| `google` | `google/vX.Y.Z` | — |
| `bus/nats` | `bus/nats/vX.Y.Z` | core, `enjintest` |
| `filestore/minio` | `filestore/minio/vX.Y.Z` | core, `enjintest` |

## Local workspace

`task work` (or `sh scripts/work.sh`) creates a `go.work` with every module, so
the nested modules build against the local core instead of the released one.
`go.work` is not committed.

## Release

Write every change under `## [Unreleased]` in `CHANGELOG.md` as you go. That
section is the only place release notes come from.

1. Preview the steps: `task release:dry VERSION=vX.Y.Z`. It runs the checks
   below and prints every `go mod edit`, `go mod tidy`, commit, tag and push
   without running them.
2. Release: `task release VERSION=vX.Y.Z`.

`task release` refuses to start unless the version looks like `v1.2.3`, it runs
in the root of the `daarxwalker/enjin` repository, you are on a clean `main`
that matches `origin/main`, none of the tags exists yet and `## [Unreleased]`
has some content. It then runs lint and tests and `scripts/release.sh`, which:

1. tags and pushes the core `vX.Y.Z` with the `[Unreleased]` section as the tag
   annotation, and creates a GitHub release with the same notes when `gh` is
   installed,
2. empties the `[Unreleased]` section, so the next release starts clean,
3. in `enjintest`, `app/openapi`, `mail/ses` and `google` raises the required
   core to `vX.Y.Z`, runs `GOWORK=off go mod tidy` (this writes `go.sum` from
   the tag just pushed), commits together with the emptied `CHANGELOG.md`,
   pushes and tags `<module>/vX.Y.Z`,
4. does the same for `bus/nats` and `filestore/minio` (they also require
   `enjintest`) and for `examples`, which is updated but never tagged,
5. asks the Go proxy for every new version so pkg.go.dev picks it up.

Versions follow [Semantic Versioning](https://semver.org/); until `v1.0.0` a
minor release may contain breaking changes.

Every module gets the same version on every release, which keeps the
requirements between them simple. A tag can never be moved or deleted once the
Go proxy has seen it; fix a broken release with a new patch version.

If the script stops halfway (network, proxy, `gh`), the tags it already pushed
stay. Fix the cause and run `task release:resume VERSION=vX.Y.Z`: it repeats
the same steps, skips every tag and the GitHub release that already exist, and
takes the notes from `CHANGELOG.md`, or from the annotation of the core tag when
the changelog is already reset. Preview it with
`DRY_RUN=1 task release:resume VERSION=vX.Y.Z`.

When `gh` is installed, it has to be logged in (`gh auth login`); the release
checks that before it tags anything.
