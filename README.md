# Xion Daemon

The Xion Daemon is scaffolded off of [CosmWasm/wasmd](https://github.com/CosmWasm/wasmd)
rather than being scaffolded with ignite in order to more easily achieve
compatibility with the latest cosmos-sdk and CosmWasm releases.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Build Targets](#build-targets)
- [Test Targets](#test-targets)
- [Coverage Targets](#coverage-targets)
- [Protobuf Targets](#protobuf-targets)
- [Linting Targets](#linting-targets)
- [Development Targets](#development-targets)
- [Utility Targets](#utility-targets)
- [Help](#help)

## Prerequisites

- [golang](https://golang.org) - Go programming language
- [docker](https://docs.docker.com/get-docker/) - Required for protobuf generation and some build targets
- [heighliner](https://github.com/strangelove-ventures/heighliner) - Required for integration testing

## Quick Start

```bash
# Build and install xiond
make install

# Run tests with coverage
make test-cover

# Get help
make help
```

## Build Targets

| Target | Description |
|--------|-------------|
| `make install` | Install the xiond binary |
| `make build` | Build the xiond binary |
| `make build-all` | Build all platforms using Docker |
| `make build-local` | Build for local platform using Docker |
| `make build-linux-amd64` | Build for Linux AMD64 |
| `make build-linux-arm64` | Build for Linux ARM64 |
| `make build-darwin-amd64` | Build for Darwin AMD64 |
| `make build-darwin-arm64` | Build for Darwin ARM64 |
| `make build-windows-amd64` | Build for Windows AMD64 |
| `make build-docker` | Build Docker image |
| `make build-heighliner` | Build using Heighliner |
| `make release-snapshot` | Create release snapshot |
| `make release` | Create production release |

## Test Targets

| Target | Description |
|--------|-------------|
| `make test` | Run unit tests |
| `make test-unit` | Run unit tests |
| `make test-race` | Run tests with race detection |
| `make test-integration` | Run integration tests |
| `make compile-integration-tests` | Compile integration test binary |
| `make run-integration-test` | Run specific integration test |
| `make test-sim` | Run simulation tests |
| `make test-sim-import-export` | Run simulation import/export tests |
| `make test-sim-multi-seed-short` | Run multi-seed simulation tests |
| `make test-sim-deterministic` | Run deterministic simulation tests |

### Specific Integration Tests

```bash
make test-integration-min-fee
make test-integration-mint-module-inflation-high-fees
make test-integration-mint-module-inflation-low-fees
make test-integration-mint-module-inflation-no-fees
make test-integration-mint-module-no-inflation-no-fees
make test-integration-register-jwt-abstract-account
make test-integration-simulate
make test-integration-single-aa-mig
make test-integration-treasury-contract
make test-integration-treasury-multi
make test-integration-upgrade-ibc
make test-integration-upgrade-network
make test-integration-web-auth-n-abstract-account
make test-integration-xion-abstract-account
make test-integration-xion-abstract-account-event
make test-integration-xion-min-default
make test-integration-xion-min-multi-denom
make test-integration-xion-min-multi-denom-ibc
make test-integration-xion-min-zero
make test-integration-xion-send-platform-fee
make test-integration-xion-token-factory
make test-integration-xion-treasury-grants
make test-integration-xion-update-treasury-configs
make test-integration-xion-update-treasury-configs-aa
make test-integration-xion-update-treasury-params
```

## Coverage Targets

| Target | Description |
|--------|-------------|
| `make test-cover` | Run coverage analysis (development) |
| `make test-cover-ci` | Run coverage analysis (CI) |
| `make test-cover-validate` | Validate coverage thresholds |
| `make test-cover-html` | Generate HTML coverage report |
| `make test-cover-summary` | Show coverage summary |
| `make test-cover-analyze` | Detailed coverage analysis |
| `make test-cover-run` | Run tests with coverage |
| `make test-cover-filter` | Filter coverage report |
| `make test-cover-clean` | Clean coverage files |

### Coverage Configuration

The coverage system excludes certain packages and has configurable thresholds:

- **Threshold**: 85% total coverage required
- **Excluded packages**: `api/`, `cmd/` packages
- **Reports**: HTML, filtered, and detailed analysis available

## Protobuf Targets

**Note**: All protobuf commands require Docker

| Target | Description |
|--------|-------------|
| `make proto-all` | Full protobuf pipeline |
| `make proto-gen` | Generate protobuf files |
| `make proto-gen-gogo` | Generate gogo protobuf files |
| `make proto-gen-openapi` | Generate OpenAPI specs |
| `make proto-gen-swagger` | Generate Swagger specs |
| `make proto-gen-pulsar` | Generate pulsar protobuf files |
| `make proto-format` | Format protobuf files |
| `make proto-lint` | Lint protobuf files |
| `make proto-check-breaking` | Check for breaking changes |

## Linting Targets

| Target | Description |
|--------|-------------|
| `make lint` | Lint Go code |
| `make format` | Format Go code |
| `make format-tools` | Install formatting tools |

## Development Targets

| Target | Description |
|--------|-------------|
| `make go-mod-cache` | Download go modules to cache |
| `make draw-deps` | Generate dependency graph (requires Graphviz) |
| `make all` | Default target: install, lint, test |

## Utility Targets

| Target | Description |
|--------|-------------|
| `make clean` | Clean build artifacts |
| `make distclean` | Deep clean (includes vendor/) |
| `make guard-%` | Check environment variable is set |

## Help

The makefile includes comprehensive help documentation:

```bash
# Brief help with common targets
make help

# Complete help with all available targets
make help-full
```

### Modular Organization

The makefile is organized into modular components:

- `make/build.mk` - Build and release targets
- `make/test.mk` - Testing and simulation targets  
- `make/coverage.mk` - Coverage analysis targets
- `make/proto.mk` - Protobuf generation targets
- `make/lint.mk` - Code formatting and linting targets

Each module maintains its own help documentation to ensure targets and documentation stay synchronized.

## Examples

```bash
# Development workflow
make install && make test-cover

# CI workflow  
make build && make test-cover-ci

# Full protobuf regeneration
make proto-all

# Build for multiple platforms
make build-all

# Run specific integration test
make test-integration-xion-abstract-account

# Generate dependency visualization
make draw-deps
```
