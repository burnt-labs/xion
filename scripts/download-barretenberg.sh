#!/usr/bin/env bash

set -euo pipefail

if [[ $# -lt 3 || $# -gt 4 ]]; then
    echo "Usage: $0 <goos> <goarch> <destination> [gnu|musl]" >&2
    exit 1
fi

GOOS_TARGET="$1"
GOARCH_TARGET="$2"
DESTINATION="$3"
LIBC_VARIANT="${4:-gnu}"

if [[ "$LIBC_VARIANT" != "gnu" && "$LIBC_VARIANT" != "musl" ]]; then
    echo "Unsupported libc variant: $LIBC_VARIANT" >&2
    exit 1
fi

BB_VERSION="$(awk '$1 == "github.com/burnt-labs/barretenberg-go" { print $2; exit }' go.mod)"
if [[ -z "$BB_VERSION" ]]; then
    echo "github.com/burnt-labs/barretenberg-go is not present in go.mod" >&2
    exit 1
fi

PLATFORM="${GOOS_TARGET}_${GOARCH_TARGET}"
if [[ "$PLATFORM" == "linux_arm64" && "$LIBC_VARIANT" == "musl" ]]; then
    PLATFORM="linux_arm64_musl"
fi

ASSET="libbarretenberg_${PLATFORM}.a"
RELEASE_URL="${BARRETENBERG_RELEASE_URL:-https://github.com/burnt-labs/barretenberg-go/releases/download/${BB_VERSION}}"
DESTINATION_DIR="$(dirname "$DESTINATION")"
mkdir -p "$DESTINATION_DIR"

WORK_DIR="$(mktemp -d "$DESTINATION_DIR/.barretenberg-download.XXXXXX")"
trap 'rm -rf "$WORK_DIR"' EXIT

sha256_file() {
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$1" | awk '{ print $1 }'
    else
        shasum -a 256 "$1" | awk '{ print $1 }'
    fi
}

curl --fail --location --retry 5 --retry-all-errors \
    --output "$WORK_DIR/checksums.txt" "$RELEASE_URL/checksums.txt"
EXPECTED_SHA256="$(awk -v asset="$ASSET" '$2 == asset { print $1; exit }' "$WORK_DIR/checksums.txt")"
if [[ ! "$EXPECTED_SHA256" =~ ^[0-9a-f]{64}$ ]]; then
    echo "$ASSET is missing from $BB_VERSION checksums.txt" >&2
    exit 1
fi

if [[ -f "$DESTINATION" ]]; then
    CURRENT_SHA256="$(sha256_file "$DESTINATION")"
    if [[ "$CURRENT_SHA256" == "$EXPECTED_SHA256" ]] && ar t "$DESTINATION" >/dev/null; then
        if [[ "$PLATFORM" == "linux_arm64_musl" && "$(basename "$DESTINATION_DIR")" == "linux_arm64" ]]; then
            MUSL_DESTINATION_DIR="$(dirname "$DESTINATION_DIR")/linux_arm64_musl"
            MUSL_DESTINATION="$MUSL_DESTINATION_DIR/libbarretenberg.a"
            mkdir -p "$MUSL_DESTINATION_DIR"
            if [[ ! -f "$MUSL_DESTINATION" ]] || \
                [[ "$(sha256_file "$MUSL_DESTINATION")" != "$EXPECTED_SHA256" ]] || \
                ! ar t "$MUSL_DESTINATION" >/dev/null; then
                MUSL_INSTALL_CANDIDATE="$MUSL_DESTINATION_DIR/.libbarretenberg.a.$$.tmp"
                cp "$DESTINATION" "$MUSL_INSTALL_CANDIDATE"
                mv -f "$MUSL_INSTALL_CANDIDATE" "$MUSL_DESTINATION"
            fi
        fi
        echo "$ASSET already installed at $DESTINATION"
        exit 0
    fi
fi

DOWNLOADED_ARCHIVE="$WORK_DIR/$ASSET"
curl --fail --location --retry 5 --retry-all-errors \
    --output "$DOWNLOADED_ARCHIVE" "$RELEASE_URL/$ASSET"
if [[ "$(sha256_file "$DOWNLOADED_ARCHIVE")" != "$EXPECTED_SHA256" ]]; then
    echo "SHA-256 mismatch for $ASSET" >&2
    exit 1
fi

if ! ar t "$DOWNLOADED_ARCHIVE" >/dev/null; then
    echo "$ASSET is not a valid static archive" >&2
    exit 1
fi

chmod -R u+w "$DESTINATION_DIR" 2>/dev/null || true
INSTALL_CANDIDATE="$DESTINATION_DIR/.libbarretenberg.a.$$.tmp"
cp "$DOWNLOADED_ARCHIVE" "$INSTALL_CANDIDATE"
mv -f "$INSTALL_CANDIDATE" "$DESTINATION"

# barretenberg-go releases that know about the muslc build tag use a sibling
# linux_arm64_musl directory. Populate it as well so this downloader works
# during the transition from the legacy linux_arm64 link file.
if [[ "$PLATFORM" == "linux_arm64_musl" && "$(basename "$DESTINATION_DIR")" == "linux_arm64" ]]; then
    MUSL_DESTINATION_DIR="$(dirname "$DESTINATION_DIR")/linux_arm64_musl"
    MUSL_DESTINATION="$MUSL_DESTINATION_DIR/libbarretenberg.a"
    mkdir -p "$MUSL_DESTINATION_DIR"
    MUSL_INSTALL_CANDIDATE="$MUSL_DESTINATION_DIR/.libbarretenberg.a.$$.tmp"
    cp "$DOWNLOADED_ARCHIVE" "$MUSL_INSTALL_CANDIDATE"
    mv -f "$MUSL_INSTALL_CANDIDATE" "$MUSL_DESTINATION"
fi

echo "Installed $ASSET ($EXPECTED_SHA256) at $DESTINATION"
