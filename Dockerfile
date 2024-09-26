# syntax=docker/dockerfile:1

ARG GORELEASER_VERSION="1.22.7"
ARG ALPINE_VERSION="3.18"

# --------------------------------------------------------
# Builder
# --------------------------------------------------------

FROM ghcr.io/burnt-labs/goreleaser-cross:${GORELEASER_VERSION} AS builder


# Always set by buildkit
ARG TARGETPLATFORM
ARG TARGETARCH
ARG TARGETOS
ARG CALLER

# Consume Args to env
ENV GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH}

# Set the workdir
WORKDIR /root/go/bin

# Get cosmovisor
RUN set -eux; \
    go install cosmossdk.io/tools/cosmovisor/cmd/cosmovisor@v1.5.0;

# Set the workdir
WORKDIR /root/go/src/github.com/burnt-labs/xion

COPY . .

# run goreleaser
RUN set -eux; \
    if [ "${CALLER:-}" = "goreleaser" ]; then \
        cp $(find . -name xiond | grep ${TARGETARCH}) /root/go/bin/xiond; \
    else \
        # use the binary from goreleaser if it exists
        SHORT_SHA=$(git rev-parse --short HEAD); \
        if [ ! -f "dist/${SHORT_SHA}/xiond_${TARGETOS}_${TARGETARCH}/xiond" ]; then \
            /entrypoint.sh build --clean --single-target --skip validate --config=.goreleaser.yaml; \
        fi; \
        cp dist/${SHORT_SHA}/xiond_${TARGETOS}_${TARGETARCH}/xiond /root/go/bin/xiond; \
    fi;

# --------------------------------------------------------
# Heighliner
# --------------------------------------------------------

# Build final image from scratch
FROM scratch AS heighliner

WORKDIR /bin
ENV PATH=/bin

ARG ALPINE_VERSION

# Install xiond
COPY --from=builder /root/go/bin/xiond /bin/xiond

# Install busybox
COPY --from=busybox:1.36-musl /bin/busybox /bin/busybox

# users and group
COPY --from=busybox:1.36-musl /etc/passwd /etc/group /etc/

# Install trusted CA certificates
COPY --from=alpine:3.20 /etc/ssl/cert.pem /etc/ssl/cert.pem

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

FROM alpine:${ALPINE_VERSION} AS release
COPY --from=builder /root/go/bin/xiond /usr/bin/xiond
COPY --from=builder /root/go/bin/cosmovisor /usr/bin/cosmovisor

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
