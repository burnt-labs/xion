#!/usr/bin/env bash

set -euo pipefail

ZIG_VERSION="0.14.1"
ZIG_X86_64_SHA256="24aeeec8af16c381934a6cd7d95c807a8cb2cf7df9fa40d359aa884195c4716c"
ZIG_AARCH64_SHA256="f7a654acc967864f7a050ddacfaa778c7504a0eca8d2b678839c21eea47c992b"

if [[ $# -ne 1 || -z "$1" ]]; then
    echo "Usage: $0 <install-directory>" >&2
    exit 1
fi

if [[ "$(uname -s)" != "Linux" ]]; then
    echo "The pinned Zig toolchain installer supports Linux hosts only." >&2
    exit 1
fi

INSTALL_DIR="${1%/}"
case "$(uname -m)" in
    x86_64)
        ZIG_ARCH="x86_64"
        EXPECTED_SHA256="$ZIG_X86_64_SHA256"
        ;;
    aarch64|arm64)
        ZIG_ARCH="aarch64"
        EXPECTED_SHA256="$ZIG_AARCH64_SHA256"
        ;;
    *)
        echo "Unsupported Zig host architecture: $(uname -m)" >&2
        exit 1
        ;;
esac

if [[ -x "$INSTALL_DIR/zig" ]]; then
    INSTALLED_VERSION="$("$INSTALL_DIR/zig" version)"
    if [[ "$INSTALLED_VERSION" != "$ZIG_VERSION" ]]; then
        echo "$INSTALL_DIR contains Zig $INSTALLED_VERSION; expected $ZIG_VERSION" >&2
        exit 1
    fi
else
    if [[ -e "$INSTALL_DIR" ]]; then
        echo "$INSTALL_DIR already exists but does not contain the pinned Zig toolchain" >&2
        exit 1
    fi

    INSTALL_PARENT="$(dirname "$INSTALL_DIR")"
    mkdir -p "$INSTALL_PARENT"
    WORK_DIR="$(mktemp -d "$INSTALL_PARENT/.zig-${ZIG_VERSION}.XXXXXX")"
    trap 'rm -rf "$WORK_DIR"' EXIT

    ARCHIVE="$WORK_DIR/zig.tar.xz"
    URL="https://ziglang.org/download/${ZIG_VERSION}/zig-${ZIG_ARCH}-linux-${ZIG_VERSION}.tar.xz"
    curl --fail --location --retry 5 --retry-all-errors --output "$ARCHIVE" "$URL"
    if command -v sha256sum &>/dev/null; then
        ACTUAL_SHA256="$(sha256sum "$ARCHIVE" | awk '{print $1}')"
    elif command -v shasum &>/dev/null; then
        ACTUAL_SHA256="$(shasum -a 256 "$ARCHIVE" | awk '{print $1}')"
    else
        echo "Neither sha256sum nor shasum is available to verify the Zig archive." >&2
        exit 1
    fi
    if [[ "$ACTUAL_SHA256" != "$EXPECTED_SHA256" ]]; then
        echo "Zig archive SHA-256 mismatch." >&2
        echo "  Expected: $EXPECTED_SHA256" >&2
        echo "  Got:      $ACTUAL_SHA256" >&2
        exit 1
    fi

    EXTRACT_DIR="$WORK_DIR/extract"
    mkdir -p "$EXTRACT_DIR"
    tar -xJf "$ARCHIVE" --strip-components=1 -C "$EXTRACT_DIR"
    mv "$EXTRACT_DIR" "$INSTALL_DIR"
    trap - EXIT
    rm -rf "$WORK_DIR"
fi

mkdir -p "$INSTALL_DIR/bin"
cat > "$INSTALL_DIR/bin/aarch64-linux-musl-zig-cc" <<'EOF'
#!/usr/bin/env sh
# Go 1.25 probes Linux/ARM64 external linkers for GNU gold to avoid an old
# ld.bfd relocation bug. Zig uses LLD, which is also unaffected, so identify
# only that version probe as gold-compatible.
probe_gold=false
probe_version=false
for arg in "$@"; do
  [ "$arg" = "-fuse-ld=gold" ] && probe_gold=true
  [ "$arg" = "-Wl,--version" ] && probe_version=true
done
if [ "$probe_gold" = true ] && [ "$probe_version" = true ]; then
  echo "GNU gold (Zig LLD compatibility wrapper)"
  exit 0
fi
exec "$(dirname "$0")/../zig" cc -target aarch64-linux-musl "$@"
EOF
cat > "$INSTALL_DIR/bin/aarch64-linux-musl-zig-c++" <<'EOF'
#!/usr/bin/env sh
exec "$(dirname "$0")/../zig" c++ -target aarch64-linux-musl "$@"
EOF
chmod +x "$INSTALL_DIR/bin/aarch64-linux-musl-zig-cc" \
    "$INSTALL_DIR/bin/aarch64-linux-musl-zig-c++"

echo "$INSTALL_DIR/bin"
