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

# Install CMake for Barretenberg build
RUN apt-get update && apt-get install -y --no-install-recommends \
    cmake \
    && rm -rf /var/lib/apt/lists/*

# Set the workdir
WORKDIR /go/src/github.com/burnt-labs/xion

# Copy local files
COPY . .

# Barretenberg git reference (pinned to aztec-packages stable release)
ARG BB_REF=v3.0.3

# Build Barretenberg library for the target platform
# This must happen before xiond build so CGo can link against libbarretenberg.a
RUN --mount=type=cache,target=/root/.cache/cmake \
    set -eux; \
    BB_PLATFORM="linux_${TARGETARCH}"; \
    cd x/zk/barretenberg/wrapper && \
    mkdir -p build/${BB_PLATFORM} && \
    cd build/${BB_PLATFORM} && \
    cmake ../.. \
        -DCMAKE_BUILD_TYPE=Release \
        -DTARGET_PLATFORM=${BB_PLATFORM} \
        -DBB_REF=${BB_REF} && \
    cmake --build . --target barretenberg_wrapper --parallel $(nproc) && \
    echo "Checking for library files..." && \
    find /go/src/github.com/burnt-labs/xion/x/zk/barretenberg -name "*.a" -ls && \
    test -f /go/src/github.com/burnt-labs/xion/x/zk/barretenberg/lib/${BB_PLATFORM}/libbarretenberg.a && \
    echo "Barretenberg library built successfully"

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

USER root:root

COPY --from=builder /go/bin/xiond /usr/bin/xiond

# Add tools and cosmovisor
RUN set -euxo pipefail; \
    apk add --no-cache bash openssl curl htop jq lz4 tini; \
    curl -sSL https://github.com/cosmos/cosmos-sdk/releases/download/cosmovisor%2Fv1.5.0/cosmovisor-v1.5.0-linux-amd64.tar.gz \
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
