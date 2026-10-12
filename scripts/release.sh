#!/bin/sh
set -eu

root=github.com/daarxwalker/enjin
first_wave="enjintest app/openapi mail/ses google"
second_wave="bus/nats filestore/minio"
untagged="examples"
version="${1:-}"
dry_run="${DRY_RUN:-}"
resume="${RESUME:-}"

cd "$(dirname "$0")/.."

fail() {
	echo "release: $*" >&2
	exit 1
}

run() {
	echo "+ $*"
	if [ -z "$dry_run" ]; then
		"$@"
	fi
}

echo "$version" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$' ||
	fail "version must look like v1.2.3 or v1.2.3-rc.1, got '$version'"

[ "$(git rev-parse --show-toplevel)" = "$(pwd -P)" ] || fail "run from the root of the enjin repository, not from a monorepo"
case "$(git remote get-url origin)" in
*daarxwalker/enjin*) ;;
*) fail "origin is not the daarxwalker/enjin repository" ;;
esac
[ "$(git rev-parse --abbrev-ref HEAD)" = "main" ] || fail "release from the main branch"
[ -z "$(git status --porcelain)" ] || fail "the working tree is not clean"
git fetch --quiet --tags origin
[ "$(git rev-parse HEAD)" = "$(git rev-parse origin/main)" ] || fail "main is not in sync with origin/main"
for module in . $first_wave $second_wave; do
	tag="$version"
	[ "$module" = "." ] || tag="$module/$version"
	if git rev-parse -q --verify "refs/tags/$tag" >/dev/null && [ -z "$resume" ]; then
		fail "tag $tag already exists; finish an interrupted release with: task release:resume VERSION=$version"
	fi
done
if command -v gh >/dev/null 2>&1; then
	gh auth status >/dev/null 2>&1 || fail "gh is installed but not logged in; run: gh auth login"
fi

notes="$(mktemp)"
trap 'rm -f "$notes"' EXIT
grep -q '^## \[Unreleased\]$' CHANGELOG.md || fail "CHANGELOG.md has no '## [Unreleased]' section"
sed -n '/^## \[Unreleased\]$/,$p' CHANGELOG.md | tail -n +2 >"$notes"
core_tagged=""
git rev-parse -q --verify "refs/tags/$version" >/dev/null && core_tagged=1
if grep -q '[^[:space:]]' "$notes"; then
	echo "release: notes for $version ($(grep -c '' "$notes") lines from CHANGELOG.md)"
	changelog_used=1
elif [ -n "$core_tagged" ]; then
	git tag -l --format='%(contents)' "$version" >"$notes"
	echo "release: CHANGELOG.md is already reset, notes for $version come from the tag annotation"
	changelog_used=""
else
	fail "CHANGELOG.md has no changes under '## [Unreleased]'"
fi

if [ -n "$core_tagged" ]; then
	echo "release: core tag $version already exists, skipping"
else
	echo "release: tagging the core $version"
	run git tag -a --cleanup=verbatim "$version" -F "$notes"
	run git push origin "$version"
fi
if ! command -v gh >/dev/null 2>&1; then
	echo "release: gh is not installed, the notes stay in the annotation of tag $version"
elif gh release view "$version" >/dev/null 2>&1; then
	echo "release: GitHub release $version already exists, skipping"
else
	run gh release create "$version" --verify-tag --title "$version" --notes-file "$notes"
fi

if [ -n "$changelog_used" ]; then
	echo "+ reset CHANGELOG.md to an empty '## [Unreleased]' section"
	if [ -z "$dry_run" ]; then
		sed '/^## \[Unreleased\]$/q' CHANGELOG.md >CHANGELOG.md.tmp
		mv CHANGELOG.md.tmp CHANGELOG.md
	fi
fi

for wave in "$first_wave" "$second_wave $untagged"; do
	for module in $wave; do
		for dependency in "$root" "$root/enjintest" "$root/app/openapi"; do
			if grep -Eq "^[[:space:]]*(require[[:space:]]+)?$dependency v" "$module/go.mod"; then
				run go mod edit -require="$dependency@$version" "$module/go.mod"
			fi
		done
		echo "+ (cd $module && GOWORK=off GOPROXY=direct go mod tidy)"
		if [ -z "$dry_run" ]; then
			(cd "$module" && GOWORK=off GOPROXY=direct go mod tidy)
		fi
	done
	if [ -n "$dry_run" ] || [ -n "$(git status --porcelain)" ]; then
		run git commit -am "release $version: $wave"
		run git push origin main
	fi
	for module in $wave; do
		case " $untagged " in
		*" $module "*) continue ;;
		esac
		if git rev-parse -q --verify "refs/tags/$module/$version" >/dev/null; then
			echo "release: tag $module/$version already exists, skipping"
			continue
		fi
		run git tag -a "$module/$version" -m "$module/$version"
		run git push origin "$module/$version"
	done
done

for module in . $first_wave $second_wave; do
	path="$root"
	[ "$module" = "." ] || path="$root/$module"
	echo "+ GOPROXY=https://proxy.golang.org GOWORK=off go list -m $path@$version"
	if [ -z "$dry_run" ]; then
		GOPROXY=https://proxy.golang.org GOWORK=off go list -m "$path@$version" ||
			echo "release: the proxy does not list $path@$version yet, it will pick it up on the first go get" >&2
	fi
done

if [ -n "$dry_run" ]; then
	echo "release: dry run finished, nothing was tagged, committed or pushed"
else
	echo "release: done, $version is tagged for the core and every nested module"
fi
