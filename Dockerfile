# syntax=docker/dockerfile:1

ARG GO_VERSION="1.22"
ARG ALPINE_VERSION="3.18"

# --------------------------------------------------------
# Builder
# --------------------------------------------------------

FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS builder

# Always set by buildkit
ARG TARGETPLATFORM
ARG TARGETARCH
ARG TARGETOS

# Install dependencies
RUN apk add --no-cache \
    build-base \
    ca-certificates \
    linux-headers \
    binutils-gold \
    git

# Set the workdir
WORKDIR /go/src/github.com/burnt-labs/xion

# Download go dependencies
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/root/pkg/mod \
    go mod download

# Cosmwasm - Download correct libwasmvm version
RUN set -eux; \
    WASMVM_REPO="github.com/CosmWasm/wasmvm"; \
    WASMVM_VERSION="$(go list -m github.com/CosmWasm/wasmvm/v2 | cut -d ' ' -f 2)"; \
    [ ${TARGETPLATFORM} = "linux/amd64" ] && LIBWASM="libwasmvm_muslc.x86_64.a"; \
    [ ${TARGETPLATFORM} = "linux/arm64" ] && LIBWASM="libwasmvm_muslc.aarch64.a"; \
    [ ${TARGETOS} = "darwin" ] && LIBWASM="libwasmvmstatic_darwin.a"; \
    [ -z "$LIBWASM" ] && echo "Arch ${TARGETARCH} not recognized" && exit 1; \
    wget "https://${WASMVM_REPO}/releases/download/${WASMVM_VERSION}/${LIBWASM}" -O "/lib/${LIBWASM}"; \
    # verify checksum
    EXPECTED=$(wget -q "https://${WASMVM_REPO}/releases/download/${WASMVM_VERSION}/checksums.txt" -O- | grep "${LIBWASM}" | awk '{print $1}'); \
    sha256sum "/lib/${LIBWASM}" | grep "${EXPECTED}"; \
    cp /lib/${LIBWASM} /lib/libwasmvm_muslc.a;

# Copy local files
COPY . .

# Build xiond binary
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/root/pkg/mod \
    set -eux; \
    export GOOS=${TARGETOS} GOARCH=${TARGETARCH}; \
    export CGO_ENABLED=1 LINK_STATICALLY=true BUILD_TAGS=muslc; \
    make test-version; \
    make install;

# --------------------------------------------------------
# Runner
# --------------------------------------------------------

FROM alpine:${ALPINE_VERSION} AS release
COPY --from=builder /go/bin/xiond /usr/bin/xiond

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

RUN set -euxo pipefail; \
    apk add --no-cache bash openssl curl htop jq lz4 tini; \
    addgroup --gid 1000 -S xiond; \
    adduser --uid 1000 -S xiond \
        --disabled-password \
        --gecos xiond \
        --ingroup xiond; \
    mkdir -p /home/xiond; \
    chown -R xiond:xiond /home/xiond

USER xiond:xiond
WORKDIR /home/xiond/.xiond
CMD ["/usr/bin/xiond"]
