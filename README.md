# Xion Daemon

The Xion Daemon is scaffolded off of [CosmWasm/wasmd](https://github.com/CosmWasm/wasmd)
rather than being scaffolded with ignite in order to more easily achieve
compatibility with the latest cosmos-sdk and CosmWasm releases.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Build](#build)
- [Testing Prerequisites](#testing-prerequisites)
- [Testing](#testing)
- [Tools & Dependencies](#tools--dependencies)
- [Linting](#linting)
- [Protobuf](#protobuf)
- [Cleaning](#cleaning)

## Prerequisites

- [golang](https://golang.org)

## Installation

To install the `xiond` binary:

```sh
make install
```

For Windows client:

```sh
make build-windows-client
```

## Build

To build the project:

```sh
make build
```

## Testing Prerequisites

- [golang](https://golang.org)
- [docker](https://docs.docker.com/get-docker/)
- [heighliner](https://github.com/strangelove-ventures/heighliner)

## Testing

There are various test targets available:

- `make test` - Run unit tests
- `make test-all` - Run all tests including unit, race, and coverage
- `make test-unit` - Run unit tests
- `make test-integration` - Run integration tests
- `make test-race` - Run tests with race condition detection
- `make test-cover` - Run tests with coverage
- `make benchmark` - Run benchmarks

## Specific Integration Tests

You can run specific integration tests by using the following commands:

```sh
make test-integration-dungeon-transfer-block
make test-integration-mint-module-no-inflation-no-fees
make test-integration-mint-module-inflation-high-fees
make test-integration-mint-module-inflation-low-fees
make test-integration-jwt-abstract-account
make test-integration-register-jwt-abstract-account
make test-integration-xion-send-platform-fee
make test-integration-xion-abstract-account
make test-integration-xion-min-default
make test-integration-xion-min-zero
make test-integration-xion-token-factory
make test-integration-xion-treasury-grants
make test-integration-min
make test-integration-web-auth-n-abstract-account
make test-integration-upgrade
make test-integration-upgrade-network
make test-integration-xion-mig
```

## Tools & Dependencies

To ensure all Go modules are downloaded:

```sh
make go-mod-cache
```

To verify dependencies:

```sh
make go.sum
```

To draw dependencies graph (requires Graphviz):

```sh
make draw-deps
```

## Linting

To format and lint the code:

```sh
make format
```

To just lint the code:

```sh
make lint
```

## Protobuf

*** Note: The prorobuf commands require Docker

To generate protobuf files:

```sh
make proto-gen
```

To format protobuf files:

```sh
make proto-format
```

To lint protobuf files:

```sh
make proto-lint
```

To check for breaking changes in protobuf files:

```sh
make proto-check-breaking
```

## Cleaning

To clean build artifacts:

```sh
make clean
```

To perform a full clean including vendor directory:

```sh
make distclean
```

For more detailed usage, refer to the individual make targets in the Makefile.
