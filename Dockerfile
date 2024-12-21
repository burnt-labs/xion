FROM golang:1.21-alpine3.18 AS go-builder
ARG arch=x86_64

ENV WASMVM_VERSION=v1.5.2
ENV WASMVM_CHECKSUM_AARCH64=e78b224c15964817a3b75a40e59882b4d0e06fd055b39514d61646689cef8c6e
ENV WASMVM_CHECKSUM_x86_64=e660a38efb2930b34ee6f6b0bb12730adccb040b6ab701b8f82f34453a426ae7

RUN set -euxo pipefail \
  && apk add --no-cache \
    ca-certificates  \
    build-base \
    git

# NOTE: add these to run with LEDGER_ENABLED=true
# RUN apk add libusb-dev linux-headers

WORKDIR /code
COPY go.mod /code/
COPY go.sum /code/
RUN go mod download

COPY ./.git /code/.git
COPY ./app /code/app
COPY ./cmd /code/cmd
COPY ./contrib /code/contrib
COPY ./proto /code/proto
COPY ./x /code/x
COPY ./wasmbindings /code/wasmbindings
COPY Makefile /code/

# See https://github.com/CosmWasm/wasmvm/releases
ADD https://github.com/CosmWasm/wasmvm/releases/download/${WASMVM_VERSION}/libwasmvm_muslc.aarch64.a /lib/libwasmvm_muslc.aarch64.a
ADD https://github.com/CosmWasm/wasmvm/releases/download/${WASMVM_VERSION}/libwasmvm_muslc.x86_64.a /lib/libwasmvm_muslc.x86_64.a
RUN sha256sum /lib/libwasmvm_muslc.aarch64.a | grep ${WASMVM_CHECKSUM_AARCH64}
RUN sha256sum /lib/libwasmvm_muslc.x86_64.a | grep ${WASMVM_CHECKSUM_x86_64}

# Copy the library you want to the final location that will be found by the linker flag `-lwasmvm_muslc`
RUN cp -vf /lib/libwasmvm_muslc.${arch}.a /lib/libwasmvm_muslc.a

# force it to use static lib (from above) not standard libgo_cosmwasm.so file
RUN set -eux \
  && make test-version \
  && LEDGER_ENABLED=false BUILD_TAGS=muslc LINK_STATICALLY=true make build

RUN echo "Ensuring binary is statically linked ..." \
  && (file /code/build/xiond | grep "statically linked")

# --------------------------------------------------------
FROM alpine:3.19.1 AS xion-base
COPY --from=go-builder /code/build/xiond /usr/bin/xiond

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
FROM xion-base AS xion-dev

COPY ./docker/entrypoint.sh /home/xiond/entrypoint.sh
WORKDIR /home/xiond/

CMD ["/home/xiond/entrypoint.sh"]

# --------------------------------------------------------
FROM xion-base as xion-release

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
