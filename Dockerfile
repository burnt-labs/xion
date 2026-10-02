# syntax=docker/dockerfile:1

ARG GORELEASER_IMAGE="ghcr.io/goreleaser/goreleaser-cross"
ARG GORELEASER_VERSION="v1.25.3"
# Docker Official Image, pinned by digest (multi-arch index). The
# ghcr.io/linuxcontainers mirror stops at 3.20, which is past end of support.
ARG ALPINE_IMAGE="docker.io/library/alpine"
ARG ALPINE_VERSION="3.24.2"
ARG ALPINE_DIGEST="sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6"

# cosmovisor is built from pinned upstream source instead of the release
# tarball, whose bundled dependencies carry fixable HIGH/CRITICAL CVEs. See
# scripts/cosmovisor.md for provenance, the patch and the update procedure.
# COSMOVISOR_COMMIT: cosmos/cosmos-sdk commit; COSMOVISOR_SOURCE_SHA256: SHA256
# of its codeload tarball; COSMOVISOR_PATCH_SHA256: SHA256 of
# scripts/cosmovisor-patches/0001-decode-db-backend-output.patch;
# COSMOVISOR_GOLEVELDB_PATCH_SHA256: SHA256 of
# scripts/cosmovisor-patches/0002-pin-goleveldb.patch.
ARG COSMOVISOR_COMMIT="642a9c00b69ae3c9cb866a251eb32116de416c55"
ARG COSMOVISOR_SOURCE_SHA256="0f830f7bb3834d201ea8542a47b18cea1c113911eabdaa1d1f5e04b6f5ebf0dc"
ARG COSMOVISOR_PATCH_SHA256="896134c6af70b6baf167052c57e418e97a075bb4ae31ebbb7c8563ba8c7b6242"
ARG COSMOVISOR_GOLEVELDB_PATCH_SHA256="2a43d911a70328fe9c8fde7c2c9d54bc0f39449c5c8783d321645b68b9438359"
ARG COSMOVISOR_BUILD_IMAGE="docker.io/library/golang:1.26.8-bookworm@sha256:a688600ca24f8a4d3ca77f95b0dd40704a9fc787c826660eb7ba0b641b8b175d"

# --------------------------------------------------------
# cosmovisor builder (release image only)
# --------------------------------------------------------
FROM --platform=$BUILDPLATFORM ${COSMOVISOR_BUILD_IMAGE} AS cosmovisor-builder

# Always set by buildkit
ARG TARGETOS
ARG TARGETARCH

ARG COSMOVISOR_COMMIT
ARG COSMOVISOR_SOURCE_SHA256
ARG COSMOVISOR_PATCH_SHA256
ARG COSMOVISOR_GOLEVELDB_PATCH_SHA256

COPY scripts/cosmovisor-patches/0001-decode-db-backend-output.patch \
     scripts/cosmovisor-patches/0002-pin-goleveldb.patch /patches/

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    set -eux; \
    case "${TARGETOS}/${TARGETARCH}" in \
        linux/amd64|linux/arm64) ;; \
        *) echo "unsupported cosmovisor platform ${TARGETOS}/${TARGETARCH}" >&2; exit 1 ;; \
    esac; \
    echo "${COSMOVISOR_PATCH_SHA256}  /patches/0001-decode-db-backend-output.patch" | sha256sum -c -; \
    echo "${COSMOVISOR_GOLEVELDB_PATCH_SHA256}  /patches/0002-pin-goleveldb.patch" | sha256sum -c -; \
    mkdir -p /src /out; \
    curl -fsSL -o /tmp/cosmos-sdk.tar.gz \
        "https://codeload.github.com/cosmos/cosmos-sdk/tar.gz/${COSMOVISOR_COMMIT}"; \
    echo "${COSMOVISOR_SOURCE_SHA256}  /tmp/cosmos-sdk.tar.gz" | sha256sum -c -; \
    tar -xzf /tmp/cosmos-sdk.tar.gz -C /src --strip-components=1; \
    rm /tmp/cosmos-sdk.tar.gz; \
    cd /src; \
    for p in /patches/0001-decode-db-backend-output.patch /patches/0002-pin-goleveldb.patch; do \
        git apply --check -p1 "$p"; \
        git apply -p1 "$p"; \
    done; \
    cd /src/tools/cosmovisor; \
    export GOWORK=off GOTOOLCHAIN=local GOFLAGS=-mod=readonly; \
    go mod verify; \
    CGO_ENABLED=0 GOOS="${TARGETOS}" GOARCH="${TARGETARCH}" \
        go build -trimpath -buildvcs=false -o /out/cosmovisor ./cmd/cosmovisor;

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
COPY --from=cosmovisor-builder /out/cosmovisor /usr/bin/cosmovisor

ARG COSMOVISOR_COMMIT
ARG COSMOVISOR_SOURCE_SHA256
ARG COSMOVISOR_PATCH_SHA256
ARG COSMOVISOR_GOLEVELDB_PATCH_SHA256
LABEL io.burnt.cosmovisor.revision="${COSMOVISOR_COMMIT}" \
      io.burnt.cosmovisor.source-sha256="${COSMOVISOR_SOURCE_SHA256}" \
      io.burnt.cosmovisor.patch-sha256="${COSMOVISOR_PATCH_SHA256}" \
      io.burnt.cosmovisor.goleveldb-patch-sha256="${COSMOVISOR_GOLEVELDB_PATCH_SHA256}"

# Add tools
RUN set -euxo pipefail; \
    apk add --no-cache bash openssl curl htop jq lz4 tini; \
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
