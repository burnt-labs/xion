#!/usr/bin/make -f

PACKAGES_SIMTEST = $(shell go list ./... | grep '/simulation')
VERSION ?= $(shell echo $(shell git describe --tags) | sed 's/^v//')
COMMIT ?= $(shell git log -1 --format='%H')
LEDGER_ENABLED ?= true
SDK_PACK := $(shell go list -m github.com/cosmos/cosmos-sdk | sed  's/ /\@/g')
BINDIR ?= $(GOPATH)/bin
BUILDDIR ?= $(CURDIR)/build
SIMAPP = ./app
XION_IMAGE = xiond:local
XION_TEST_IMAGE = xiond:heighliner

# docker and goreleaser
DOCKER := $(shell which docker)
GORELEASER_CROSS_IMAGE ?= goreleaser/goreleaser-cross
GORELEASER_CROSS_VERSION ?= v1.22.7
# need custom image
GORELEASER_IMAGE ?= $(GORELEASER_CROSS_IMAGE)
GORELEASER_VERSION ?= $(GORELEASER_CROSS_VERSION)
GORELEASER_RELEASE ?= false
GORELEASER_SKIP_FLAGS ?= ""

# process build tags
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

whitespace :=
empty = $(whitespace) $(whitespace)
comma := ,
build_tags_comma_sep := $(subst $(empty),$(comma),$(build_tags))

# process linker flags

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

# The below include contains the tools and runsim targets.
include contrib/devtools/Makefile

all: install lint test

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
		--platform linux/amd64 \
		--volume $(CURDIR):/root/go/src/github.com/burnt-network/xion \
		--workdir /root/go/src/github.com/burnt-network/xion \
		$(GORELEASER_CROSS_IMAGE):$(GORELEASER_CROSS_VERSION) \
		build --clean --skip validate

build-local:
	$(DOCKER) run --rm \
		--env GOOS=$(shell go env -json | jq -r '.GOOS') \
		--env GOARCH=$(shell go env -json | jq -r '.GOARCH') \
		--platform linux/amd64 \
		--volume $(CURDIR):/root/go/src/github.com/burnt-network/xion \
		--workdir /root/go/src/github.com/burnt-network/xion \
		$(GORELEASER_CROSS_IMAGE):$(GORELEASER_CROSS_VERSION) \
		build --clean --single-target --skip validate

build-linux-arm64:
	$(DOCKER) run --rm \
		--env GOOS=linux \
		--env GOARCH=arm64 \
		--volume $(CURDIR):/root/go/src/github.com/burnt-network/xion \
		--workdir /root/go/src/github.com/burnt-network/xion \
		$(GORELEASER_IMAGE):$(GORELEASER_VERSION) \
		build --clean --single-target --skip validate

build-linux-amd64:
	$(DOCKER) run --rm \
		--env GOOS=linux \
		--env GOARCH=amd64 \
		--platform linux/amd64 \
		--volume $(CURDIR):/root/go/src/github.com/burnt-network/xion \
		--workdir /root/go/src/github.com/burnt-network/xion \
		$(GORELEASER_IMAGE):$(GORELEASER_VERSION) \
		build --clean --single-target --skip validate

build-darwin-amd64:
	$(DOCKER) run --rm \
		--env GOOS=darwin \
		--env GOARCH=amd64 \
		--platform linux/amd64 \
		--volume $(CURDIR):/root/go/src/github.com/burnt-network/xion \
		--workdir /root/go/src/github.com/burnt-network/xion \
		$(GORELEASER_CROSS_IMAGE):$(GORELEASER_CROSS_VERSION) \
		build --clean --single-target --skip validate

build-darwin-arm64:
	$(DOCKER) run --rm \
		--env GOOS=darwin \
		--env GOARCH=arm64 \
		--platform linux/amd64 \
		--volume $(CURDIR):/root/go/src/github.com/burnt-network/xion \
		--workdir /root/go/src/github.com/burnt-network/xion \
		$(GORELEASER_CROSS_IMAGE):$(GORELEASER_CROSS_VERSION) \
		build --clean --single-target --skip validate

build-darwin-all:
	$(DOCKER) run --rm \
		--platform linux/amd64 \
		--volume $(CURDIR):/root/go/src/github.com/burnt-network/xion \
		--workdir /root/go/src/github.com/burnt-network/xion \
		$(GORELEASER_CROSS_IMAGE):$(GORELEASER_CROSS_VERSION) \
		build --clean --id "xiond_darwin_amd64" --id "xiond_darwin_arm64" --skip validate

build-docker:
	$(DOCKER) build \
	  --target=release \
	  --progress=plain \
	  --tag $(XION_IMAGE) .

build-docker-amd64:
	$(DOCKER) build \
	  --platform linux/amd64 \
	  --target=release \
	  --progress=plain \
	  --tag $(XION_IMAGE) .

build-docker-arm64:
	$(DOCKER) build \
	  --platform linux/arm64 \
	  --target=release \
	  --progress=plain \
	  --tag $(XION_IMAGE) .

build-heighliner:
	$(DOCKER) build \
	  --target=heighliner \
	  --progress=plain \
	  --tag $(XION_TEST_IMAGE) .

build-heighliner-amd64:
	$(DOCKER) build \
	  --platform linux/amd64 \
	  --target=heighliner \
	  --progress=plain \
	  --tag $(XION_TEST_IMAGE) .

build-heighliner-arm64:
	$(DOCKER) build \
	  --platform linux/arm64 \
	  --target=heighliner \
	  --progress=plain \
	  --tag $(XION_TEST_IMAGE) .

release-dryrun:
	$(DOCKER) run --rm \
		--platform linux/amd64 \
		--volume $(CURDIR):/root/go/src/github.com/burnt-network/xion \
		--volume /var/run/docker.sock:/var/run/docker.sock \
		--workdir /root/go/src/github.com/burnt-network/xion \
		$(GORELEASER_CROSS_IMAGE):$(GORELEASER_CROSS_VERSION) \
		--clean --skip validate --skip publish

################################################################################
###                         Tools & dependencies                             ###
################################################################################

go-mod-cache: go.sum
	@echo "--> Download go modules to local cache"
	@go mod download

go.sum: go.mod
	@echo "--> Ensure dependencies have not been modified"
	@go mod verify

draw-deps:
	@# requires brew install graphviz or apt-get install graphviz
	go install github.com/RobotsAndPencils/goviz@latest
	@goviz -i ./cmd/xiond -d 2 | dot -Tpng -o dependency-graph.png

clean:
	rm -rf snapcraft-local.yaml build/

distclean: clean
	rm -rf vendor/

guard-%:
	@ if [ "${${*}}" = "" ]; then \
        echo "Environment variable $* not set"; \
        exit 1; \
	fi

###############################################################################
###                                Testing                                  ###
###############################################################################

test: test-unit
test-all: check test-race test-cover

test-version:
	@echo $(VERSION)

test-unit:
	@version=$(version) go test -mod=readonly -tags='ledger test_ledger_mock' ./...

compile_integration_tests:
	@cd integration_tests && go test -c

test-integration:
	@XION_TEST_IMAGE=$(XION_TEST_IMAGE) cd integration_tests && go test -mod=readonly -tags='ledger test_ledger_mock'  ./...

test-integration-dungeon-transfer-block: compile_integration_tests
	@XION_TEST_IMAGE=$(XION_TEST_IMAGE) ./integration_tests/integration_tests.test -test.failfast -test.v -test.run TestDungeonTransferBlock

test-integration-mint-module-no-inflation-no-fees: compile_integration_tests
	@XION_TEST_IMAGE=$(XION_TEST_IMAGE) ./integration_tests/integration_tests.test -test.failfast -test.v -test.run TestMintModuleNoInflationNoFees

test-integration-mint-module-inflation-no-fees: compile_integration_tests
	@XION_TEST_IMAGE=$(XION_TEST_IMAGE) ./integration_tests/integration_tests.test -test.failfast -test.v -test.run TestMintModuleInflationNoFees

test-integration-mint-module-inflation-high-fees: compile_integration_tests
	@XION_TEST_IMAGE=$(XION_TEST_IMAGE) ./integration_tests/integration_tests.test -test.failfast -test.v -test.run TestMintModuleInflationHighFees

test-integration-mint-module-inflation-low-fees: compile_integration_tests
	@XION_TEST_IMAGE=$(XION_TEST_IMAGE) ./integration_tests/integration_tests.test -test.failfast -test.v -test.run TestMintModuleInflationLowFees

test-integration-jwt-abstract-account: compile_integration_tests
	@XION_TEST_IMAGE=$(XION_TEST_IMAGE) ./integration_tests/integration_tests.test -test.failfast -test.v -test.run TestJWTAbstractAccount

test-integration-register-jwt-abstract-account: compile_integration_tests
	@XION_TEST_IMAGE=$(XION_TEST_IMAGE) ./integration_tests/integration_tests.test -test.failfast -test.v -test.run TestXionAbstractAccountJWTCLI

test-integration-xion-send-platform-fee: compile_integration_tests
	@XION_TEST_IMAGE=$(XION_TEST_IMAGE) ./integration_tests/integration_tests.test -test.failfast -test.v -test.run XionSendPlatformFee

test-integration-xion-abstract-account: compile_integration_tests
	@XION_TEST_IMAGE=$(XION_TEST_IMAGE) ./integration_tests/integration_tests.test -test.failfast -test.v -test.run XionAbstractAccount

test-integration-xion-min-default: compile_integration_tests
	@XION_TEST_IMAGE=$(XION_TEST_IMAGE) ./integration_tests/integration_tests.test -test.failfast -test.v -test.run TestXionMinimumFeeDefault

test-integration-xion-min-zero: compile_integration_tests
	@XION_TEST_IMAGE=$(XION_TEST_IMAGE) ./integration_tests/integration_tests.test -test.failfast -test.v -test.run TestXionMinimumFeeZero

test-integration-xion-token-factory: compile_integration_tests
	@XION_TEST_IMAGE=$(XION_TEST_IMAGE) ./integration_tests/integration_tests.test -test.failfast -test.v -test.run TestXionTokenFactory

test-integration-xion-treasury-grants: compile_integration_tests
	@XION_TEST_IMAGE=$(XION_TEST_IMAGE) ./integration_tests/integration_tests.test -test.failfast -test.v -test.run TestTreasuryContract

test-integration-xion-treasury-multi: compile_integration_tests
	@XION_TEST_IMAGE=$(XION_TEST_IMAGE) ./integration_tests/integration_tests.test -test.failfast -test.v -test.run TestTreasuryMulti

test-integration-min:
	@XION_TEST_IMAGE=$(XION_TEST_IMAGE) cd integration_tests && go test -v -run  TestXionMinimumFeeDefault -mod=readonly  -tags='ledger test_ledger_mock'  ./...

test-integration-web-auth-n-abstract-account: compile_integration_tests
	@XION_TEST_IMAGE=$(XION_TEST_IMAGE) ./integration_tests/integration_tests.test -test.failfast -test.v -test.run WebAuthNAbstractAccount

test-integration-upgrade:
	@XION_TEST_IMAGE=$(XION_TEST_IMAGE) cd integration_tests && go test -v -run TestXionUpgradeIBC -mod=readonly  -tags='ledger test_ledger_mock'  ./...

test-integration-upgrade-network:
	@XION_TEST_IMAGE=$(XION_TEST_IMAGE) cd integration_tests && go test -v -run TestXionUpgradeNetwork -mod=readonly  -tags='ledger test_ledger_mock'  ./...

test-integration-xion-mig: compile_integration_tests
	@XION_TEST_IMAGE=$(XION_TEST_IMAGE) ./integration_tests/integration_tests.test -test.failfast -test.v -test.run TestAbstractAccountMigration

test-integration-simulate: compile_integration_tests
	@XION_TEST_IMAGE=$(XION_TEST_IMAGE) ./integration_tests/integration_tests.test -test.failfast -test.v -test.run TestSimulate

test-race:
	@VERSION=$(VERSION) go test -mod=readonly -race -tags='ledger test_ledger_mock' ./...

test-cover:
	@go test -mod=readonly -timeout 30m -race -coverprofile=coverage.txt -covermode=atomic -tags='ledger test_ledger_mock' ./...

benchmark:
	@go test -mod=readonly -bench=. ./...

test-sim-import-export: runsim
	@echo "Running application import/export simulation. This may take several minutes..."
	@$(BINDIR)/runsim -Jobs=4 -SimAppPkg=$(SIMAPP) -ExitOnFail 50 5 TestAppImportExport

test-sim-multi-seed-short: runsim
	@echo "Running short multi-seed application simulation. This may take awhile!"
	@$(BINDIR)/runsim -Jobs=4 -SimAppPkg=$(SIMAPP) -ExitOnFail 50 5 TestFullAppSimulation

test-sim-deterministic: runsim
	@echo "Running short multi-seed application simulation. This may take awhile!"
	@$(BINDIR)/runsim -Jobs=4 -SimAppPkg=$(SIMAPP) -ExitOnFail 1 1 TestAppStateDeterminism

################################################################################
###                                 Linting                                  ###
################################################################################

format-tools:
	go install mvdan.cc/gofumpt@v0.4.0
	go install github.com/client9/misspell/cmd/misspell@v0.3.4
	go install golang.org/x/tools/cmd/goimports@latest

lint: format-tools
	golangci-lint run --tests=false
	find . -name '*.go' -type f -not -path "./vendor*" -not -path "*.git*" -not -path "*_test.go" -not -path "*.pb.go" -not -path "*.pb.gw.go" | xargs gofumpt -d

format: format-tools
	golangci-lint run --fix
	find . -name '*.go' -type f -not -path "./vendor*" -not -path "*.git*" -not -path "./client/lcd/statik/statik.go" -not -path "*.pb.go" -not -path "*.pb.gw.go" | xargs gofumpt -w
	find . -name '*.go' -type f -not -path "./vendor*" -not -path "*.git*" -not -path "./client/lcd/statik/statik.go" -not -path "*.pb.go" -not -path "*.pb.gw.go" | xargs misspell -w
	find . -name '*.go' -type f -not -path "./vendor*" -not -path "*.git*" -not -path "./client/lcd/statik/statik.go" -not -path "*.pb.go" -not -path "*.pb.gw.go" | xargs goimports -w -local github.com/burnt-labs/xiond


################################################################################
###                                 Protobuf                                 ###
################################################################################
protoVer=0.14.0
protoImageName=ghcr.io/cosmos/proto-builder:$(protoVer)
protoImage=$(DOCKER) run --rm -v $(CURDIR):/workspace --workdir /workspace $(protoImageName)
HTTPS_GIT := https://github.com/burnt-labs/xion.git

proto-all: proto-format proto-lint proto-gen format

proto-gen:
	@echo "Generating Protobuf files"
	@$(protoImage) sh ./scripts/protocgen.sh

proto-swagger-gen:
	@echo "Generating Protobuf Swagger"
	@$(protoImage) sh 'scripts/protoc-swagger-gen.sh'

proto-format:
	@echo "Formatting Protobuf files"
	@$(protoImage) find ./ -name "*.proto" -exec clang-format -i {} \;

proto-lint:
	@$(protoImage) buf lint --error-format=json

proto-check-breaking:
	@$(protoImage) buf breaking --against $(HTTPS_GIT)#branch=main

.PHONY: all install install-debug \
	go-mod-cache draw-deps clean build format \
	test test-all test-build test-cover test-unit test-race \
	test-sim-import-export build-windows-client \
