# Protobuf generation and management

# Protobuf configuration
protoVer=0.17.1
protoImageName=ghcr.io/cosmos/proto-builder:$(protoVer)
protoImage=$(shell which docker) run --rm -v $(CURDIR):/workspace --workdir /workspace -e GOTOOLCHAIN=auto $(protoImageName)
HTTPS_GIT := https://github.com/burnt-labs/xion.git

# Generate all protobuf files with full pipeline
proto-all:
	@$(protoImage) sh -c " \
		echo '🚀 ========================================' && \
		echo '🚀 STARTING PROTOBUF BUILD PIPELINE' && \
		echo '🚀 ========================================' && \
		echo '' && \
		sh ./scripts/proto-gen.sh --gogo --pulsar --openapi && \
		echo '' && \
		echo '🔧 ========================================' && \
		echo '🔧 FORMATTING PROTOBUF FILES' && \
		echo '🔧 ========================================' && \
		find ./ -name '*.proto' -exec clang-format -i {} \; && \
		echo '✅ Protobuf formatting complete' && \
		echo '' && \
		echo '🔍 ========================================' && \
		echo '🔍 LINTING PROTOBUF FILES' && \
		echo '🔍 ========================================' && \
		buf lint --error-format=json && \
		echo '✅ Protobuf linting complete' && \
		echo '' && \
		echo '🔍 ========================================' && \
		echo '🔍 CHECKING FOR BREAKING CHANGES' && \
		echo '🔍 ========================================' && \
		buf breaking --against $(HTTPS_GIT)#branch=main \
	"

# Generate protobuf files
proto-gen:
	@echo "📦 ========================================"
	@echo "📦 GENERATING PROTOBUF FILES"
	@echo "📦 ========================================"
	@$(protoImage) sh ./scripts/proto-gen.sh
	@echo "✅ Protobuf generation complete"

# Generate OpenAPI documentation from protobuf
proto-gen-openapi:
	@echo "🌐 ========================================"
	@echo "🌐 GENERATING PROTOBUF OPENAPI"
	@echo "🌐 ========================================"
	@$(protoImage) sh ./scripts/proto-gen.sh --openapi
	@echo "✅ Protobuf OpenAPI generation complete"

# Alias for backward compatibility
proto-gen-swagger: proto-gen-openapi

# Generate Pulsar protobuf files
proto-gen-pulsar:
	@echo "⚡ ========================================"
	@echo "⚡ GENERATING PROTOBUF PULSAR"
	@echo "⚡ ========================================"
	@$(protoImage) sh ./scripts/proto-gen.sh --pulsar
	@echo "✅ Protobuf Pulsar generation complete"

# Format protobuf files
proto-format:
	@echo "🔧 ========================================"
	@echo "🔧 FORMATTING PROTOBUF FILES"
	@echo "🔧 ========================================"
	@$(protoImage) find ./ -name "*.proto" -exec clang-format -i {} \;
	@echo "✅ Protobuf formatting complete"

# Lint protobuf files
proto-lint:
	@echo "🔍 ========================================"
	@echo "🔍 LINTING PROTOBUF FILES"
	@echo "🔍 ========================================"
	@$(protoImage) buf lint --error-format=json
	@echo "✅ Protobuf linting complete"

# Check for breaking changes in protobuf files
proto-check-breaking:
	@echo "🔍 ========================================"
	@echo "🔍 CHECKING FOR BREAKING CHANGES"
	@echo "🔍 ========================================"
	@$(protoImage) buf breaking --against $(HTTPS_GIT)#branch=main

# Help targets for proto module
help-proto-brief:
	@echo "  proto-gen                  Generate protobuf files"

help-proto:
	@echo "Protobuf targets:"
	@echo "  proto-all                  Full protobuf pipeline"
	@echo "  proto-gen                  Generate protobuf files"
	@echo "  proto-gen-gogo             Generate gogo protobuf files"
	@echo "  proto-gen-openapi          Generate OpenAPI specs"
	@echo "  proto-gen-swagger          Generate Swagger specs"
	@echo "  proto-gen-pulsar           Generate pulsar protobuf files"
	@echo "  proto-format               Format protobuf files"
	@echo "  proto-lint                 Lint protobuf files"
	@echo "  proto-check-breaking       Check for breaking changes"
	@echo ""

.PHONY: proto-all proto-gen proto-gen-openapi proto-gen-swagger proto-gen-pulsar \
        proto-format proto-lint proto-check-breaking help-proto help-proto-brief
