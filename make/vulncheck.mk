# Dependency vulnerability scanning

GOVULNCHECK_VERSION ?= v1.7.0
GOVULNCHECK_ALLOWLIST ?= .github/govulncheck-allowlist.txt
# Exceptions that hold only because the code runs in the e2e harness. Only the
# e2e scan reads this list, so none of them can hide a finding in xiond.
GOVULNCHECK_ALLOWLIST_E2E ?= .github/govulncheck-allowlist-e2e.txt
# Pin the toolchain to what go.mod asks for. The standard library findings
# depend on the Go version doing the scanning, so a developer on a newer
# toolchain would otherwise see a different set from CI and from this
# repository's allowlist.
GOVULNCHECK_TOOLCHAIN := go$(shell sed -En 's/^go (.*)$$/\1/p' go.mod)
GOVULNCHECK_GATE := GOTOOLCHAIN=$(GOVULNCHECK_TOOLCHAIN) PATH="$(shell go env GOPATH)/bin:$$PATH" scripts/govulncheck-gate.sh

vulncheck-tools:
	go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)

# Gate the build on reachable advisories. Scans the chain and the e2e module;
# the e2e module is test-only, so it needs --test to be scanned at all.
vulncheck: vulncheck-tools
	@$(GOVULNCHECK_GATE) --name xiond --dir . --allowlist $(GOVULNCHECK_ALLOWLIST)
	@$(GOVULNCHECK_GATE) --name e2e_tests --dir e2e_tests --test --allowlist $(GOVULNCHECK_ALLOWLIST) --allowlist $(GOVULNCHECK_ALLOWLIST_E2E)

# Same scan, never fails. Use it to regenerate the allowlist after a dependency
# or toolchain change.
vulncheck-report: vulncheck-tools
	@$(GOVULNCHECK_GATE) --name xiond --dir . --allowlist $(GOVULNCHECK_ALLOWLIST) --report-only
	@$(GOVULNCHECK_GATE) --name e2e_tests --dir e2e_tests --test --allowlist $(GOVULNCHECK_ALLOWLIST) --allowlist $(GOVULNCHECK_ALLOWLIST_E2E) --report-only

# Help targets for vulncheck module
help-vulncheck-brief:
	@echo "  vulncheck                  Fail on reachable, un-allowlisted advisories"

help-vulncheck:
	@echo "Vulnerability targets:"
	@echo "  vulncheck                  Fail on reachable, un-allowlisted advisories"
	@echo "  vulncheck-report           Same scan, reports without failing"
	@echo "  vulncheck-tools            Install govulncheck"
	@echo ""

.PHONY: vulncheck vulncheck-report vulncheck-tools help-vulncheck help-vulncheck-brief
