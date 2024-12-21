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
COPY . /code/
# See https://github.com/CosmWasm/wasmvm/releases
ADD https://github.com/CosmWasm/wasmvm/releases/download/v1.2.2/libwasmvm_muslc.aarch64.a /lib/libwasmvm_muslc.aarch64.a
ADD https://github.com/CosmWasm/wasmvm/releases/download/v1.2.2/libwasmvm_muslc.x86_64.a /lib/libwasmvm_muslc.x86_64.a
RUN sha256sum /lib/libwasmvm_muslc.aarch64.a | grep 20ab42fceddd5347b973c254717eed62b2d46925c098f58304d09488ed59464a
RUN sha256sum /lib/libwasmvm_muslc.x86_64.a | grep fc859817a1c548ceb0e02ad8cf5c8e374d240caac547d04dd02b284ace8ab33d

# Copy the library you want to the final location that will be found by the linker flag `-lwasmvm_muslc`
RUN cp /lib/libwasmvm_muslc.${arch}.a /lib/libwasmvm_muslc.a

# force it to use static lib (from above) not standard libgo_cosmwasm.so file
RUN LEDGER_ENABLED=false BUILD_TAGS=muslc LINK_STATICALLY=true make build
RUN echo "Ensuring binary is statically linked ..." \
  && (file /code/build/xiond | grep "statically linked")

# --------------------------------------------------------
FROM alpine:3.17

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

