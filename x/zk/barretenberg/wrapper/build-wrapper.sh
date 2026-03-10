#!/usr/bin/env bash
#
# build-wrapper.sh — Build libbarretenberg.a for the specified platform
#
# PINNED VERSION: Aztec aztec-packages v4.0.4
#   Barretenberg release: v4.0.4
#   Aztec release page: https://github.com/AztecProtocol/aztec-packages/releases/tag/v4.0.4
#   Static lib assets (each tarball contains only libbb-external.a):
#     barretenberg-static-amd64-linux.tar.gz  → libbb-external.a
#     barretenberg-static-amd64-darwin.tar.gz → libbb-external.a
#     barretenberg-static-arm64-darwin.tar.gz → libbb-external.a
#
# To update this to a new Aztec version:
#   1. Change BB_AZTEC_TAG below (and the header above)
#   2. Verify the new release has barretenberg-static-{arch}-{os}.tar.gz assets
#   3. Check barretenberg_wrapper.cpp for API compatibility with the new version
#   4. Re-run this script to regenerate lib/{platform}/libbarretenberg.a
#   5. Run tests: go test -tags barretenberg_stub ./x/zk/barretenberg/...
#
# Usage:
#   ./build-wrapper.sh --platform linux_amd64|darwin_amd64|darwin_arm64
#
# Supported platforms: linux_amd64, darwin_amd64, darwin_arm64
# (linux_arm64 is not officially supported — no pre-built static lib provided by Aztec)
#
# Prerequisites:
#   - clang++ with C++20 support
#   - curl
#   - ar (Linux) or libtool (Darwin)
#   - git (for sparse-checkout of headers)

set -euo pipefail

# ─── PINNED VERSION ───────────────────────────────────────────────────────────
# This is the single source of truth for the barretenberg version being built.
# All download URLs and the BB_VERSION compile flag derive from this value.
readonly BB_AZTEC_TAG="v4.0.4"
readonly BB_AZTEC_REPO="https://github.com/AztecProtocol/aztec-packages"
# ──────────────────────────────────────────────────────────────────────────────

PLATFORM=""

usage() {
    echo "Usage: $0 --platform linux_amd64|darwin_amd64|darwin_arm64" >&2
    exit 1
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --platform)
            PLATFORM="$2"
            shift 2
            ;;
        *)
            usage
            ;;
    esac
done

if [[ -z "$PLATFORM" ]]; then
    usage
fi

case "$PLATFORM" in
    linux_amd64)
        AZTEC_ARCH="amd64"
        AZTEC_OS="linux"
        EXTRA_LDFLAGS="-lstdc++ -lm -lpthread"
        ;;
    darwin_amd64)
        AZTEC_ARCH="amd64"
        AZTEC_OS="darwin"
        EXTRA_LDFLAGS="-lc++ -lm"
        DARWIN_TARGET="-target x86_64-apple-macos10.15"
        ;;
    darwin_arm64)
        AZTEC_ARCH="arm64"
        AZTEC_OS="darwin"
        EXTRA_LDFLAGS="-lc++ -lm"
        DARWIN_TARGET=""
        ;;
    *)
        echo "Unsupported platform: $PLATFORM" >&2
        echo "Supported platforms: linux_amd64, darwin_amd64, darwin_arm64" >&2
        exit 1
        ;;
esac

# Resolve paths relative to this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BARRETENBERG_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
REPO_ROOT="$(cd "$BARRETENBERG_DIR/../../../.." && pwd)"
LIB_DIR="$BARRETENBERG_DIR/lib/$PLATFORM"
INCLUDE_DIR="$BARRETENBERG_DIR/include"

TARBALL_URL="${BB_AZTEC_REPO}/releases/download/${BB_AZTEC_TAG}/barretenberg-static-${AZTEC_ARCH}-${AZTEC_OS}.tar.gz"

echo "═══════════════════════════════════════════════════════════════"
echo "  Building libbarretenberg.a"
echo "  Platform:    $PLATFORM"
echo "  Aztec tag:   $BB_AZTEC_TAG"
echo "  Output:      $LIB_DIR/libbarretenberg.a"
echo "═══════════════════════════════════════════════════════════════"

WORK_DIR="$(mktemp -d)"
trap 'rm -rf "$WORK_DIR"' EXIT

# ── Step 1: Download libbb-external.a ────────────────────────────────────────
echo ""
echo "▶ Step 1: Downloading libbb-external.a from Aztec release..."
echo "  URL: $TARBALL_URL"
curl -fsSL "$TARBALL_URL" | tar -xz -C "$WORK_DIR"

BB_EXTERNAL_A="$WORK_DIR/libbb-external.a"
if [[ ! -f "$BB_EXTERNAL_A" ]]; then
    echo "ERROR: libbb-external.a not found after extracting tarball." >&2
    echo "  Contents of work dir:" >&2
    ls -la "$WORK_DIR" >&2
    exit 1
fi
echo "  Downloaded: $(du -sh "$BB_EXTERNAL_A" | cut -f1) libbb-external.a"

# ── Step 2: Sparse-checkout barretenberg headers from aztec-packages ─────────
echo ""
echo "▶ Step 2: Fetching barretenberg headers (sparse checkout)..."
HEADERS_DIR="$WORK_DIR/az-src"
git clone \
    --filter=blob:none \
    --sparse \
    --depth=1 \
    --branch="$BB_AZTEC_TAG" \
    "$BB_AZTEC_REPO.git" \
    "$HEADERS_DIR"
git -C "$HEADERS_DIR" sparse-checkout set barretenberg/cpp/src
echo "  Headers at: $HEADERS_DIR/barretenberg/cpp/src"

# ── Step 2b: Create stubs for external headers not in the barretenberg source tree ──
echo ""
echo "▶ Step 2b: Creating external-dependency stubs..."
STUBS_DIR="$WORK_DIR/stubs"
mkdir -p "$STUBS_DIR/tracy"
# Tracy.hpp: when TRACY_ENABLE is not defined, all macros are no-ops.
# This matches Tracy's own behaviour; no Tracy symbols are linked in libbb-external.a
# (barretenberg is built with ENABLE_TRACY=OFF by default).
cat > "$STUBS_DIR/tracy/Tracy.hpp" << 'TRACY_EOF'
#pragma once
// Stub Tracy.hpp — all macros are no-ops when TRACY_ENABLE is not defined.
// Tracy is a profiler; barretenberg compiles without ENABLE_TRACY by default,
// so libbb-external.a contains no Tracy call-sites requiring link-time resolution.
#ifndef TRACY_ENABLE
#  define TracyAlloc(ptr, size)
#  define TracyFree(ptr)
#  define TracyAllocS(ptr, size, depth)
#  define TracyFreeS(ptr, depth)
#  define TracyAllocN(ptr, size, name)
#  define TracyFreeN(ptr, name)
#  define TracyAllocNS(ptr, size, depth, name)
#  define TracyFreeNS(ptr, depth, name)
#  define TracySecureAlloc(ptr, size)
#  define TracySecureFree(ptr)
#  define TracySecureAllocS(ptr, size, depth)
#  define TracySecureFreeS(ptr, depth)
#  define ZoneScoped
#  define ZoneScopedN(x)
#  define ZoneScopedC(x)
#  define ZoneScopedNC(x, y)
#  define ZoneNamedN(x, name, active)
#  define FrameMark
#  define FrameMarkNamed(x)
#  define FrameMarkStart(x)
#  define FrameMarkEnd(x)
#endif
TRACY_EOF
echo "  Created stubs/tracy/Tracy.hpp"

# ── Step 2c: Download msgpack-c include headers (barretenberg external dependency) ──
echo ""
echo "▶ Step 2c: Downloading msgpack-c include headers..."
# Barretenberg v4.0.4 pins msgpack-c at this exact Aztec-fork commit.
# msgpack-c is a header-only C++ library; we only need its include/ directory.
# GitHub provides per-commit tarballs so no git clone is required.
MSGPACK_COMMIT="c0334576ed657fb3b3c49e8e61402989fb84146d"
MSGPACK_DIR="$WORK_DIR/msgpack-c"
mkdir -p "$MSGPACK_DIR"
curl -fsSL "https://github.com/AztecProtocol/msgpack-c/archive/${MSGPACK_COMMIT}.tar.gz" \
    | tar -xz -C "$MSGPACK_DIR" --strip-components=1
echo "  msgpack-c include at: $MSGPACK_DIR/include"

# ── Step 3: Compile barretenberg_wrapper.cpp ─────────────────────────────────
echo ""
echo "▶ Step 3: Compiling barretenberg_wrapper.cpp..."
WRAPPER_O="$WORK_DIR/barretenberg_wrapper.o"

CLANG_FLAGS=(
    -std=c++20
    -fPIC
    -O2
    -fvisibility=hidden
    -I "$MSGPACK_DIR/include"                      # msgpack-c headers (barretenberg external dep)
    -I "${STUBS_DIR}"                              # macro-only stubs (e.g. tracy)
    -I "$HEADERS_DIR/barretenberg/cpp/src"
    -I "$INCLUDE_DIR"
    -DBB_VERSION="\"$BB_AZTEC_TAG\""
    -c "$SCRIPT_DIR/barretenberg_wrapper.cpp"
    -o "$WRAPPER_O"
)

if [[ "$PLATFORM" == "darwin_amd64" ]]; then
    CLANG_FLAGS=($DARWIN_TARGET "${CLANG_FLAGS[@]}")
fi

clang++ "${CLANG_FLAGS[@]}"
echo "  Compiled: $WRAPPER_O"

# ── Step 4: Merge wrapper.o + libbb-external.a → libbarretenberg.a ───────────
echo ""
echo "▶ Step 4: Merging into libbarretenberg.a..."
mkdir -p "$LIB_DIR"
OUTPUT_A="$LIB_DIR/libbarretenberg.a"

if [[ "$AZTEC_OS" == "linux" ]]; then
    MERGE_DIR="$(mktemp -d)"
    trap 'rm -rf "$WORK_DIR" "$MERGE_DIR"' EXIT
    (
        cd "$MERGE_DIR"
        ar x "$BB_EXTERNAL_A"
        ar rcs "$OUTPUT_A" ./*.o "$WRAPPER_O"
    )
elif [[ "$AZTEC_OS" == "darwin" ]]; then
    if [[ "$PLATFORM" == "darwin_amd64" ]]; then
        # Cross-compile: wrap wrapper.o in a thin static lib then merge
        libtool -static -o "$OUTPUT_A" "$BB_EXTERNAL_A" "$WRAPPER_O"
    else
        libtool -static -o "$OUTPUT_A" "$BB_EXTERNAL_A" "$WRAPPER_O"
    fi
fi

echo "  Output: $(du -sh "$OUTPUT_A" | cut -f1) $OUTPUT_A"

echo ""
echo "✅ Done! libbarretenberg.a built for $PLATFORM (Aztec $BB_AZTEC_TAG)"
echo ""
echo "Next steps:"
echo "  go build ./x/zk/barretenberg/..."
echo "  go test  ./x/zk/barretenberg/..."
