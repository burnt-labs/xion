# Barretenberg build targets

# Barretenberg git reference (branch, tag, or commit hash)
# Note: Barretenberg doesn't use semantic version tags - use branch or commit
BB_REF ?= master

# Directory paths
BB_DIR := x/zk/barretenberg
BB_WRAPPER_DIR := $(BB_DIR)/wrapper
BB_LIB_DIR := $(BB_DIR)/lib
BB_TESTDATA_DIR := $(BB_DIR)/testdata

# Detect current platform
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
BB_PLATFORM := $(GOOS)_$(GOARCH)

.PHONY: barretenberg-build barretenberg-build-stub barretenberg-build-full \
        barretenberg-build-native barretenberg-build-all \
        barretenberg-build-docker barretenberg-clean barretenberg-test \
        barretenberg-generate-testdata barretenberg-verify

# Build barretenberg static library for current platform
# By default, build the stub for development. Use barretenberg-build-full for real library.
barretenberg-build: barretenberg-build-stub

# Build stub library (for development/testing without full Barretenberg)
barretenberg-build-stub:
	@echo "Building Barretenberg stub for $(BB_PLATFORM)..."
	$(MAKE) -f Makefile.stub -C $(BB_WRAPPER_DIR)

# Build full library from source (requires all dependencies)
barretenberg-build-full: barretenberg-build-native

# Build for native platform
barretenberg-build-native:
	@echo "Building Barretenberg for $(BB_PLATFORM)..."
	cd $(BB_WRAPPER_DIR) && ./build.sh --bb-ref $(BB_REF)

# Build for all supported platforms using Docker
barretenberg-build-all:
	@echo "Building Barretenberg for all platforms..."
	cd $(BB_WRAPPER_DIR) && ./build.sh --all --docker --bb-ref $(BB_REF)

# Build using Docker (for cross-compilation)
barretenberg-build-docker:
	@echo "Building Barretenberg for $(BB_PLATFORM) using Docker..."
	cd $(BB_WRAPPER_DIR) && ./build.sh --docker --platform $(BB_PLATFORM) --bb-ref $(BB_REF)

# Build for specific platforms
barretenberg-build-linux-amd64:
	cd $(BB_WRAPPER_DIR) && ./build.sh --docker --platform linux_amd64 --bb-ref $(BB_REF)

barretenberg-build-linux-arm64:
	cd $(BB_WRAPPER_DIR) && ./build.sh --docker --platform linux_arm64 --bb-ref $(BB_REF)

barretenberg-build-darwin-arm64:
	cd $(BB_WRAPPER_DIR) && ./build.sh --platform darwin_arm64 --bb-ref $(BB_REF)

# Clean build artifacts
barretenberg-clean:
	@echo "Cleaning Barretenberg build artifacts..."
	rm -rf $(BB_WRAPPER_DIR)/build
	rm -f $(BB_LIB_DIR)/*/libbarretenberg.a

# Run barretenberg package tests
barretenberg-test:
	@echo "Testing Barretenberg bindings..."
	go test -v -race ./$(BB_DIR)/...

# Run barretenberg package benchmarks
barretenberg-bench:
	@echo "Benchmarking Barretenberg bindings..."
	go test -v -bench=. -benchmem ./$(BB_DIR)/...

# Generate test vectors (requires Noir and bb CLI)
barretenberg-generate-testdata:
	@echo "Generating Barretenberg test vectors..."
	cd $(BB_TESTDATA_DIR) && ./generate.sh

# Verify that bindings compile (doesn't run tests, just builds)
barretenberg-verify:
	@echo "Verifying Barretenberg bindings compile..."
	@if [ -f "$(BB_LIB_DIR)/$(BB_PLATFORM)/libbarretenberg.a" ]; then \
		go build ./$(BB_DIR)/...; \
		echo "Barretenberg bindings compile successfully"; \
	else \
		echo "Warning: Static library not found for $(BB_PLATFORM)"; \
		echo "Run 'make barretenberg-build' first"; \
		exit 1; \
	fi

# Help target for barretenberg module
help-barretenberg-brief:
	@echo "  barretenberg-build         Build Barretenberg static library"

help-barretenberg:
	@echo "Barretenberg targets:"
	@echo "  barretenberg-build          Build stub library (default, for dev)"
	@echo "  barretenberg-build-stub     Build stub library for development"
	@echo "  barretenberg-build-full     Build full library from source"
	@echo "  barretenberg-build-native   Build full library (native)"
	@echo "  barretenberg-build-docker   Build full library (Docker)"
	@echo "  barretenberg-build-all      Build for all platforms (Docker)"
	@echo "  barretenberg-build-linux-amd64   Build for Linux AMD64"
	@echo "  barretenberg-build-linux-arm64   Build for Linux ARM64"
	@echo "  barretenberg-build-darwin-arm64  Build for Darwin ARM64"
	@echo "  barretenberg-clean          Clean build artifacts"
	@echo "  barretenberg-test           Run package tests"
	@echo "  barretenberg-bench          Run benchmarks"
	@echo "  barretenberg-generate-testdata   Generate test vectors"
	@echo "  barretenberg-verify         Verify bindings compile"
	@echo ""
	@echo "  Note: Use barretenberg-build-stub for development without"
	@echo "        full Barretenberg dependencies. For production, build"
	@echo "        the full library or use pre-built binaries."
	@echo ""
