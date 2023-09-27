# docker build . -t cosmwasm/xiond:latest
# docker run --rm -it cosmwasm/xiond:latest /bin/sh
FROM golang:1.19-alpine3.17 AS go-builder
  ARG arch=x86_64

  # this comes from standard alpine nightly file
  #  https://github.com/rust-lang/docker-rust-nightly/blob/master/alpine3.12/Dockerfile
  # with some changes to support our toolchain, etc
  RUN set -eux; apk add --no-cache ca-certificates build-base;

  RUN apk add git
  # NOTE: add these to run with LEDGER_ENABLED=true
  # RUN apk add libusb-dev linux-headers

  WORKDIR /code
  COPY ./.git /code/.git
  COPY ./app /code/app
  COPY ./cmd /code/cmd
  COPY ./contrib /code/contrib
  COPY ./proto /code/proto
  COPY ./x /code/x
  COPY go.mod /code/
  COPY go.sum /code/
  COPY Makefile /code/

  # See https://github.com/CosmWasm/wasmvm/releases
  ADD https://github.com/CosmWasm/wasmvm/releases/download/v1.2.4/libwasmvm_muslc.aarch64.a /lib/libwasmvm_muslc.aarch64.a
  ADD https://github.com/CosmWasm/wasmvm/releases/download/v1.2.4/libwasmvm_muslc.x86_64.a /lib/libwasmvm_muslc.x86_64.a
  RUN sha256sum /lib/libwasmvm_muslc.aarch64.a | grep 682a54082e131eaff9beec80ba3e5908113916fcb8ddf7c668cb2d97cb94c13c
  RUN sha256sum /lib/libwasmvm_muslc.x86_64.a | grep ce3d892377d2523cf563e01120cb1436f9343f80be952c93f66aa94f5737b661

  # Copy the library you want to the final location that will be found by the linker flag `-lwasmvm_muslc`
  RUN cp -vf /lib/libwasmvm_muslc.${arch}.a /lib/libwasmvm_muslc.a

  # force it to use static lib (from above) not standard libgo_cosmwasm.so file
  RUN set -eux \
    && make test-version \
    && LEDGER_ENABLED=false BUILD_TAGS=muslc LINK_STATICALLY=true make build

  RUN echo "Ensuring binary is statically linked ..." \
    && (file /code/build/xiond | grep "statically linked")

# --------------------------------------------------------
FROM alpine:3.16 AS localdev

  COPY --from=go-builder /code/build/xiond /usr/bin/xiond

  COPY ./docker/local-config /xion/config
  COPY ./docker/entrypoint.sh /root/entrypoint.sh
  RUN chmod +x /root/entrypoint.sh

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

  VOLUME [ "/xion/data" ]

  CMD ["/root/entrypoint.sh"]

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
