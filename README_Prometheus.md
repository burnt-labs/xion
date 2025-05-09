# Prometheus: Add README for xion

## Project Overview

Xion Daemon is a blockchain infrastructure project built on the Cosmos SDK and CosmWasm, designed to provide a robust and flexible blockchain platform with advanced features. The project is scaffolded from the CosmWasm/wasmd repository to ensure compatibility with the latest Cosmos SDK and CosmWasm releases.

### Key Features

- **Modular Blockchain Architecture**: Leverages Cosmos SDK for a flexible and extensible blockchain framework
- **WebAssembly Smart Contracts**: Integrated CosmWasm support for secure and efficient smart contract deployment
- **Advanced Authentication**: 
  - JWT (JSON Web Token) integration
  - WebAuthn support for enhanced account security
- **Flexible Fee Management**: 
  - Global fee module for network-wide fee configurations
  - Platform fee mechanisms
- **Customizable Minting**: Flexible mint module with configurable inflation and fee parameters
- **Token Factory**: Native support for creating and managing custom tokens

### Core Capabilities

The Xion Daemon provides developers and blockchain enthusiasts with a powerful platform for building decentralized applications (dApps) with advanced security, authentication, and economic models. Its modular design allows for easy customization and extension of blockchain functionality while maintaining high performance and compatibility with the broader Cosmos ecosystem.

## Getting Started, Installation, and Setup

### Prerequisites

- [Go](https://golang.org) (latest version recommended)
- [Make](https://www.gnu.org/software/make/)
- Optional: [Docker](https://docs.docker.com/get-docker/) (for advanced testing and protobuf operations)

### Quick Start

#### Installing the Binary

You have two options for installing the Xion Daemon:

1. Install from source:
```bash
# Clone the repository
git clone https://github.com/burnt-labs/xion.git
cd xion

# Install the binary
make install
```

2. Install via package manager:
For pre-built packages, refer to the [INSTALLERS.md](INSTALLERS.md) for detailed instructions for homebrew, apt, yum, and apk.

#### Verifying Installation

To verify the installation and check the version:

```bash
xiond version
```

### Development Setup

#### Building the Project

To build the project for development:

```bash
# Build the project
make build

# The binary will be located in the build directory
./build/xiond
```

### Running Tests

Xion provides multiple testing options:

```bash
# Run all tests
make test-all

# Run unit tests
make test

# Run integration tests
make test-integration

# Run specific integration tests
make test-integration-xion-token-factory
```

### Development Workflow

#### Formatting and Linting

Keep your code clean and consistent:

```bash
# Format code
make format

# Lint code
make lint
```

#### Managing Dependencies

```bash
# Download Go module dependencies
make go-mod-cache

# Verify dependencies
make go.sum
```

### Advanced Development Tasks

#### Protobuf Operations

```bash
# Generate protobuf files (requires Docker)
make proto-gen

# Format protobuf files
make proto-format

# Lint protobuf files
make proto-lint
```

### Cleaning Up

```bash
# Clean build artifacts
make clean

# Perform full clean including vendor directory
make distclean
```