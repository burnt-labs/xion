# syntax=docker/dockerfile:1

ARG GO_VERSION="1.21"
ARG ALPINE_VERSION="3.18"

# --------------------------------------------------------
# Builder
# --------------------------------------------------------

FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS builder

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
    go install cosmossdk.io/tools/cosmovisor/cmd/cosmovisor@v1.5.0; \
    go mod download

# Cosmwasm - Download correct libwasmvm version
RUN set -eux; \
    WASMVM_REPO="github.com/CosmWasm/wasmvm"; \
    WASMVM_MOD_VERSION="$(grep ${WASMVM_REPO} go.mod | cut -d ' ' -f 1)"; \
    WASMVM_VERSION="$(go list -m ${WASMVM_MOD_VERSION} | cut -d ' ' -f 2)"; \
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
    export CGO_ENABLED=1 LINK_STATICALLY=true BUILD_TAGS=muslc; \
    make test-version; \
    make install;

# --------------------------------------------------------
# Integration test builder
# --------------------------------------------------------
FROM builder AS test-builder

WORKDIR  /go/src/github.com/burnt-labs/xion/integration_tests

RUN set -eux; \
    go test -c; \
    cp -a integration_tests.test /go/bin

# --------------------------------------------------------
# Heighliner
# --------------------------------------------------------

# Build final image from scratch
FROM scratch AS heighliner

WORKDIR /bin

# Install trusted CA certificates
COPY --from=builder /etc/ssl/cert.pem /etc/ssl/cert.pem

# Install busybox
COPY --from=builder /bin/busybox /bin/busybox

SHELL [ "/bin/busybox" ]

RUN ln busybox sh

# Add hard links for read-only utils
# Will then only have one copy of the busybox minimal binary file with all utils pointing to the same underlying inode
RUN for bin in \
  cat \
  date \
  df \
  du \
  env \
  grep \
  head \
  less \
  ls \
  md5sum \
  pwd \
  sha1sum \
  sha256sum \
  sha3sum \
  sha512sum \
  sleep \
  stty \
  tail \
  tar \
  tee \
  tr \
  watch \
  which \
  ; do /bin/busybox ln /bin/busybox $bin; done

RUN /bin/busybox addgroup --gid 1025 -S heighliner; \
    /bin/busybox adduser --uid 1025 -S heighliner -G heighliner;

WORKDIR /home/heighliner
USER heighliner

# --------------------------------------------------------
# Runner
# --------------------------------------------------------

FROM alpine:${ALPINE_VERSION} AS release
COPY --from=builder /go/bin/xiond /usr/bin/xiond
COPY --from=builder /go/bin/cosmovisor /usr/bin/cosmovisor

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
