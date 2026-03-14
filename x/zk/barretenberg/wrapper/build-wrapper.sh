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
#   ./build-wrapper.sh --platform linux_amd64|linux_arm64|darwin_amd64|darwin_arm64
#
# Supported platforms: linux_amd64, linux_arm64, darwin_amd64, darwin_arm64
#
# Prerequisites:
#   - clang++ with C++20 support (honours $CXX; defaults to clang++)
#     Linux targets require clang++ (not g++) because libbb-external.a is built by
#     Aztec with Zig/libc++ and its symbols use the std::__1 ABI — incompatible with
#     g++/libstdc++. The -stdlib=libc++ and --target=<triple> flags are injected
#     automatically for linux targets when CXX contains "clang".
#     Darwin cross-compilation: set CXX=o64-clang++ (darwin/amd64), CXX=oa64-clang++ (darwin/arm64)
#   - curl
#   - ar or llvm-ar (honours $AR; defaults to ar)
#     Cross-compilation from Linux: set AR=llvm-ar (handles Darwin Mach-O archives)
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
        # libbb-external.a is built by Aztec with Zig's libc++ (std::__1 ABI).
        # Must link libc++/libc++abi instead of libstdc++ to satisfy those symbols,
        # and compile the wrapper with clang++ -stdlib=libc++ to match function
        # signatures (std::__1::vector vs std::vector have different manglings).
        EXTRA_LDFLAGS="-lc++ -lc++abi -lm -lpthread"
        LINUX_CROSS_TARGET="--target=x86_64-linux-gnu"
        ;;
    linux_arm64)
        AZTEC_ARCH="arm64"
        AZTEC_OS="linux"
        EXTRA_LDFLAGS="-lc++ -lc++abi -lm -lpthread"
        LINUX_CROSS_TARGET="--target=aarch64-linux-gnu"
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
        DARWIN_TARGET="-mmacosx-version-min=11.0"
        ;;
    *)
        echo "ERROR: unsupported platform '$PLATFORM'." >&2
        echo "  Supported platforms: linux_amd64, linux_arm64, darwin_amd64, darwin_arm64" >&2
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
    -fvisibility-inlines-hidden
    -I "$MSGPACK_DIR/include"                      # msgpack-c headers (barretenberg external dep)
    -I "${STUBS_DIR}"                              # macro-only stubs (e.g. tracy)
    -I "$HEADERS_DIR/barretenberg/cpp/src"
    -I "$INCLUDE_DIR"
    -DBB_VERSION="\"$BB_AZTEC_TAG\""
    -c "$SCRIPT_DIR/barretenberg_wrapper.cpp"
    -o "$WRAPPER_O"
)

# Apply Darwin-specific target flags (version pin for both darwin_amd64 and darwin_arm64)
if [[ "$AZTEC_OS" == "darwin" && -n "$DARWIN_TARGET" ]]; then
    CLANG_FLAGS=($DARWIN_TARGET "${CLANG_FLAGS[@]}")
fi

# Apply Linux-specific flags when using clang++ (required to match libbb-external.a's libc++ ABI):
#   --target=<triple>  : explicit cross-compilation target so a native clang++ correctly
#                        emits arm64 or amd64 code even when running on a different arch
#   -stdlib=libc++     : use libc++ (std::__1 ABI) matching the Zig-built libbb-external.a;
#                        without this, clang++ defaults to libstdc++ on Linux and symbol
#                        mangling diverges at every std:: type in barretenberg function sigs
if [[ "$AZTEC_OS" == "linux" && "${CXX:-clang++}" == *clang* ]]; then
    CLANG_FLAGS=($LINUX_CROSS_TARGET -stdlib=libc++ "${CLANG_FLAGS[@]}")
fi

${CXX:-clang++} "${CLANG_FLAGS[@]}"
echo "  Compiled: $WRAPPER_O"

# ── Step 4: Merge wrapper.o + libbb-external.a → libbarretenberg.a ───────────
echo ""
echo "▶ Step 4: Merging into libbarretenberg.a..."
mkdir -p "$LIB_DIR"
OUTPUT_A="$LIB_DIR/libbarretenberg.a"

# Append approach: copy the Aztec pre-built archive as-is, then add our wrapper
# object on top. This avoids the extract+repack strategy, which silently loses
# symbols when the archive contains duplicate member basenames — a known property
# of libbb-external.a (barretenberg has multiple source files with the same name
# across different subdirectories). ar rcs appends to an existing archive without
# touching existing members. llvm-ar handles both ELF and Mach-O archives.
cp "$BB_EXTERNAL_A" "$OUTPUT_A"
${AR:-ar} rcs "$OUTPUT_A" "$WRAPPER_O"

echo "  Output: $(du -sh "$OUTPUT_A" | cut -f1) $OUTPUT_A"

echo ""
echo "✅ Done! libbarretenberg.a built for $PLATFORM (Aztec $BB_AZTEC_TAG)"
echo ""
echo "Next steps:"
echo "  go build ./x/zk/barretenberg/..."
echo "  go test  ./x/zk/barretenberg/..."
