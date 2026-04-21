# syntax=docker/dockerfile:1

ARG GORELEASER_IMAGE="ghcr.io/goreleaser/goreleaser-cross"
ARG GORELEASER_VERSION="v1.25.3"
ARG ALPINE_VERSION="3.20"

# --------------------------------------------------------
# Builder
# --------------------------------------------------------
FROM ${GORELEASER_IMAGE}:${GORELEASER_VERSION} AS builder

# Always set by buildkit
ARG TARGETPLATFORM
ARG TARGETARCH
ARG TARGETOS

# needed in makefile
ARG COMMIT
ARG VERSION

# Consume Args to env
ENV COMMIT=${COMMIT} \
    VERSION=${VERSION} \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} 

# Install libc++ (barretenberg static lib is built against libc++)
RUN apt-get update && apt-get install -y --no-install-recommends \
    libc++-dev libc++abi-dev \
    && rm -rf /var/lib/apt/lists/*

# Set the workdir
WORKDIR /go/src/github.com/burnt-labs/xion

# Copy local files
COPY . .

# Build xiond binary
ARG PREBUILT_BINARY
ENV PREBUILT_BINARY=${PREBUILT_BINARY}
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/root/pkg/mod \
    set -eux; \
    mkdir -p /go/bin; \
    if [ -e "${PREBUILT_BINARY:-}" ]; then \
        cp -a "${PREBUILT_BINARY}" /go/bin/xiond; \
        chmod a+x /go/bin/xiond; \
    else \
        # Download wasmvm static library and place in module cache
        WASMVM_VERSION=$(grep 'github.com/CosmWasm/wasmvm' go.mod | cut -d ' ' -f 2); \
        WASM_ARCH=$([ "${GOARCH}" = "arm64" ] && echo "aarch64" || echo "x86_64"); \
        WASM_LIB="libwasmvm_muslc.${WASM_ARCH}.a"; \
        mkdir -p /tmp/wasmvm; \
        curl -sSfL "https://github.com/CosmWasm/wasmvm/releases/download/${WASMVM_VERSION}/${WASM_LIB}" \
            -o "/tmp/wasmvm/${WASM_LIB}"; \
        WASM_MODPATH=$(grep 'github.com/CosmWasm/wasmvm' go.mod | awk '{print $1}'); \
        WASM_MOD_DIR=$(go mod download -json "${WASM_MODPATH}@${WASMVM_VERSION}" | grep '"Dir"' | cut -d'"' -f4); \
        chmod -R u+w "${WASM_MOD_DIR}" 2>/dev/null || true; \
        cp "/tmp/wasmvm/${WASM_LIB}" "${WASM_MOD_DIR}/internal/api/${WASM_LIB}"; \
        # Fix barretenberg-go LFS pointers (go mod download gets pointer files, not real binaries)
        BB_VERSION=$(grep 'github.com/burnt-labs/barretenberg-go' go.mod | cut -d ' ' -f 2); \
        if [ -n "${BB_VERSION}" ]; then \
            BB_MOD_DIR=$(go mod download -json "github.com/burnt-labs/barretenberg-go@${BB_VERSION}" | grep '"Dir"' | cut -d'"' -f4); \
            BB_LIB="${BB_MOD_DIR}/lib/linux_${GOARCH}/libbarretenberg.a"; \
            if [ ! -f "${BB_LIB}" ] || head -1 "${BB_LIB}" 2>/dev/null | grep -q 'git-lfs'; then \
                echo "Downloading libbarretenberg ${BB_VERSION} for linux/${GOARCH}"; \
                chmod -R u+w "${BB_MOD_DIR}" 2>/dev/null || true; \
                mkdir -p "$(dirname "${BB_LIB}")"; \
                curl -sSfL "https://github.com/burnt-labs/barretenberg-go/releases/download/${BB_VERSION}/libbarretenberg_linux_${GOARCH}.a" \
                    -o "${BB_LIB}"; \
            fi; \
        fi; \
        goreleaser build \
            --config .goreleaser/build.yaml \
            --snapshot --clean --single-target --skip validate; \
        cp -a $(find ./dist -name xiond-${GOOS}-${GOARCH}) /go/bin/xiond; \
        chmod a+x /go/bin/xiond; \
    fi;

# --------------------------------------------------------
# Heighliner image
# --------------------------------------------------------
FROM ghcr.io/linuxcontainers/alpine:${ALPINE_VERSION} AS heighliner

COPY --from=builder /go/bin/xiond /usr/bin/xiond

# Add tools and cosmovisor
RUN set -euxo pipefail; \
    apk add --no-cache jq; 

# Add heighliner user and group
RUN set -euxo pipefail; \
    addgroup -g 1025 heighliner; \
    adduser -u 1025 -D -h /var/cosmos-chain -s /bin/bash -G heighliner heighliner; 

USER heighliner:heighliner

# --------------------------------------------------------
# Heighliner image
# --------------------------------------------------------
FROM heighliner AS release

# Always set by buildkit
ARG TARGETARCH

USER root:root

COPY --from=builder /go/bin/xiond /usr/bin/xiond

# Add tools and cosmovisor
RUN set -euxo pipefail; \
    apk add --no-cache bash openssl curl htop jq lz4 tini; \
    curl -sSL https://github.com/cosmos/cosmos-sdk/releases/download/cosmovisor%2Fv1.5.0/cosmovisor-v1.5.0-linux-${TARGETARCH}.tar.gz \
    | tar -xz -C /usr/bin;

# Add xiond users and groups
RUN set -euxo pipefail; \
    addgroup -g 1000 xiond; \
    adduser -u 1000 -D -s /bin/bash -G xiond xiond; \
    mkdir -m 0775 -p /home/xiond/.xiond; \
    chown xiond:xiond /home/xiond/.xiond;

# api
EXPOSE 1317
# grpc
EXPOSE 9090
# p2p
EXPOSE 26656
# rpc
EXPOSE 26657
# prometheus
EXPOSE 26660

USER xiond:xiond
WORKDIR /home/xiond/.xiond
CMD ["/usr/bin/xiond"]
