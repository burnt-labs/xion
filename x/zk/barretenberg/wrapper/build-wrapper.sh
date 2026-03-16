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
#   5. Run tests: go test ./x/zk/barretenberg/...
#
# Usage:
#   ./build-wrapper.sh --platform linux_amd64|linux_arm64|darwin_amd64|darwin_arm64
#
# Supported platforms: linux_amd64, linux_arm64, darwin_amd64, darwin_arm64
#
# Prerequisites:
#   - clang++ with C++20 support (honours $CXX; defaults to clang++)
#     The stdlib used to compile the wrapper MUST match how Aztec built libbb-external.a:
#       linux/arm64  → Aztec uses Zig/libc++ (std::__1 ABI) → compile with -stdlib=libc++
#       linux/amd64  → Aztec uses g++/libstdc++ (std:: ABI) → compile without -stdlib flag
#       darwin/*     → Aztec uses clang++/libc++ (std::__1 ABI)
#     The --target=<triple> flag is injected automatically for linux when CXX contains "clang".
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
    echo "Usage: $0 --platform linux_amd64|linux_arm64|darwin_amd64|darwin_arm64" >&2
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
        # Aztec builds libbb-external.a for linux/amd64 with g++/libstdc++ (std:: ABI).
        # Compile the wrapper without -stdlib=libc++ so clang++ defaults to libstdc++,
        # keeping function signatures (std::vector, std::string, etc.) consistent.
        # CGo links with -lstdc++ accordingly (see link_linux_amd64.go).
        EXTRA_LDFLAGS="-lstdc++ -lm -lpthread"
        LINUX_CROSS_TARGET="--target=x86_64-linux-gnu"
        LINUX_STDLIB=""   # use compiler default (libstdc++) — matches Aztec's amd64 build
        # SHA256 digest from https://github.com/AztecProtocol/aztec-packages/releases/tag/v4.0.4
        EXPECTED_SHA256="7578c9fc80dec89988acd6038ff733f88ff1847eef2e06d16504357e8ad6373a"
        ;;
    linux_arm64)
        AZTEC_ARCH="arm64"
        AZTEC_OS="linux"
        # Aztec builds libbb-external.a for linux/arm64 with Zig/libc++ (std::__1 ABI).
        # Compile the wrapper with -stdlib=libc++ to match those function signatures.
        # CGo links with -lc++ accordingly (see link_linux_arm64.go).
        EXTRA_LDFLAGS="-lc++ -lm -lpthread"
        LINUX_CROSS_TARGET="--target=aarch64-linux-gnu"
        LINUX_STDLIB="-stdlib=libc++"   # match Zig/libc++ used for arm64
        # SHA256 digest from https://github.com/AztecProtocol/aztec-packages/releases/tag/v4.0.4
        EXPECTED_SHA256="1a0b3bd4b2a6dc95c7e15106cd71a161e5f47c40b2bdc06bc155f8c69e44a958"
        ;;
    darwin_amd64)
        AZTEC_ARCH="amd64"
        AZTEC_OS="darwin"
        EXTRA_LDFLAGS="-lc++ -lm"
        DARWIN_TARGET="-target x86_64-apple-macos10.15"
        # SHA256 digest from https://github.com/AztecProtocol/aztec-packages/releases/tag/v4.0.4
        EXPECTED_SHA256="20f77d04b477770f288074e4876ace2044ae265b42a290a7acc164efdb8d9d7e"
        ;;
    darwin_arm64)
        AZTEC_ARCH="arm64"
        AZTEC_OS="darwin"
        EXTRA_LDFLAGS="-lc++ -lm"
        DARWIN_TARGET="-mmacosx-version-min=11.0"
        # SHA256 digest from https://github.com/AztecProtocol/aztec-packages/releases/tag/v4.0.4
        EXPECTED_SHA256="324249ed62dc266a1d6fea9ee33b37494ea0edc80b0c848bf94f0a1b3ce77a8e"
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

# ── Step 1: Download and verify libbb-external.a ─────────────────────────────
echo ""
echo "▶ Step 1: Downloading libbb-external.a from Aztec release..."
echo "  URL: $TARBALL_URL"
TARBALL="$WORK_DIR/barretenberg-static.tar.gz"
curl -fsSL -o "$TARBALL" "$TARBALL_URL"

echo "  Verifying SHA256 checksum..."
# sha256sum on Linux; shasum -a 256 on macOS
if command -v sha256sum &>/dev/null; then
    ACTUAL_SHA256="$(sha256sum "$TARBALL" | awk '{print $1}')"
else
    ACTUAL_SHA256="$(shasum -a 256 "$TARBALL" | awk '{print $1}')"
fi
if [[ "$ACTUAL_SHA256" != "$EXPECTED_SHA256" ]]; then
    echo "ERROR: SHA256 mismatch for barretenberg-static-${AZTEC_ARCH}-${AZTEC_OS}.tar.gz" >&2
    echo "  Expected: $EXPECTED_SHA256" >&2
    echo "  Got:      $ACTUAL_SHA256" >&2
    echo "  Refusing to continue — the tarball may have been tampered with." >&2
    exit 1
fi
echo "  Checksum OK: $ACTUAL_SHA256"
tar -xz -C "$WORK_DIR" -f "$TARBALL"

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

# Apply Linux-specific flags when using clang++:
#   --target=<triple>  : explicit cross-compilation target so a native clang++ correctly
#                        emits arm64 or amd64 code even when running on a different arch
#   -stdlib=...        : stdlib selection is ARCH-SPECIFIC (see platform cases above):
#                          arm64 → -stdlib=libc++  (Aztec arm64 built with Zig/libc++)
#                          amd64 → (omitted)       (Aztec amd64 built with g++/libstdc++;
#                                                   clang++ defaults to libstdc++ on Linux)
if [[ "$AZTEC_OS" == "linux" && "${CXX:-clang++}" == *clang* ]]; then
    STDLIB_FLAGS=()
    [[ -n "${LINUX_STDLIB:-}" ]] && STDLIB_FLAGS=($LINUX_STDLIB)
    CLANG_FLAGS=($LINUX_CROSS_TARGET "${STDLIB_FLAGS[@]}" "${CLANG_FLAGS[@]}")
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
