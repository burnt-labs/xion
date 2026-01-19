#!/usr/bin/env bash
#
# build.sh - Build script for the Barretenberg wrapper library
#
# This script builds the static library for the specified platform(s).
# It supports native builds and cross-compilation via Docker.
#
# Usage:
#   ./build.sh [OPTIONS]
#
# Options:
#   --platform PLATFORM  Target platform (linux_amd64, linux_arm64, darwin_arm64)
#   --all                Build for all supported platforms
#   --native             Build for the current platform only (default)
#   --docker             Use Docker for cross-compilation
#   --clean              Clean build artifacts before building
#   --bb-ref REF         Barretenberg git ref - branch/tag/commit (default: master)
#   --help               Show this help message
#

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Default configuration
# Note: Barretenberg doesn't use semantic version tags - use branch or commit hash
BB_REF="${BB_REF:-master}"
USE_DOCKER=false
CLEAN_BUILD=false
BUILD_ALL=false
TARGET_PLATFORMS=()

# Docker images for cross-compilation
DOCKER_IMAGE_LINUX_AMD64="ghcr.io/AztecProtocol/barretenberg-x86_64-linux-gnu:latest"
DOCKER_IMAGE_LINUX_ARM64="ghcr.io/AztecProtocol/barretenberg-aarch64-linux-gnu:latest"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

show_help() {
    head -n 25 "$0" | tail -n 20
}

detect_platform() {
    local os arch

    case "$(uname -s)" in
        Linux)  os="linux" ;;
        Darwin) os="darwin" ;;
        *)      log_error "Unsupported OS: $(uname -s)"; exit 1 ;;
    esac

    case "$(uname -m)" in
        x86_64|amd64)   arch="amd64" ;;
        aarch64|arm64)  arch="arm64" ;;
        *)              log_error "Unsupported architecture: $(uname -m)"; exit 1 ;;
    esac

    echo "${os}_${arch}"
}

build_native() {
    local platform="$1"
    local build_dir="${SCRIPT_DIR}/build/${platform}"

    log_info "Building for ${platform} (native)..."

    mkdir -p "${build_dir}"
    cd "${build_dir}"

    cmake "${SCRIPT_DIR}" \
        -DCMAKE_BUILD_TYPE=Release \
        -DTARGET_PLATFORM="${platform}" \
        -DBB_REF="${BB_REF}"

    cmake --build . --parallel "$(nproc 2>/dev/null || sysctl -n hw.ncpu)"

    log_info "Build complete: ${PROJECT_ROOT}/lib/${platform}/libbarretenberg.a"
}

build_docker() {
    local platform="$1"
    local docker_image

    case "${platform}" in
        linux_amd64) docker_image="${DOCKER_IMAGE_LINUX_AMD64}" ;;
        linux_arm64) docker_image="${DOCKER_IMAGE_LINUX_ARM64}" ;;
        *)
            log_error "Docker build not supported for ${platform}"
            exit 1
            ;;
    esac

    log_info "Building for ${platform} (Docker)..."

    docker run --rm \
        -v "${PROJECT_ROOT}:/workspace" \
        -w /workspace/wrapper \
        "${docker_image}" \
        bash -c "
            set -e
            mkdir -p build/${platform}
            cd build/${platform}
            cmake ../.. \
                -DCMAKE_BUILD_TYPE=Release \
                -DTARGET_PLATFORM=${platform} \
                -DBB_VERSION=${BB_VERSION}
            cmake --build . --parallel \$(nproc)
        "

    log_info "Build complete: ${PROJECT_ROOT}/lib/${platform}/libbarretenberg.a"
}

clean_build() {
    log_info "Cleaning build artifacts..."
    rm -rf "${SCRIPT_DIR}/build"
    rm -rf "${PROJECT_ROOT}/lib/"*/*.a
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case "$1" in
        --platform)
            TARGET_PLATFORMS+=("$2")
            shift 2
            ;;
        --all)
            BUILD_ALL=true
            shift
            ;;
        --native)
            shift
            ;;
        --docker)
            USE_DOCKER=true
            shift
            ;;
        --clean)
            CLEAN_BUILD=true
            shift
            ;;
        --bb-ref)
            BB_REF="$2"
            shift 2
            ;;
        --help)
            show_help
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# Clean if requested
if [[ "${CLEAN_BUILD}" == true ]]; then
    clean_build
fi

# Determine target platforms
if [[ "${BUILD_ALL}" == true ]]; then
    TARGET_PLATFORMS=("linux_amd64" "linux_arm64" "darwin_arm64")
elif [[ ${#TARGET_PLATFORMS[@]} -eq 0 ]]; then
    TARGET_PLATFORMS=("$(detect_platform)")
fi

log_info "Barretenberg ref: ${BB_REF}"
log_info "Target platforms: ${TARGET_PLATFORMS[*]}"

# Build each platform
for platform in "${TARGET_PLATFORMS[@]}"; do
    case "${platform}" in
        linux_amd64|linux_arm64)
            if [[ "${USE_DOCKER}" == true ]]; then
                build_docker "${platform}"
            else
                # Check if we need cross-compilation
                native_platform="$(detect_platform)"
                if [[ "${native_platform}" != "${platform}" ]]; then
                    log_warn "Cross-compilation required for ${platform}, using Docker"
                    build_docker "${platform}"
                else
                    build_native "${platform}"
                fi
            fi
            ;;
        darwin_arm64)
            # Darwin can only be built natively on macOS
            if [[ "$(uname -s)" == "Darwin" ]]; then
                build_native "${platform}"
            else
                log_error "Darwin builds require macOS host"
                exit 1
            fi
            ;;
        *)
            log_error "Unknown platform: ${platform}"
            exit 1
            ;;
    esac
done

log_info "All builds completed successfully!"
