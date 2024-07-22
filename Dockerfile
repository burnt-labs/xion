# syntax=docker/dockerfile:1

ARG GO_VERSION="1.22"
ARG ALPINE_VERSION="3.18"
ARG BUILDPLATFORM=linux/amd64
ARG BASE_IMAGE="golang:${GO_VERSION}-alpine${ALPINE_VERSION}"

# Builder
# -----------------------------------------------------------------------------

FROM --platform=${BUILDPLATFORM} ${BASE_IMAGE} AS builder

ARG GOOS=linux \
    GOARCH=amd64

ENV GOOS=$GOOS \
    GOARCH=$GOARCH

RUN apk add --no-cache \
    ca-certificates \
    build-base \
    linux-headers \
    git

# Download go dependencies
WORKDIR /xion
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/root/go/pkg/mod \
    go mod download -x

# Cosmwasm - Download correct libwasmvm version
RUN set -eux && \
    WASMVM_REPO="github.com/CosmWasm/wasmvm" && \
    WASMVM_VERSION=$(go list -m $WASMVM_REPO | cut -d ' ' -f 2) && \
    WASMVM_RELEASE=$WASMVM_REPO/releases/download/$WASMVM_VERSION && \
    LIBWASMVM_SOURCE="libwasmvm_muslc.$(uname -m).a" && \
    wget $WASMVM_RELEASE/$LIBWASMVM_SOURCE -O /lib/libwasmvm_muslc.a && \
    # verify checksum
    wget https://github.com/CosmWasm/wasmvm/releases/download/$WASMVM_VERSION/checksums.txt -O /tmp/checksums.txt && \
    sha256sum /lib/libwasmvm_muslc.a | grep $(cat /tmp/checksums.txt | grep libwasmvm_muslc.$(uname -m) | cut -d ' ' -f 1)

# Build xiond binary
COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/root/go/pkg/mod \
    make test-version \
    && LEDGER_ENABLED=false BUILD_TAGS=muslc LINK_STATICALLY=true make build

# -----------------------------------------------------------------------------
# Base
# -----------------------------------------------------------------------------

FROM alpine:${ALPINE_VERSION} AS xion-base
COPY --from=builder /xion/build/xiond /usr/bin/xiond

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

RUN set -eux \
  && apk add --no-cache \
    bash \
    openssl \
    curl \
    htop \
    jq \
    lz4 \
    tini

RUN set -eux \
  && addgroup -S xiond \
  && adduser xiond \
    --disabled-password \
    --gecos xiond \
    --ingroup xiond \
  && chown -R xiond:xiond /home/xiond

# -----------------------------------------------------------------------------
# Development
# -----------------------------------------------------------------------------
FROM xion-base AS dev

COPY ./docker/entrypoint.sh /home/xiond/entrypoint.sh
WORKDIR /home/xiond/.xiond

ENTRYPOINT ["/home/xiond/entrypoint.sh"]
CMD ["xiond", "start", "--trace"]

# -----------------------------------------------------------------------------
# Release
# -----------------------------------------------------------------------------
FROM xion-base AS release

USER xiond:xiond
WORKDIR /home/xiond/.xiond

CMD ["/usr/bin/xiond", "version"]
