FROM bufbuild/buf:1.36.0 as builder

FROM swift:6.0.3-noble as swift-protoc-builder

RUN apt-get update  \
    && apt-get install -y software-properties-common
RUN add-apt-repository ppa:longsleep/golang-backports
RUN apt-get update  \
    && apt-get install -y  \
    libstdc++-12-dev  \
    nodejs \
    npm \
    git \
    make \
    clangd \
    g++ \
    jq \
    golang \
    protobuf-compiler

RUN npm install -g swagger-combine

COPY --link extramoduleimports.patch /app/extramoduleimports.patch
WORKDIR /app
RUN git clone --depth 1 --branch 1.28.2 https://github.com/apple/swift-protobuf --recursive
WORKDIR /app/swift-protobuf
RUN git apply /app/extramoduleimports.patch
RUN swift build -c release --static-swift-stdlib -Xlinker -s
COPY --from=builder /usr/local/bin /usr/local/bin
RUN mv /app/swift-protobuf/.build/release/protoc-gen-swift /usr/local/bin

#ARG UNAME=protobuild
#ARG UID=10000
#RUN adduser --uid $UID --shell /bin/sh $UNAME --disabled-password
#USER $UNAME

ENV GOLANG_PROTOBUF_VERSION=1.28.1 \
  GRPC_GATEWAY_VERSION=1.16.0 \
  GRPC_GATEWAY_PROTOC_GEN_OPENAPIV2_VERSION=2.20.0

ENV GOPATH=/usr/local/go
ENV PATH=$GOPATH/bin:/usr/local/go/bin:$PATH
RUN mkdir -p "$GOPATH/src" "$GOPATH/bin" && chmod -R 1777 "$GOPATH"

WORKDIR $GOPATH

RUN go install github.com/cosmos/cosmos-proto/cmd/protoc-gen-go-pulsar@latest && \
  go install google.golang.org/protobuf/cmd/protoc-gen-go@v${GOLANG_PROTOBUF_VERSION} && \
  go install github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway@v${GRPC_GATEWAY_VERSION} && \
  go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@v${GRPC_GATEWAY_PROTOC_GEN_OPENAPIV2_VERSION} && \
  go install cosmossdk.io/orm/cmd/protoc-gen-go-cosmos-orm@v1.0.0-beta.3 && \
  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# install all gogo protobuf binaries
RUN git clone https://github.com/cosmos/gogoproto.git; \
  cd gogoproto; \
  go mod download; \
  make install