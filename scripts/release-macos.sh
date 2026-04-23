#!/bin/sh

set -eu

REPO_ROOT=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
TMP_DIR=$(mktemp -d "${TMPDIR:-/tmp}/openmigrate-release.XXXXXX")
VERSION=${VERSION:-$(git -C "$REPO_ROOT" describe --tags --always --dirty 2>/dev/null || echo dev)}
COMMIT=${COMMIT:-$(git -C "$REPO_ROOT" rev-parse --short HEAD 2>/dev/null || echo none)}
BUILD_DATE=${BUILD_DATE:-$(date -u +"%Y-%m-%dT%H:%M:%SZ")}
DIST_ROOT=${DIST_ROOT:-"$REPO_ROOT/dist"}
DIST_DIR=${DIST_DIR:-"$DIST_ROOT/$VERSION"}

cleanup() {
	rm -rf "$TMP_DIR"
}
trap cleanup EXIT INT TERM

cd "$REPO_ROOT"

mkdir -p "$DIST_DIR"

LDFLAGS="-s -w -X github.com/openmigrate/openmigrate/internal/buildinfo.Version=$VERSION -X github.com/openmigrate/openmigrate/internal/buildinfo.Commit=$COMMIT -X github.com/openmigrate/openmigrate/internal/buildinfo.BuildDate=$BUILD_DATE"

CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags "$LDFLAGS" -o "$TMP_DIR/openmigrate_darwin_amd64" ./cmd/openmigrate
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags "$LDFLAGS" -o "$TMP_DIR/openmigrate_darwin_arm64" ./cmd/openmigrate

lipo -create -output "$TMP_DIR/openmigrate" "$TMP_DIR/openmigrate_darwin_amd64" "$TMP_DIR/openmigrate_darwin_arm64"

if [ -n "${APPLE_SIGN_IDENTITY:-}" ]; then
	codesign --force --options runtime --timestamp --sign "$APPLE_SIGN_IDENTITY" "$TMP_DIR/openmigrate"
fi

PKG_DIR="$TMP_DIR/openmigrate_${VERSION}_darwin_universal"
ARCHIVE_PATH="$DIST_DIR/openmigrate_${VERSION}_darwin_universal.tar.gz"
CHECKSUM_PATH="$DIST_DIR/checksums.txt"

mkdir -p "$PKG_DIR"
cp "$TMP_DIR/openmigrate" "$PKG_DIR/openmigrate"
tar -czf "$ARCHIVE_PATH" -C "$TMP_DIR" "$(basename "$PKG_DIR")"

(
	cd "$DIST_DIR"
	shasum -a 256 "$(basename "$ARCHIVE_PATH")" > "$(basename "$CHECKSUM_PATH")"
)

printf 'created %s\n' "$ARCHIVE_PATH"
printf 'created %s\n' "$CHECKSUM_PATH"
