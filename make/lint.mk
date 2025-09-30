# Linting and formatting targets

# Install formatting tools
format-tools:
	go install mvdan.cc/gofumpt@v0.4.0
	go install github.com/client9/misspell/cmd/misspell@v0.3.4
	go install golang.org/x/tools/cmd/goimports@latest

# Lint Go code
lint: format-tools
	golangci-lint run --tests=false
	find . -name '*.go' -type f -not -path "./api/*" -not -path "*.git*" -not -path "*_test.go" -not -path "*.pb.go" -not -path "*.pb.gw.go" | xargs gofumpt -d

# Format Go code
format: format-tools
	golangci-lint run --fix
	find . -name '*.go' -type f -not -path "./api/*" -not -path "*.git*" -not -path "*.pb.go" -not -path "*.pb.gw.go" | xargs gofumpt -w
	find . -name '*.go' -type f -not -path "./api/*" -not -path "*.git*" -not -path "*.pb.go" -not -path "*.pb.gw.go" | xargs misspell -w
	find . -name '*.go' -type f -not -path "./api/*" -not -path "*.git*" -not -path "*.pb.go" -not -path "*.pb.gw.go" | xargs goimports -w -local github.com/burnt-labs/xiond

# Help targets for lint module
help-lint-brief:
	@echo "  lint                       Lint and format code"

help-lint:
	@echo "Linting targets:"
	@echo "  lint                       Lint Go code"
	@echo "  format                     Format Go code"
	@echo "  format-tools               Install formatting tools"
	@echo ""

.PHONY: format-tools lint format help-lint help-lint-brief
