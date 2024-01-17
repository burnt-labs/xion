# docker build . -t cosmwasm/xiond:latest
# docker run --rm -it cosmwasm/xiond:latest /bin/sh
FROM golang:1.19-alpine3.17 AS go-builder
  ARG arch=x86_64

  ENV WASMVM_VERSION=v1.4.1
  ENV WASMVM_CHECKSUM_AARCH64=a8259ba852f1b68f2a5f8eb666a9c7f1680196562022f71bb361be1472a83cfd
  ENV WASMVM_CHECKSUM_x86_64=324c1073cb988478d644861783ed5a7de21cfd090976ccc6b1de0559098fbbad

  # this comes from standard alpine nightly file
  #  https://github.com/rust-lang/docker-rust-nightly/blob/master/alpine3.12/Dockerfile
  # with some changes to support our toolchain, etc
  RUN set -eux; apk add --no-cache ca-certificates build-base;

  RUN apk add git
  # NOTE: add these to run with LEDGER_ENABLED=true
  # RUN apk add libusb-dev linux-headers

  WORKDIR /code
  COPY go.mod /code/
  COPY go.sum /code/
  RUN go mod download -x

  COPY ./.git /code/.git
  COPY ./app /code/app
  COPY ./cmd /code/cmd
  COPY ./contrib /code/contrib
  COPY ./proto /code/proto
  COPY ./x /code/x
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
FROM alpine:3.17 AS xion-dev
  COPY --from=go-builder /code/build/xiond /usr/bin/xiond

  # rest server
  EXPOSE 1317
  # tendermint grpc
  EXPOSE 9090
  # tendermint p2p
  EXPOSE 26656
  # tendermint rpc
  EXPOSE 26657
  # tendermint prometheus
  EXPOSE 26660

  RUN mkdir /xion

  RUN set -euxo pipefail \
    && apk add --no-cache \
    bash \
    curl \
    htop \
    jq \
    lz4 \
    tini

  RUN set -euxo pipefail \
    && addgroup -S xiond \
    && adduser \
       --disabled-password \
       --gecos xiond \
       --ingroup xiond \
       xiond

  RUN set -eux \
    && chown -R xiond:xiond /home/xiond \
    && chown -R xiond:xiond /xion

  USER xiond:xiond

  COPY ./docker/entrypoint.sh /home/xiond/entrypoint.sh

  CMD ["/home/xiond/entrypoint.sh"]

# --------------------------------------------------------
FROM alpine:3.17 AS xion-release

  COPY --from=go-builder /code/build/xiond /usr/bin/xiond

  # rest server
  EXPOSE 1317
  # tendermint grpc
  EXPOSE 9090
  # tendermint p2p
  EXPOSE 26656
  # tendermint rpc
  EXPOSE 26657
  # tendermint prometheus
  EXPOSE 26660

  RUN set -euxo pipefail \
    && apk add --no-cache \
      aria2 \
      aws-cli \
      bash \
      curl \
      htop \
      jq \
      lz4 \
      tini

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
