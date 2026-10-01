# syntax=docker/dockerfile:1

ARG GORELEASER_IMAGE="ghcr.io/goreleaser/goreleaser-cross"
ARG GORELEASER_VERSION="v1.25.3"
# Docker Official Image, pinned by digest (multi-arch index). The
# ghcr.io/linuxcontainers mirror stops at 3.20, which is past end of support.
ARG ALPINE_IMAGE="docker.io/library/alpine"
ARG ALPINE_VERSION="3.24.2"
ARG ALPINE_DIGEST="sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6"

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
        # Replace the barretenberg-go LFS pointer with the verified musl archive.
        BB_VERSION=$(grep 'github.com/burnt-labs/barretenberg-go' go.mod | cut -d ' ' -f 2); \
        if [ -n "${BB_VERSION}" ]; then \
            BB_MOD_DIR=$(go mod download -json "github.com/burnt-labs/barretenberg-go@${BB_VERSION}" | grep '"Dir"' | cut -d'"' -f4); \
            BB_LIB="${BB_MOD_DIR}/lib/linux_${GOARCH}/libbarretenberg.a"; \
            ./scripts/download-barretenberg.sh linux "${GOARCH}" "${BB_LIB}" musl; \
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
FROM ${ALPINE_IMAGE}:${ALPINE_VERSION}@${ALPINE_DIGEST} AS heighliner

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

# cosmovisor release and the SHA256 of each linux tarball, from the release's
# SHA256SUMS-cosmovisor-<version>.txt.
ARG COSMOVISOR_VERSION="v1.7.3"
ARG COSMOVISOR_SHA256_AMD64="3df6ef38cf976b00d226f391dc6866b8dc4040fc2f1b4a780d248f6e1cc9332e"
ARG COSMOVISOR_SHA256_ARM64="ff27992e1356fbcb858a604455ad28a9727415c3e35b947a4fdb30d8f91295cd"

# Add tools and cosmovisor
RUN set -euxo pipefail; \
    apk add --no-cache bash openssl curl htop jq lz4 tini; \
    case "${TARGETARCH}" in \
        amd64) COSMOVISOR_SHA256="${COSMOVISOR_SHA256_AMD64}" ;; \
        arm64) COSMOVISOR_SHA256="${COSMOVISOR_SHA256_ARM64}" ;; \
        *) echo "no cosmovisor checksum for ${TARGETARCH}" >&2; exit 1 ;; \
    esac; \
    COSMOVISOR_TGZ="cosmovisor-${COSMOVISOR_VERSION}-linux-${TARGETARCH}.tar.gz"; \
    curl -sSfL -o "/tmp/${COSMOVISOR_TGZ}" \
        "https://github.com/cosmos/cosmos-sdk/releases/download/cosmovisor%2F${COSMOVISOR_VERSION}/${COSMOVISOR_TGZ}"; \
    echo "${COSMOVISOR_SHA256}  /tmp/${COSMOVISOR_TGZ}" | sha256sum -c -; \
    tar -xzf "/tmp/${COSMOVISOR_TGZ}" -C /usr/bin cosmovisor; \
    rm "/tmp/${COSMOVISOR_TGZ}"; \
    cosmovisor version --help >/dev/null;

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
