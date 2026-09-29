#!/bin/sh
set -eu

version=${1:?Usage: scripts/build-release.sh v0.1.0}
case "$version" in v[0-9]*) ;; *) printf 'Expected a version tag such as v0.1.0\n' >&2; exit 1 ;; esac
case "$version" in *[!a-zA-Z0-9._-]*) printf 'Invalid version tag\n' >&2; exit 1 ;; esac

cd "$(dirname "$0")/.."
mkdir -p dist
release_dir=$(mktemp -d "${TMPDIR:-/tmp}/remove-shit-release.XXXXXX")
trap 'rm -rf "$release_dir"' 0
trap 'exit 1' HUP INT TERM

for arch in arm64 amd64; do
    CGO_ENABLED=0 GOOS=darwin GOARCH="$arch" go build -trimpath -ldflags="-s -w -X main.version=$version" -o "$release_dir/remove-shit" .
    COPYFILE_DISABLE=1 tar -czf "dist/remove-shit_darwin_${arch}.tar.gz" -C "$release_dir" remove-shit
done
(cd dist && shasum -a 256 remove-shit_darwin_arm64.tar.gz remove-shit_darwin_amd64.tar.gz > checksums.txt)
printf 'Built macOS releases for Apple Silicon and Intel in dist/\n'
