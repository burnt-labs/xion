# Build targets and configuration

# Project metadata
VERSION ?= $(shell echo $(shell git describe --tags) | sed 's/^v//')
COMMIT ?= $(shell git log -1 --format='%H')

# External tools
DOCKER := $(shell which docker)

# Environment detection
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
COMMA := ,

# Build-specific configuration
LEDGER_ENABLED ?= true
SDK_PACK := $(shell go list -m github.com/cosmos/cosmos-sdk | sed  's/ /\@/g')

# Docker and goreleaser configuration
GORELEASER_CROSS_IMAGE := $(if $(GORELEASER_KEY),ghcr.io/goreleaser/goreleaser-cross-pro,ghcr.io/goreleaser/goreleaser-cross)
GORELEASER_CROSS_VERSION ?= v1.24.5
GORELEASER_IMAGE ?= $(GORELEASER_CROSS_IMAGE)
GORELEASER_VERSION ?= $(GORELEASER_CROSS_VERSION)
GORELEASER_RELEASE ?= false
GORELEASER_SKIP_FLAGS ?= ""
XION_IMAGE ?= xiond:$(GOARCH)
HEIGHLINER_IMAGE ?= heighliner:$(GOARCH)

# Build tags processing
build_tags = netgo
ifeq ($(LEDGER_ENABLED),true)
  ifeq ($(OS),Windows_NT)
    GCCEXE = $(shell where gcc.exe 2> NUL)
    ifeq ($(GCCEXE),)
      $(error gcc.exe not installed for ledger support, please install or set LEDGER_ENABLED=false)
    else
      build_tags += ledger
    endif
  else
    UNAME_S = $(shell uname -s)
    ifeq ($(UNAME_S),OpenBSD)
      $(warning OpenBSD detected, disabling ledger support (https://github.com/cosmos/cosmos-sdk/issues/1988))
    else
      GCC = $(shell command -v gcc 2> /dev/null)
      ifeq ($(GCC),)
        $(error gcc not installed for ledger support, please install or set LEDGER_ENABLED=false)
      else
        build_tags += ledger
      endif
    endif
  endif
endif

ifeq ($(WITH_CLEVELDB),yes)
  build_tags += gcc
endif
build_tags += $(BUILD_TAGS)
build_tags := $(strip $(build_tags))

build_tags_comma_sep := $(shell echo $(build_tags) | sed 's/ /,/g')
ldflags = -X github.com/cosmos/cosmos-sdk/version.Name=xion \
		  -X github.com/cosmos/cosmos-sdk/version.AppName=xiond \
		  -X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
		  -X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT) \
		  -X github.com/CosmWasm/wasmd/app.Bech32Prefix=xion \
		  -X "github.com/cosmos/cosmos-sdk/version.BuildTags=$(build_tags_comma_sep)"

ifeq ($(WITH_CLEVELDB),yes)
  ldflags += -X github.com/cosmos/cosmos-sdk/types.DBBackend=cleveldb
endif
ifeq ($(LINK_STATICALLY),true)
	ldflags += -linkmode=external -extldflags "-Wl,-z,muldefs -static"
endif
ldflags += $(LDFLAGS)
ldflags := $(strip $(ldflags))

BUILD_FLAGS := -tags "$(build_tags_comma_sep)" -ldflags '$(ldflags)' -trimpath

install: go.sum
	go install -mod=readonly $(BUILD_FLAGS) ./cmd/xiond

build: guard-VERSION guard-COMMIT
ifeq ($(OS),Windows_NT)
	$(error wasmd server not supported. Use "make build-windows-client" for client)
	exit 1
else
	go build -mod=readonly $(BUILD_FLAGS) -o build/xiond ./cmd/xiond
endif

build-all:
	$(DOCKER) run --rm \
		--env NODISTDIR=false \
		--platform linux/amd64 \
		--volume $(CURDIR):/root/go/src/github.com/burnt-network/xion \
		--workdir /root/go/src/github.com/burnt-network/xion \
		$(GORELEASER_CROSS_IMAGE):$(GORELEASER_CROSS_VERSION) \
		build --config .goreleaser/build.yaml --clean --skip validate

build-local:
	$(DOCKER) run --rm \
		--env GOOS=$(GOOS) \
		--env GOARCH=$(GOARCH) \
		--env NODISTDIR=true \
		--env GORELEASER_KEY=$(GORELEASER_KEY) \
		--volume $(CURDIR):/root/go/src/github.com/burnt-network/xion \
		--workdir /root/go/src/github.com/burnt-network/xion \
		$(GORELEASER_CROSS_IMAGE):$(GORELEASER_CROSS_VERSION) \
		build --config .goreleaser/build.yaml --clean --skip validate --single-target

build-linux-arm64 build-linux-amd64 build-darwin-amd64 build-darwin-arm64 build-windows-amd64:
	$(MAKE) build-local \
		GOOS=$(if $(findstring windows,$@),windows,$(if $(findstring darwin,$@),darwin,linux)) \
		GOARCH=$(if $(findstring arm64,$@),arm64,amd64)

build-docker:
	$(DOCKER) build \
	  --platform linux/$(GOARCH) \
	  --target=$(if $(TARGET),$(TARGET),release) \
	  --progress=plain \
	  --build-arg=GORELEASER_IMAGE=$(GORELEASER_IMAGE) \
	  --build-arg=GORELEASER_VERSION=$(GORELEASER_VERSION) \
	  --tag $(XION_IMAGE) .

build-docker-arm64 build-docker-amd64:
	$(MAKE) build-docker \
		GOARCH=$(if $(findstring arm64,$@),arm64,amd64) \
		XION_IMAGE="xiond:$(GOARCH)"

build-heighliner build-heighliner-amd64 build-heighliner-arm64:
	$(MAKE) build-docker \
		GOARCH=$(if $(findstring arm64,$@),arm64,$(if $(findstring amd64,$@),amd64,$(GOARCH))) \
		XION_IMAGE=heighliner:$(GOARCH) \
		TARGET=heighliner

release-snapshot:
	$(DOCKER) run --rm \
		--env "GORELEASER_KEY=$(GORELEASER_KEY)" \
		--volume $(CURDIR):/root/go/src/github.com/burnt-network/xion \
		--workdir /root/go/src/github.com/burnt-network/xion \
		$(GORELEASER_CROSS_IMAGE):$(GORELEASER_CROSS_VERSION) \
		release --config .goreleaser/release.yaml --snapshot --clean

release:
	$(DOCKER) run --rm \
		--env "GORELEASER_KEY=$(GORELEASER_KEY)" \
		--volume $(CURDIR):/root/go/src/github.com/burnt-network/xion \
		--workdir /root/go/src/github.com/burnt-network/xion \
		$(GORELEASER_CROSS_IMAGE):$(GORELEASER_CROSS_VERSION) \
		release --config .goreleaser/release.yaml --auto-snapshot --clean

# Help targets for build module
help-build-brief:
	@echo "  build                      Build the xiond binary"

help-build:
	@echo "Build targets:"
	@echo "  install                    Install the xiond binary"
	@echo "  build                      Build the xiond binary"
	@echo "  build-all                  Build all platforms using Docker"
	@echo "  build-local                Build for local platform using Docker"
	@echo "  build-linux-amd64          Build for Linux AMD64"
	@echo "  build-linux-arm64          Build for Linux ARM64"
	@echo "  build-darwin-amd64         Build for Darwin AMD64"
	@echo "  build-darwin-arm64         Build for Darwin ARM64"
	@echo "  build-windows-amd64        Build for Windows AMD64"
	@echo "  build-docker               Build Docker image"
	@echo "  build-heighliner           Build using Heighliner"
	@echo "  release-snapshot           Create release snapshot"
	@echo "  release                    Create production release"
	@echo ""

.PHONY: install build build-all build-local build-docker release-snapshot release \
        build-linux-arm64 build-linux-amd64 build-darwin-amd64 build-darwin-arm64 build-windows-amd64 \
        build-docker-arm64 build-docker-amd64 build-heighliner build-heighliner-amd64 build-heighliner-arm64 \
        help-build help-build-brief
