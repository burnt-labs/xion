#!/usr/bin/env bash

set -euo pipefail

# barretenberg-go links against LLVM's libc++ (-lc++). The goreleaser-cross
# image ships clang but not the libc++ development libraries, so install them
# before any Linux/amd64 link runs. Linux/arm64 uses the Zig musl toolchain and
# Darwin uses the macOS SDK, both of which bundle their own libc++.

if [[ "$(uname -s)" != "Linux" ]]; then
    echo "install-libcxx: non-Linux host, nothing to do"
    exit 0
fi

if ls /usr/lib/llvm-*/lib/libc++.a /usr/lib/*/libc++.a &>/dev/null; then
    echo "install-libcxx: libc++ already installed"
    exit 0
fi

if ! command -v apt-get &>/dev/null; then
    echo "install-libcxx: libc++ is missing and apt-get is unavailable; install libc++ and libc++abi manually" >&2
    exit 1
fi

SUDO=""
if [[ "$(id -u)" -ne 0 ]]; then
    if command -v sudo &>/dev/null; then
        SUDO="sudo"
    else
        echo "install-libcxx: libc++ is missing and this user cannot run apt-get" >&2
        exit 1
    fi
fi

export DEBIAN_FRONTEND=noninteractive
$SUDO apt-get update
$SUDO apt-get install -y --no-install-recommends libc++-dev libc++abi-dev
$SUDO rm -rf /var/lib/apt/lists/*
echo "install-libcxx: installed libc++-dev and libc++abi-dev"
