# syntax=docker/dockerfile:1

ARG GO_VERSION="1.22"
ARG ALPINE_VERSION="3.18"
ARG BASE_IMAGE="golang:${GO_VERSION}-alpine${ALPINE_VERSION}"

# --------------------------------------------------------
# Builder
# --------------------------------------------------------

FROM --platform=${BUILDPLATFORM} ${BASE_IMAGE} AS builder

ARG BUILDPLATFORM

ARG TARGETARCH
ENV TARGETARCH=${TARGETARCH}

# Install dependencies
RUN apk add --no-cache \
    ca-certificates \
    build-base \
    linux-headers \
    binutils-gold \
    git

# Set the workdir
WORKDIR /go/src/github.com/burnt-labs/xion

# Download go dependencies
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/root/pkg/mod \
    go mod download -x

# setup environment
ENV PROFILE=/root/.profile
RUN set -eux; \
    case "${BUILDPLATFORM}" in \
        linux/amd64) \
            echo "export GOOS=linux GOARCH=${TARGETARCH:-amd64} ARCH=x86_64" >> ${PROFILE}; \
            ;; \
        linux/arm64) \
            echo "export GOOS=linux GOARCH=${TARGETARCH:-arm64} ARCH=aarch64" >> ${PROFILE}; \
            ;; \
        *) \
            echo "Could not identify architecture"; \
            exit 1; \
            ;; \
    esac; \
    source ${PROFILE}; \
    if [ "${GOARCH}" != "$(uname -m)" ]; then \
        wget -c "https://musl.cc/${ARCH}-linux-musl-cross.tgz" -O - | tar -xzv --strip-components 1 -C /usr; \
    fi;

# Cosmwasm - Download correct libwasmvm version
RUN set -eux; \
    source ${PROFILE}; \
    LIBWASM="libwasmvm_muslc.${ARCH}.a"; \
    WASMVM_REPO="github.com/CosmWasm/wasmvm"; \
    WASMVM_VERSION="$(go list -m github.com/CosmWasm/wasmvm | cut -d ' ' -f 2)"; \
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
    source ${PROFILE}; \
    if [ "${GOARCH}" != "$(uname -m)" ]; then \
        LIBDIR=/usr/${ARCH}-linux-musl/lib; \
        mkdir -p /usr/${ARCH}-linux-musl/lib; \
        cp -a /lib/libwasmvm* /usr/${ARCH}-linux-musl/lib/; \
        export CC="${ARCH}-linux-musl-gcc" CXX="${ARCH}-linux-musl-g++"; \
    fi; \
    export CGO_ENABLED=1 LINK_STATICALLY=true BUILD_TAGS=muslc; \
    make test-version; \
    make install;

# --------------------------------------------------------
# Runner
# --------------------------------------------------------

FROM --platform=${BUILDPLATFORM} alpine:${ALPINE_VERSION} AS release
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

# # --------------------------------------------------------
# FROM release AS dev

# COPY ./docker/entrypoint.sh /home/xiond/entrypoint.sh

# USER root:root
# WORKDIR /home/xiond/

# ENTRYPOINT ["/home/xiond/entrypoint.sh"]
