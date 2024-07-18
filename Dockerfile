# syntax=docker/dockerfile:1

ARG GO_VERSION="1.22"
ARG ALPINE_VERSION_BUILDER="3.18"
ARG ALPINE_VERSION_RUNNER="3.19"
ARG BUILDPLATFORM=linux/amd64
ARG BASE_IMAGE="golang:${GO_VERSION}-alpine${ALPINE_VERSION_BUILDER}"

# --------------------------------------------------------
# Builder
# --------------------------------------------------------

FROM --platform=${BUILDPLATFORM} ${BASE_IMAGE} AS builder

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
    go mod download

# Cosmwasm - Download correct libwasmvm version
RUN WASMVM_VERSION=$(go list -m github.com/CosmWasm/wasmvm | cut -d ' ' -f 2) && \
    wget https://github.com/CosmWasm/wasmvm/releases/download/$WASMVM_VERSION/libwasmvm_muslc.$(uname -m).a \
      -O /lib/libwasmvm_muslc.a && \
    # verify checksum
    wget https://github.com/CosmWasm/wasmvm/releases/download/$WASMVM_VERSION/checksums.txt -O /tmp/checksums.txt && \
    sha256sum /lib/libwasmvm_muslc.a | grep $(cat /tmp/checksums.txt | grep libwasmvm_muslc.$(uname -m) | cut -d ' ' -f 1)

# Build xiond binary
COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/root/go/pkg/mod \
    make test-version \
    && LEDGER_ENABLED=false BUILD_TAGS=muslc LINK_STATICALLY=true make build

# --------------------------------------------------------
# Runner
# --------------------------------------------------------

FROM alpine:${ALPINE_VERSION_RUNNER} AS xion-base
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

RUN set -euxo pipefail \
  && echo http://dl-cdn.alpinelinux.org/alpine/edge/main >> /etc/apk/repositories \
  && apk add --no-cache \
    bash \
    openssl \
    curl \
    htop \
    jq \
    lz4 \
    tini

# --------------------------------------------------------
FROM xion-base AS dev

COPY ./docker/entrypoint.sh /home/xiond/entrypoint.sh
WORKDIR /home/xiond/

CMD ["/home/xiond/entrypoint.sh"]

# --------------------------------------------------------
FROM xion-base AS release

RUN set -euxo pipefail \
  && addgroup -S xiond \
  && adduser \
    --disabled-password \
    --gecos xiond \
    --ingroup xiond \
    xiond

RUN set -eux \
  && chown -R xiond:xiond /home/xiond

USER xiond:xiond
WORKDIR /home/xiond/.xiond

CMD ["/usr/bin/xiond", "version"]
