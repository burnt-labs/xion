# syntax=docker/dockerfile:1

ARG GORELEASER_IMAGE="ghcr.io/goreleaser/goreleaser-cross"
ARG GORELEASER_VERSION="v1.23.6"
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
        goreleaser build \
            --config .goreleaser/build.yaml \
            --snapshot --clean --single-target --skip validate; \
        cp -a $(find ./dist -name xiond-${GOOS}-${GOARCH}) /go/bin/xiond; \
        chmod a+x /go/bin/xiond; \
    fi;

# --------------------------------------------------------
# Heighliner
# --------------------------------------------------------

# Build final image from scratch
FROM scratch AS heighliner

WORKDIR /bin
ENV PATH=/bin

# Install busybox
COPY --from=busybox:1.36-musl /bin/busybox /bin/busybox

# users and group
COPY --from=busybox:1.36-musl /etc/passwd /etc/group /etc/

# Install trusted CA certificates
COPY --from=ghcr.io/linuxcontainers/alpine:3 /etc/ssl/cert.pem /etc/ssl/cert.pem

# Install xiond
COPY --from=builder /go/bin/xiond /bin/xiond

# Install jq
COPY --from=ghcr.io/strangelove-ventures/infra-toolkit:v0.1.4 /usr/local/bin/jq /bin/jq

# link shell
RUN ["busybox", "ln", "/bin/busybox", "sh"]

# Add hard links for read-only utils
# Will then only have one copy of the busybox minimal binary file with all utils pointing to the same underlying inode
RUN set -eux; \
    for bin in \
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
    ; do busybox ln /bin/busybox $bin; \
    done;

RUN set -eux; \
    busybox mkdir -p /tmp /home/heighliner; \
    busybox addgroup --gid 1025 -S heighliner; \
    busybox adduser --uid 1025 -h /home/heighliner -S heighliner -G heighliner; \
    busybox chown 1025:1025 /tmp /home/heighliner; \
    busybox unlink busybox;

WORKDIR /home/heighliner
USER heighliner

# --------------------------------------------------------
# Runner
# --------------------------------------------------------

FROM ghcr.io/linuxcontainers/alpine:${ALPINE_VERSION} AS release
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
    curl -sSL https://github.com/cosmos/cosmos-sdk/releases/download/cosmovisor%2Fv1.5.0/cosmovisor-v1.5.0-linux-amd64.tar.gz \
    | tar -xz -C /usr/bin; \
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
