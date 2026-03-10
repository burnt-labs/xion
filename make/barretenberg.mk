# Barretenberg build targets

# Barretenberg version — pinned to aztec-packages tag with pre-built libbb-external.a.
# The version is also embedded inside build-wrapper.sh (BB_AZTEC_TAG) for traceability.
BB_REF ?= v4.0.4

# Directory paths
BB_DIR := x/zk/barretenberg
BB_WRAPPER_DIR := $(BB_DIR)/wrapper
BB_LIB_DIR := $(BB_DIR)/lib
BB_TESTDATA_DIR := $(BB_DIR)/testdata

# Detect current platform
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
BB_PLATFORM := $(GOOS)_$(GOARCH)

.PHONY: barretenberg-build barretenberg-build-stub barretenberg-build-wrapper \
        barretenberg-build-linux-amd64 barretenberg-build-darwin-amd64 barretenberg-build-darwin-arm64 \
        barretenberg-clean barretenberg-test \
        barretenberg-generate-testdata barretenberg-verify

# Build libbarretenberg.a for the current platform using the pinned pre-built Aztec static lib.
barretenberg-build: barretenberg-build-wrapper

# Build stub library (for development/testing without the real Barretenberg library).
# The stub links a no-op static lib built from barretenberg_stub.cpp via stub.mk.
barretenberg-build-stub:
	@echo "Building Barretenberg stub for $(BB_PLATFORM)..."
	$(MAKE) -f stub.mk -C $(BB_WRAPPER_DIR)

# Download the pinned Aztec libbb-external.a, compile the C++ wrapper shim against it,
# and merge into lib/$(BB_PLATFORM)/libbarretenberg.a.
# Version is pinned inside build-wrapper.sh (BB_AZTEC_TAG=$(BB_REF)).
barretenberg-build-wrapper:
	@echo "Building Barretenberg wrapper for $(BB_PLATFORM) (pinned: $(BB_REF))..."
	$(BB_WRAPPER_DIR)/build-wrapper.sh --platform $(BB_PLATFORM)

# Per-platform convenience targets
barretenberg-build-linux-amd64:
	$(BB_WRAPPER_DIR)/build-wrapper.sh --platform linux_amd64

barretenberg-build-darwin-arm64:
	$(BB_WRAPPER_DIR)/build-wrapper.sh --platform darwin_arm64

barretenberg-build-darwin-amd64:
	$(BB_WRAPPER_DIR)/build-wrapper.sh --platform darwin_amd64

# Clean build artifacts, FetchContent cache, and stub objects
barretenberg-clean:
	@echo "Cleaning Barretenberg build artifacts, cache, and stub..."
	rm -rf $(BB_WRAPPER_DIR)/build
	rm -f $(BB_WRAPPER_DIR)/*.o
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
	@echo "  barretenberg-build              Build libbarretenberg.a for current platform (default)"
	@echo "  barretenberg-build-wrapper      Download pinned Aztec lib + compile wrapper shim"
	@echo "  barretenberg-build-stub         Build stub library for development (no real lib needed)"
	@echo "  barretenberg-build-linux-amd64  Build for Linux AMD64"
	@echo "  barretenberg-build-darwin-arm64 Build for Darwin ARM64"
	@echo "  barretenberg-build-darwin-amd64 Build for Darwin AMD64"
	@echo "  barretenberg-clean              Clean build artifacts"
	@echo "  barretenberg-test               Run package tests"
	@echo "  barretenberg-bench              Run benchmarks"
	@echo "  barretenberg-generate-testdata  Generate test vectors"
	@echo "  barretenberg-verify             Verify bindings compile"
	@echo ""
	@echo "  Supported platforms: linux_amd64, darwin_amd64, darwin_arm64"
	@echo "  (linux_arm64 is not supported — no pre-built Aztec static lib available)"
	@echo ""
