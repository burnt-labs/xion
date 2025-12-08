# Test targets and configuration

SIMAPP = ./app
BINDIR ?= $(GOPATH)/bin
export XION_IMAGE ?= xiond:local

# Target to ensure Docker image exists (only runs once when needed)
.ensure-docker-image:
	if ! docker images --format "{{.Repository}}:{{.Tag}}" | grep -q "^$(XION_IMAGE)$$"; then \
		echo "Docker image $(XION_IMAGE) not found. Building..."; \
		$(MAKE) -f make/build.mk build-docker XION_IMAGE=$(XION_IMAGE); \
	else \
		echo "Docker image $(XION_IMAGE) found."; \
	fi

.PHONY: .ensure-docker-image

test: test-unit
test-all: check test-race test-cover

benchmark:
	@go test -mod=readonly -bench=. ./...

test-unit:
	@version=$(version) go test -mod=readonly -tags='ledger test_ledger_mock' -ldflags="-w" ./...

test-race:
	@version=$(version) go test -mod=readonly -race -tags='ledger test_ledger_mock' -ldflags="-w" ./...

test-e2e-all: .ensure-docker-image
	@cd ./e2e_tests && go test -mod=readonly -tags='ledger test_ledger_mock' -ldflags="-w" ./abstract-account/... ./app/... ./dkim/... ./indexer/... ./jwk/... ./xion/...

test-aa-all: .ensure-docker-image
	@cd ./e2e_tests/abstract-account && go test -mod=readonly -tags='ledger test_ledger_mock' -ldflags="-w" -v ./...

test-app-all: .ensure-docker-image
	@cd ./e2e_tests/app && go test -mod=readonly -tags='ledger test_ledger_mock' -ldflags="-w" -v ./...

test-dkim-all: .ensure-docker-image
	@cd ./e2e_tests/dkim && go test -mod=readonly -tags='ledger test_ledger_mock' -ldflags="-w" -v ./...

test-jwk-all: .ensure-docker-image
	@cd ./e2e_tests/jwk && go test -mod=readonly -tags='ledger test_ledger_mock' -ldflags="-w" -v ./...

test-xion-all: .ensure-docker-image
	@cd ./e2e_tests/xion && go test -mod=readonly -tags='ledger test_ledger_mock' -ldflags="-w" -v ./...

test-indexer-all: .ensure-docker-image
	@cd ./e2e_tests/indexer && go test -mod=readonly -tags='ledger test_ledger_mock' -ldflags="-w" -v ./...

test-run: .ensure-docker-image
	@echo "Running test: $(TEST_NAME) in directory: $(DIR_NAME)"
	@cd ./e2e_tests/$(DIR_NAME) && go test -mod=readonly -tags='ledger test_ledger_mock' -ldflags="-w" -failfast -v -run $(TEST_NAME) ./...

# Abstract Account Module Tests
test-aa-basic:
	$(MAKE) test-run DIR_NAME=abstract-account TEST_NAME=TestAABasic

test-aa-client-event:
	$(MAKE) test-run DIR_NAME=abstract-account TEST_NAME=TestAAClientEvent

test-aa-jwt-cli:
	$(MAKE) test-run DIR_NAME=abstract-account TEST_NAME=TestAAJWTCLI

test-aa-multi-auth:
	$(MAKE) test-run DIR_NAME=abstract-account TEST_NAME=TestAAMultiAuth

test-aa-panic:
	$(MAKE) test-run DIR_NAME=abstract-account TEST_NAME=TestAAPanic

test-aa-single-migration:
	$(MAKE) test-run DIR_NAME=abstract-account TEST_NAME=TestAASingleMigration

test-aa-webauthn:
	$(MAKE) test-run DIR_NAME=abstract-account TEST_NAME=TestAAWebAuthn

# App Module Tests
test-app-governance:
	$(MAKE) test-run DIR_NAME=app TEST_NAME=TestAppGovernance

test-app-ibc-timeout:
	$(MAKE) test-run DIR_NAME=app TEST_NAME=TestAppIBCTimeout

test-app-ibc-transfer:
	$(MAKE) test-run DIR_NAME=app TEST_NAME=TestAppIBCTransfer

test-app-mint-inflation-high-fees:
	$(MAKE) test-run DIR_NAME=app TEST_NAME=TestAppMintInflationHighFees

test-app-mint-inflation-low-fees:
	$(MAKE) test-run DIR_NAME=app TEST_NAME=TestAppMintInflationLowFees

test-app-mint-inflation-no-fees:
	$(MAKE) test-run DIR_NAME=app TEST_NAME=TestAppMintInflationNoFees

test-app-mint-no-inflation-no-fees:
	$(MAKE) test-run DIR_NAME=app TEST_NAME=TestAppMintNoInflationNoFees

test-app-send-platform-fee:
	$(MAKE) test-run DIR_NAME=app TEST_NAME=TestAppSendPlatformFee

test-app-simulate:
	$(MAKE) test-run DIR_NAME=app TEST_NAME=TestAppSimulate

test-app-token-factory:
	$(MAKE) test-run DIR_NAME=app TEST_NAME=TestAppTokenFactory

test-app-treasury-contract:
	$(MAKE) test-run DIR_NAME=app TEST_NAME=TestAppTreasuryContract

test-app-treasury-grants:
	$(MAKE) test-run DIR_NAME=app TEST_NAME=TestAppTreasuryContract

test-app-treasury-multi:
	$(MAKE) test-run DIR_NAME=app TEST_NAME=TestAppTreasuryMulti

test-app-update-treasury-configs:
	$(MAKE) test-run DIR_NAME=app TEST_NAME=TestAppUpdateTreasuryConfigs configUrl="$(configUrl)"

test-app-update-treasury-configs-aa:
	$(MAKE) test-run DIR_NAME=app TEST_NAME=TestAppUpdateTreasuryConfigsAA configUrl="$(configUrl)"

test-app-update-treasury-params:
	$(MAKE) test-run DIR_NAME=app TEST_NAME=TestAppUpdateTreasuryParams

test-app-upgrade-ibc:
	$(MAKE) test-run DIR_NAME=app TEST_NAME=TestAppUpgradeIBC

test-app-upgrade-network:
	$(MAKE) test-run DIR_NAME=app TEST_NAME=TestAppUpgradeNetwork

# v24 upgrade tests removed - v24 migration was flawed and has been replaced by v25
# TODO: Add v25 upgrade tests once migrator is implemented

# DKIM Module Tests
test-dkim-governance:
	$(MAKE) test-run DIR_NAME=dkim TEST_NAME=TestDKIMGovernance

test-dkim-key-revocation:
	$(MAKE) test-run DIR_NAME=dkim TEST_NAME=TestDKIMKeyRevocation

test-dkim-module:
	$(MAKE) test-run DIR_NAME=dkim TEST_NAME=TestDKIMModule

test-dkim-zk-email:
	$(MAKE) test-run DIR_NAME=dkim TEST_NAME=TestDKIMZKEmail

test-dkim-zk-proof:
	$(MAKE) test-run DIR_NAME=dkim TEST_NAME=TestZKEmailAuthenticator

# Indexer Module Tests
test-indexer-authz-create:
	$(MAKE) test-run DIR_NAME=indexer TEST_NAME=TestIndexerAuthzCreate

test-indexer-authz-multiple:
	$(MAKE) test-run DIR_NAME=indexer TEST_NAME=TestIndexerAuthzMultiple

test-indexer-authz-revoke:
	$(MAKE) test-run DIR_NAME=indexer TEST_NAME=TestIndexerAuthzRevoke

test-indexer-feegrant-create:
	$(MAKE) test-run DIR_NAME=indexer TEST_NAME=TestIndexerFeeGrantCreate

test-indexer-feegrant-multiple:
	$(MAKE) test-run DIR_NAME=indexer TEST_NAME=TestIndexerFeeGrantMultiple

test-indexer-feegrant-periodic:
	$(MAKE) test-run DIR_NAME=indexer TEST_NAME=TestIndexerFeeGrantPeriodic

test-indexer-feegrant-revoke:
	$(MAKE) test-run DIR_NAME=indexer TEST_NAME=TestIndexerFeeGrantRevoke

# JWK Module Tests
test-jwk-algorithm-confusion:
	$(MAKE) test-run DIR_NAME=jwk TEST_NAME=TestJWKAlgorithmConfusion

test-jwk-audience-mismatch:
	$(MAKE) test-run DIR_NAME=jwk TEST_NAME=TestJWKAudienceMismatch

test-jwk-expired-token:
	$(MAKE) test-run DIR_NAME=jwk TEST_NAME=TestJWKExpiredToken

test-jwk-invalid-signature:
	$(MAKE) test-run DIR_NAME=jwk TEST_NAME=TestJWKInvalidSignature

test-jwk-jwt-aa:
	$(MAKE) test-run DIR_NAME=jwk TEST_NAME=TestJWKJWTAA

test-jwk-key-rotation:
	$(MAKE) test-run DIR_NAME=jwk TEST_NAME=TestJWKKeyRotation

test-jwk-malformed-tokens:
	$(MAKE) test-run DIR_NAME=jwk TEST_NAME=TestJWKMalformedTokens

test-jwk-missing-claims:
	$(MAKE) test-run DIR_NAME=jwk TEST_NAME=TestJWKMissingClaims

test-jwk-multiple-audiences:
	$(MAKE) test-run DIR_NAME=jwk TEST_NAME=TestJWKMultipleAudiences

test-jwk-transaction-hash:
	$(MAKE) test-run DIR_NAME=jwk TEST_NAME=TestJWKTransactionHash

# Xion Module Tests
test-xion-genesis-export-import:
	$(MAKE) test-run DIR_NAME=xion TEST_NAME=TestGenesisExportImport

test-xion-indexer-authz:
	$(MAKE) test-run DIR_NAME=xion TEST_NAME=TestXionIndexerAuthz

test-xion-indexer-feegrant:
	$(MAKE) test-run DIR_NAME=xion TEST_NAME=TestXionIndexerFeeGrant

test-xion-indexer-non-consensus-critical:
	$(MAKE) test-run DIR_NAME=xion TEST_NAME=TestIndexerNonConsensusCritical

test-xion-min-fee-bypass:
	$(MAKE) test-run DIR_NAME=xion TEST_NAME=TestXionMinFeeBypass

test-xion-min-fee-default:
	$(MAKE) test-run DIR_NAME=xion TEST_NAME=TestXionMinFeeDefault

test-xion-min-fee-multi-denom:
	$(MAKE) test-run DIR_NAME=xion TEST_NAME=TestXionMinFeeMultiDenom

test-xion-min-fee-multi-denom-ibc:
	$(MAKE) test-run DIR_NAME=xion TEST_NAME=TestXionMinFeeMultiDenomIBC

test-xion-min-fee-zero:
	$(MAKE) test-run DIR_NAME=xion TEST_NAME=TestXionMinFeeZero

test-xion-platform-fee:
	$(MAKE) test-run DIR_NAME=xion TEST_NAME=TestXionPlatformFee

test-xion-platform-fee-bypass:
	$(MAKE) test-run DIR_NAME=xion TEST_NAME=TestXionPlatformFeeBypass

test-xion-platform-min-codec-bug:
	$(MAKE) test-run DIR_NAME=xion TEST_NAME=TestXionPlatformMinCodecBug

test-xion-platform-min-direct:
	$(MAKE) test-run DIR_NAME=xion TEST_NAME=TestXionPlatformMinDirect

# New comprehensive MinFee tests (functional tests only)
test-xion-min-fee-multi-denom-advanced:
	$(MAKE) test-run DIR_NAME=xion TEST_NAME=TestXionMinFeeMultiDenomAdvanced

test-xion-min-fee-extreme-values:
	$(MAKE) test-run DIR_NAME=xion TEST_NAME=TestXionMinFeeExtremeValues

test-xion-min-fee-concurrent-transactions:
	$(MAKE) test-run DIR_NAME=xion TEST_NAME=TestXionMinFeeConcurrentTransactions

test-xion-min-fee-sequence-handling:
	$(MAKE) test-run DIR_NAME=xion TEST_NAME=TestXionMinFeeSequenceHandling

test-xion-platform-minimum-with-fees:
	$(MAKE) test-run DIR_NAME=xion TEST_NAME=TestXionPlatformMinimumWithFees

test-xion-platform-minimum-codec:
	$(MAKE) test-run DIR_NAME=xion TEST_NAME=TestXionPlatformMinimumCodecValidation

test-xion-platform-minimum-bypass:
	$(MAKE) test-run DIR_NAME=xion TEST_NAME=TestXionPlatformMinimumBypassInteraction

test-xion-min-fee-error-messages:
	$(MAKE) test-run DIR_NAME=xion TEST_NAME=TestXionMinFeeErrorMessages

test-xion-min-fee-insufficient-balance:
	$(MAKE) test-run DIR_NAME=xion TEST_NAME=TestXionMinFeeInsufficientBalance

test-xion-min-fee-edge-cases:
	$(MAKE) test-run DIR_NAME=xion TEST_NAME=TestXionMinFeeEdgeCaseScenarios

test-xion-min-fee-mempool:
	$(MAKE) test-run DIR_NAME=xion TEST_NAME=TestXionMinFeeMemPoolBehavior

# Critical Security Tests (Priority 1)
test-xion-min-fee-gas-cap-boundaries:
	$(MAKE) test-run DIR_NAME=xion TEST_NAME=TestXionMinFeeGasCapBoundaries

test-xion-min-fee-gas-cap-with-fees:
	$(MAKE) test-run DIR_NAME=xion TEST_NAME=TestXionMinFeeGasCapWithFees

test-xion-min-fee-gas-cap-multiple-messages:
	$(MAKE) test-run DIR_NAME=xion TEST_NAME=TestXionMinFeeGasCapMultipleMessages

test-xion-min-fee-with-feegrant:
	$(MAKE) test-run DIR_NAME=xion TEST_NAME=TestXionMinFeeWithFeeGrant

test-xion-min-fee-feegrant-allowance-types:
	$(MAKE) test-run DIR_NAME=xion TEST_NAME=TestXionMinFeeFeeGrantAllowanceTypes

test-xion-min-fee-feegrant-expiration:
	$(MAKE) test-run DIR_NAME=xion TEST_NAME=TestXionMinFeeFeeGrantExpiration

test-xion-min-fee-multiple-feegrants:
	$(MAKE) test-run DIR_NAME=xion TEST_NAME=TestXionMinFeeMultipleFeeGrants

test-xion-min-fee-multi-message-mixed-types:
	$(MAKE) test-run DIR_NAME=xion TEST_NAME=TestXionMinFeeMultiMessageMixedTypes

test-xion-min-fee-multi-message-same-type:
	$(MAKE) test-run DIR_NAME=xion TEST_NAME=TestXionMinFeeMultiMessageSameType

test-xion-min-fee-multi-message-gas-accounting:
	$(MAKE) test-run DIR_NAME=xion TEST_NAME=TestXionMinFeeMultiMessageGasAccounting

test-xion-min-fee-multi-message-with-feegrant:
	$(MAKE) test-run DIR_NAME=xion TEST_NAME=TestXionMinFeeMultiMessageWithFeeGrant

test-xion-min-fee-multi-message-error-paths:
	$(MAKE) test-run DIR_NAME=xion TEST_NAME=TestXionMinFeeMultiMessageErrorPaths

test-xion-min-fee-multi-message-sequential:
	$(MAKE) test-run DIR_NAME=xion TEST_NAME=TestXionMinFeeMultiMessageSequential

test-xion-min-fee-bypass-message-types:
	$(MAKE) test-run DIR_NAME=xion TEST_NAME=TestXionMinFeeBypassMessageTypes

# Grouped test targets for new MinFee coverage
test-xion-min-fee-coverage-all: \
	test-xion-min-fee-multi-denom-advanced \
	test-xion-min-fee-extreme-values \
	test-xion-min-fee-concurrent-transactions \
	test-xion-min-fee-sequence-handling \
	test-xion-platform-minimum-with-fees \
	test-xion-platform-minimum-codec \
	test-xion-platform-minimum-bypass \
	test-xion-min-fee-error-messages \
	test-xion-min-fee-insufficient-balance \
	test-xion-min-fee-edge-cases \
	test-xion-min-fee-mempool

# Grouped test targets for critical security tests
# All tests now functional with real transaction execution
test-xion-min-fee-critical-all: \
	test-xion-min-fee-gas-cap-boundaries \
	test-xion-min-fee-gas-cap-with-fees \
	test-xion-min-fee-gas-cap-multiple-messages \
	test-xion-min-fee-with-feegrant \
	test-xion-min-fee-feegrant-allowance-types \
	test-xion-min-fee-feegrant-expiration \
	test-xion-min-fee-multiple-feegrants \
	test-xion-min-fee-multi-message-mixed-types \
	test-xion-min-fee-multi-message-same-type \
	test-xion-min-fee-multi-message-gas-accounting \
	test-xion-min-fee-multi-message-with-feegrant \
	test-xion-min-fee-multi-message-error-paths \
	test-xion-min-fee-multi-message-sequential \
	test-xion-min-fee-bypass-message-types

# Run all MinFee tests (old + new + critical)
test-xion-min-fee-all: \
	test-xion-min-fee-bypass \
	test-xion-min-fee-default \
	test-xion-min-fee-multi-denom \
	test-xion-min-fee-multi-denom-ibc \
	test-xion-min-fee-zero \
	test-xion-min-fee-coverage-all \
	test-xion-min-fee-critical-all

test-integration-dkim-module: compile-integration-tests
	$(MAKE) run-integration-test TEST_NAME=TestDKIMModule
	
test-integration-zkemail-abstract-account: compile-integration-tests
	$(MAKE) run-integration-test TEST_NAME=TestZKEmailAuthenticator

# Simulation tests
test-sim-import-export: runsim
	@echo "Running application import/export simulation. This may take several minutes..."
	@$(BINDIR)/runsim -Jobs=4 -SimAppPkg=$(SIMAPP) -ExitOnFail 50 5 TestAppImportExport

test-sim-multi-seed-short: runsim
	@echo "Running short multi-seed application simulation. This may take awhile!"
	@$(BINDIR)/runsim -Jobs=4 -SimAppPkg=$(SIMAPP) -ExitOnFail 50 5 TestFullAppSimulation

test-sim-deterministic: runsim
	@echo "Running short multi-seed application simulation. This may take awhile!"
	@$(BINDIR)/runsim -Jobs=4 -SimAppPkg=$(SIMAPP) -ExitOnFail 1 1 TestAppStateDeterminism

.PHONY: test test-all test-unit test-race benchmark \
        test-run test-e2e-all \
        test-aa-all test-app-all test-dkim-all test-jwk-all test-xion-all test-indexer-all \
        test-aa-basic test-aa-client-event test-aa-jwt-cli \
        test-aa-multi-auth test-aa-panic \
        test-aa-single-migration test-aa-webauthn \
        test-app-governance test-app-ibc-timeout test-app-ibc-transfer \
        test-app-mint-inflation-high-fees test-app-mint-inflation-low-fees \
        test-app-mint-inflation-no-fees test-app-mint-no-inflation-no-fees \
        test-app-send-platform-fee test-app-simulate test-app-token-factory \
        test-app-treasury-contract test-app-treasury-grants test-app-treasury-multi \
        test-app-update-treasury-configs test-app-update-treasury-configs-aa \
        test-app-update-treasury-params test-app-upgrade-ibc test-app-upgrade-network \
        test-dkim-governance test-dkim-key-revocation test-dkim-module \
        test-dkim-zk-email test-dkim-zk-proof \
        test-indexer-authz-create test-indexer-authz-multiple test-indexer-authz-revoke \
        test-indexer-feegrant-create test-indexer-feegrant-multiple \
        test-indexer-feegrant-periodic test-indexer-feegrant-revoke \
        test-jwk-algorithm-confusion test-jwk-audience-mismatch test-jwk-expired-token \
        test-jwk-invalid-signature test-jwk-jwt-aa test-jwk-key-rotation \
        test-jwk-malformed-tokens test-jwk-missing-claims test-jwk-multiple-audiences \
        test-jwk-transaction-hash \
        test-xion-genesis-export-import test-xion-indexer-authz test-xion-indexer-feegrant \
        test-xion-indexer-non-consensus-critical \
        test-xion-min-fee-bypass test-xion-min-fee-default test-xion-min-fee-multi-denom \
        test-xion-min-fee-multi-denom-ibc test-xion-min-fee-zero test-xion-platform-fee \
        test-xion-platform-fee-bypass test-xion-platform-min-codec-bug \
        test-xion-platform-min-direct \
        test-xion-min-fee-multi-denom-advanced test-xion-min-fee-extreme-values \
        test-xion-min-fee-concurrent-transactions test-xion-min-fee-sequence-handling \
        test-xion-platform-minimum-with-fees test-xion-platform-minimum-codec \
        test-xion-platform-minimum-bypass test-xion-min-fee-error-messages \
        test-xion-min-fee-insufficient-balance test-xion-min-fee-edge-cases \
        test-xion-min-fee-mempool test-xion-min-fee-coverage-all \
        test-xion-min-fee-gas-cap-boundaries test-xion-min-fee-gas-cap-with-fees \
        test-xion-min-fee-gas-cap-multiple-messages test-xion-min-fee-with-feegrant \
        test-xion-min-fee-feegrant-allowance-types test-xion-min-fee-feegrant-expiration \
        test-xion-min-fee-multiple-feegrants test-xion-min-fee-multi-message-mixed-types \
        test-xion-min-fee-multi-message-same-type test-xion-min-fee-multi-message-gas-accounting \
        test-xion-min-fee-multi-message-with-feegrant test-xion-min-fee-multi-message-error-paths \
        test-xion-min-fee-multi-message-sequential test-xion-min-fee-bypass-message-types \
        test-xion-min-fee-critical-all test-xion-min-fee-all \
        test-sim-import-export test-sim-multi-seed-short test-sim-deterministic

# Help targets for test module
help-test-brief:
	@echo "  test                       Run unit tests"

help-test:
	@echo "Test targets:"
	@echo "  test                       Run unit tests"
	@echo "  test-unit                  Run unit tests"
	@echo "  test-race                  Run tests with race detection"
	@echo "  test-e2e                   Run all e2e tests (deprecated - use test-e2e-all)"
	@echo "  test-e2e-all               Run all e2e tests"
	@echo "  test-run                   Run specific e2e test"
	@echo ""
	@echo "E2E Tests by Module:"
	@echo "  test-e2e-abstract-account  Run all Abstract Account module tests (deprecated - use test-aa-all)"
	@echo "  test-e2e-app               Run all App module tests (deprecated - use test-app-all)"
	@echo "  test-e2e-dkim              Run all DKIM module tests (deprecated - use test-dkim-all)"
	@echo "  test-e2e-indexer           Run all Indexer module tests (deprecated - use test-indexer-all)"
	@echo "  test-e2e-jwk               Run all JWK module tests (deprecated - use test-jwk-all)"
	@echo "  test-e2e-xion              Run all Xion module tests (deprecated - use test-xion-all)"
	@echo ""
	@echo "E2E Module Test Suites (Recommended):"
	@echo "  test-aa-all                Run all Abstract Account module tests"
	@echo "  test-app-all               Run all App module tests"
	@echo "  test-dkim-all              Run all DKIM module tests"
	@echo "  test-indexer-all           Run all Indexer module tests"
	@echo "  test-jwk-all               Run all JWK module tests"
	@echo "  test-xion-all              Run all Xion module tests"
	@echo ""
	@echo "  Abstract Account Module Individual Tests:"
	@echo "    test-aa-basic                       Test Xion abstract account"
	@echo "    test-aa-client-event                Test client events"
	@echo "    test-aa-jwt-cli                     Test JWT abstract account CLI"
	@echo "    test-aa-multi-auth                  Test multiple authenticators"
	@echo "    test-aa-panic                       Test panic handling"
	@echo "    test-aa-single-migration            Test single account migration"
	@echo "    test-aa-webauthn                    Test WebAuthn abstract account"
	@echo ""
	@echo "  App Module Individual Tests:"
	@echo "    test-app-governance                 Test governance proposal"
	@echo "    test-app-ibc-timeout                Test IBC timeout handling"
	@echo "    test-app-ibc-transfer               Test IBC token transfer"
	@echo "    test-app-mint-inflation-high-fees   Test mint module with inflation and high fees"
	@echo "    test-app-mint-inflation-low-fees    Test mint module with inflation and low fees"
	@echo "    test-app-mint-inflation-no-fees     Test mint module with inflation and no fees"
	@echo "    test-app-mint-no-inflation-no-fees  Test mint module with no inflation and no fees"
	@echo "    test-app-send-platform-fee          Test platform fee sending"
	@echo "    test-app-simulate                   Test simulation"
	@echo "    test-app-token-factory              Test token factory"
	@echo "    test-app-treasury-contract          Test treasury contract"
	@echo "    test-app-treasury-grants            Test treasury grants"
	@echo "    test-app-treasury-multi             Test treasury multi-signature"
	@echo "    test-app-update-treasury-configs    Test treasury config updates"
	@echo "    test-app-update-treasury-configs-aa Test treasury config updates with AA"
	@echo "    test-app-update-treasury-params     Test treasury parameter updates"
	@echo "    test-app-upgrade-ibc                Test IBC upgrade"
	@echo "    test-app-upgrade-network            Test network upgrade"
	@echo ""
	@echo "  DKIM Module Individual Tests:"
	@echo "    test-dkim-governance                Test governance-only key registration"
	@echo "    test-dkim-key-revocation            Test key revocation"
	@echo "    test-dkim-module                    Test DKIM module functionality"
	@echo "    test-dkim-zk-email                  Test ZK email authenticator"
	@echo "    test-dkim-zk-proof                  Test ZK proof validation"
	@echo ""
	@echo "  Indexer Module Individual Tests:"
	@echo "    test-indexer-authz-create           Test authz grant indexing"
	@echo "    test-indexer-authz-multiple         Test multiple authz grants"
	@echo "    test-indexer-authz-revoke           Test authz grant revocation"
	@echo "    test-indexer-feegrant-create        Test feegrant allowance indexing"
	@echo "    test-indexer-feegrant-multiple      Test multiple feegrant allowances"
	@echo "    test-indexer-feegrant-periodic      Test periodic allowance types"
	@echo "    test-indexer-feegrant-revoke        Test feegrant allowance revocation"
	@echo ""
	@echo "  JWK Module Individual Tests:"
	@echo "    test-jwk-algorithm-confusion        Test algorithm confusion prevention"
	@echo "    test-jwk-audience-mismatch          Test audience mismatch validation"
	@echo "    test-jwk-expired-token              Test expired token handling"
	@echo "    test-jwk-invalid-signature          Test invalid JWT signature rejection"
	@echo "    test-jwk-jwt-aa                     Test JWT abstract account"
	@echo "    test-jwk-key-rotation               Test key rotation functionality"
	@echo "    test-jwk-malformed-tokens           Test malformed token handling"
	@echo "    test-jwk-missing-claims             Test required claims validation"
	@echo "    test-jwk-multiple-audiences         Test multiple audiences validation"
	@echo "    test-jwk-transaction-hash           Test replay attack prevention"
	@echo ""
	@echo "  Xion Module Individual Tests:"
	@echo "    test-xion-genesis-export-import         Test genesis export and import cycle"
	@echo "    test-xion-indexer-authz                 Test authz grant indexing (includes robustness tests)"
	@echo "    test-xion-indexer-feegrant              Test fee grant indexing (includes robustness tests)"
	@echo "    test-xion-indexer-non-consensus-critical Test indexer non-consensus-critical operation"
	@echo "    test-xion-min-fee-bypass            Test minimum fee bypass prevention"
	@echo "    test-xion-min-fee-default           Test minimum fee default"
	@echo "    test-xion-min-fee-multi-denom       Test multi-denom min global fee"
	@echo "    test-xion-min-fee-multi-denom-ibc   Test multi-denom min global fee IBC"
	@echo "    test-xion-min-fee-zero              Test minimum fee zero"
	@echo "    test-xion-platform-fee              Test platform fee collection"
	@echo "    test-xion-platform-fee-bypass       Test platform fee bypass prevention"
	@echo "    test-xion-platform-min-codec-bug    Test platform minimum codec bug fix"
	@echo "    test-xion-platform-min-direct       Test platform minimum direct transaction"
	@echo ""
	@echo "Simulation tests:"
	@echo "  test-sim                   Run simulation tests"
	@echo "  test-sim-import-export     Run simulation import/export tests"
	@echo "  test-sim-multi-seed-short  Run multi-seed simulation tests"
	@echo "  test-sim-deterministic     Run deterministic simulation tests"
	@echo ""

