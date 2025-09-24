#!/usr/bin/make -f

# Main Makefile with modular includes
# All variables and targets are now organized in the make/ directory

# Include all modular makefiles
include make/build.mk
include make/test.mk
include make/coverage.mk
include make/proto.mk
include make/lint.mk

# The below include contains the tools and runsim targets.
include contrib/devtools/Makefile

################################################################################
###                         Core Targets                                    ###
################################################################################

# Default targets
all: install lint test

################################################################################
###                         Tools & Dependencies                             ###
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

################################################################################
###                              Help Target                                ###
################################################################################

help:
	@echo "Xion - Blockchain Development Toolkit"
	@echo ""
	@echo "Common targets:"
	@$(MAKE) --no-print-directory help-build-brief
	@$(MAKE) --no-print-directory help-test-brief
	@$(MAKE) --no-print-directory help-coverage-brief
	@$(MAKE) --no-print-directory help-proto-brief
	@$(MAKE) --no-print-directory help-lint-brief
	@echo "  clean                      Clean build artifacts"
	@echo ""
	@echo "Use 'make help-full' for complete list of targets"

help-full:
	@echo "Xion - Blockchain Development Toolkit"
	@echo "======================================"
	@echo ""
	@$(MAKE) --no-print-directory help-build
	@$(MAKE) --no-print-directory help-test
	@$(MAKE) --no-print-directory help-coverage
	@$(MAKE) --no-print-directory help-proto
	@$(MAKE) --no-print-directory help-lint
	@echo "Development targets:"
	@echo "  go-mod-cache               Download go modules to cache"
	@echo "  draw-deps                  Generate dependency graph"
	@echo ""
	@echo "Utility targets:"
	@echo "  clean                      Clean build artifacts"
	@echo "  distclean                  Deep clean (includes vendor/)"
	@echo "  help                       Show brief help"
	@echo "  help-full                  Show this complete help"

.PHONY: all go-mod-cache draw-deps clean distclean help help-full
