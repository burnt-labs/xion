# Coverage testing configuration and targets

# Coverage configuration
COVERAGE_THRESHOLD ?= 85
COVERAGE_OUT ?= coverage.out
COVERAGE_FILTERED ?= coverage_filtered.out
COVERAGE_HTML ?= coverage.html
PACKAGES_SIMTEST = $(shell go list ./... | grep '/simulation')

# Coverage exclusions - patterns to ignore in low coverage reporting
COVERAGE_EXCLUSIONS := github.com/burnt-labs/xion/x/xion/client/cli/tx.go:.*NewSignCmd \
                       github.com/burnt-labs/xion/x/xion/keeper/grpc_query.go:.*WebAuthNVerifyRegister \
                       github.com/burnt-labs/xion/x/xion/keeper/mint.go:.*StakedInflationMintFn 

# Test exclusions - packages to skip during testing
TEST_EXCLUSIONS := github.com/burnt-labs/xion/api \
									 github.com/burnt-labs/xion/cmd
									 
TEST_EXCLUSIONS_PATTERN := $(shell echo "$(TEST_EXCLUSIONS)" | sed 's/ /\\|/g')

# Get testable packages, excluding configured patterns
GO_PACKAGES = $(shell go list ./...)
TESTABLE_PACKAGES = $(shell go list ./... |  grep -v '$(TEST_EXCLUSIONS_PATTERN)')

# Run tests with coverage on selected packages
test-cover-run:
	@echo "🧪 Running tests with coverage..."
	@echo "Testing packages (excluding: $(TEST_EXCLUSIONS))..."
	@set -o pipefail; go test $(TESTABLE_PACKAGES) -coverprofile=$(COVERAGE_OUT) -covermode=atomic -timeout=30m -race -tags='ledger test_ledger_mock' 2>&1 | { grep -v "has malformed LC_DYSYMTAB" | grep -v "DBG\|INF" | grep -v "params.*send_enabled" | grep -v "loadVersion\|SAVE TREE\|BATCH SAVE" | grep -v "Upgrading IAVL storage" | grep -v "Finished loading IAVL tree"; }

# Filter coverage report (remove generated files)
test-cover-filter: test-cover-run
	@echo "🔍 Filtering coverage report..."
	@if [ -f $(COVERAGE_OUT) ]; then \
		grep -v "\.pb\.go:" $(COVERAGE_OUT) | grep -v "\.pb\.gw\.go:" > $(COVERAGE_FILTERED); \
		echo "✅ Coverage filtered: $(COVERAGE_FILTERED)"; \
	else \
		echo "❌ Coverage file not found: $(COVERAGE_OUT)"; \
		exit 1; \
	fi

# Generate HTML coverage report
test-cover-html: test-cover-filter
	@echo "📊 Generating HTML coverage report..."
	@go tool cover -html=$(COVERAGE_FILTERED) -o $(COVERAGE_HTML)
	@echo "✅ HTML report generated: $(COVERAGE_HTML)"

# Show basic coverage summary
test-cover-summary: test-cover-filter
	@echo ""
	@echo "=== COVERAGE SUMMARY ==="
	@go tool cover -func=$(COVERAGE_FILTERED) | tail -1

# Validate coverage meets thresholds
test-cover-validate: test-cover-filter
	@echo ""
	@echo "=== COVERAGE VALIDATION ==="
	@TOTAL_COV=$$(go tool cover -func=$(COVERAGE_FILTERED) | tail -1 | awk '{print $$3}' | sed 's/%//'); \
	echo "Total coverage: $$TOTAL_COV%"; \
	if command -v bc >/dev/null 2>&1; then \
		if [ $$(echo "$$TOTAL_COV < $(COVERAGE_THRESHOLD)" | bc -l) -eq 1 ]; then \
			echo "❌ FAIL: Coverage $$TOTAL_COV% below $(COVERAGE_THRESHOLD)% threshold"; \
			exit 1; \
		else \
			echo "✅ PASS: Coverage $$TOTAL_COV% meets $(COVERAGE_THRESHOLD)% threshold"; \
		fi; \
	else \
		echo "⚠️  bc not available, skipping threshold validation"; \
	fi

# Advanced coverage analysis using script
test-cover-analyze: test-cover-filter
	@echo ""
	@echo "=== DETAILED COVERAGE ANALYSIS ==="
	@chmod +x scripts/coverage-analyze.sh
	@scripts/coverage-analyze.sh $(COVERAGE_FILTERED) "$(COVERAGE_EXCLUSIONS)"

# Full coverage workflow for CI
test-cover-ci: test-cover-validate test-cover-analyze
	@echo ""
	@echo "🎉 Coverage analysis complete!"

# Full coverage workflow for development
test-cover-dev: test-cover-html test-cover-analyze
	@echo ""
	@echo "🎉 Coverage analysis complete! Open $(COVERAGE_HTML) to view detailed report."

# Clean coverage files
test-cover-clean:
	@echo "🧹 Cleaning coverage files..."
	@rm -f $(COVERAGE_OUT) $(COVERAGE_FILTERED) $(COVERAGE_HTML)

# Legacy coverage target - now points to development workflow
test-cover: test-cover-dev

# Help targets for coverage module
help-coverage-brief:
	@echo "  test-cover                 Run coverage analysis"

help-coverage:
	@echo "Coverage targets:"
	@echo "  test-cover                 Run coverage analysis (development)"
	@echo "  test-cover-ci              Run coverage analysis (CI)"
	@echo "  test-cover-validate        Validate coverage thresholds"
	@echo "  test-cover-html            Generate HTML coverage report"
	@echo "  test-cover-summary         Show coverage summary"
	@echo "  test-cover-analyze         Detailed coverage analysis"
	@echo "  test-cover-run             Run tests with coverage"
	@echo "  test-cover-filter          Filter coverage report"
	@echo "  test-cover-clean           Clean coverage files"
	@echo ""

.PHONY: test-cover-run test-cover-filter test-cover-html test-cover-summary \
        test-cover-validate test-cover-analyze test-cover-ci test-cover-dev test-cover-clean test-cover \
        help-coverage help-coverage-brief
