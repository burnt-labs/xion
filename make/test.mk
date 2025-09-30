# Test targets and configuration

SIMAPP = ./app
BINDIR ?= $(GOPATH)/bin
TEST_BIN ?= ./integration_tests/integration_tests.test

test: test-unit
test-all: check test-race test-cover

benchmark:
	@go test -mod=readonly -bench=. ./...

test-unit:
	@version=$(version) go test -mod=readonly -tags='ledger test_ledger_mock' ./...

test-race:
	@version=$(version) go test -mod=readonly -race -tags='ledger test_ledger_mock' ./...

compile-integration-tests:
	@cd integration_tests && go test -c -mod=readonly -tags='ledger test_ledger_mock'

test-integration:
	@XION_IMAGE=$(HEIGHLINER_IMAGE) cd ./integration_tests && go test -mod=readonly -tags='ledger test_ledger_mock' ./...

run-integration-test:
	@XION_IMAGE=$(HEIGHLINER_IMAGE) $(TEST_BIN) -test.failfast -test.v -test.run $(TEST_NAME)

test-integration-abstract-account-migration: compile-integration-tests
	$(MAKE) run-integration-test TEST_NAME=TestAbstractAccountMigration

test-integration-jwt-abstract-account: compile-integration-tests
	$(MAKE) run-integration-test TEST_NAME=TestJWTAbstractAccount

test-integration-min-fee: compile-integration-tests
	$(MAKE) run-integration-test TEST_NAME=TestXionMinimumFeeDefault

test-integration-mint-module-inflation-high-fees: compile-integration-tests
	$(MAKE) run-integration-test TEST_NAME=TestMintModuleInflationHighFees

test-integration-mint-module-inflation-low-fees: compile-integration-tests
	$(MAKE) run-integration-test TEST_NAME=TestMintModuleInflationLowFees

test-integration-mint-module-inflation-no-fees: compile-integration-tests
	$(MAKE) run-integration-test TEST_NAME=TestMintModuleInflationNoFees

test-integration-mint-module-no-inflation-no-fees: compile-integration-tests
	$(MAKE) run-integration-test TEST_NAME=TestMintModuleNoInflationNoFees

test-integration-register-jwt-abstract-account: compile-integration-tests
	$(MAKE) run-integration-test TEST_NAME=TestXionAbstractAccountJWTCLI

test-integration-simulate: compile-integration-tests
	$(MAKE) run-integration-test TEST_NAME=TestSimulate

test-integration-single-aa-mig: compile-integration-tests
	$(MAKE) run-integration-test TEST_NAME=TestSingleAbstractAccountMigration

test-integration-treasury-contract: compile-integration-tests
	$(MAKE) run-integration-test TEST_NAME=TestTreasuryContract

test-integration-treasury-multi: compile-integration-tests
	$(MAKE) run-integration-test TEST_NAME=TestTreasuryMulti

test-integration-upgrade-ibc: compile-integration-tests
	$(MAKE) run-integration-test TEST_NAME=TestXionUpgradeIBC

test-integration-upgrade-network: compile-integration-tests
	$(MAKE) run-integration-test TEST_NAME=TestXionUpgradeNetwork

test-integration-web-auth-n-abstract-account: compile-integration-tests
	$(MAKE) run-integration-test TEST_NAME=TestWebAuthNAbstractAccount

test-integration-xion-abstract-account: compile-integration-tests
	$(MAKE) run-integration-test TEST_NAME=TestXionAbstractAccount

test-integration-xion-abstract-account-event: compile-integration-tests
	$(MAKE) run-integration-test TEST_NAME=TestXionClientEvent

test-integration-xion-min-default: compile-integration-tests
	$(MAKE) run-integration-test TEST_NAME=TestXionMinimumFeeDefault

test-integration-xion-min-multi-denom: compile-integration-tests
	$(MAKE) run-integration-test TEST_NAME=TestMultiDenomMinGlobalFee

test-integration-xion-min-multi-denom-ibc: compile-integration-tests
	$(MAKE) run-integration-test TEST_NAME=TestMultiDenomMinGlobalFeeIBC

test-integration-xion-min-zero: compile-integration-tests
	$(MAKE) run-integration-test TEST_NAME=TestXionMinimumFeeZero

test-integration-xion-send-platform-fee: compile-integration-tests
	$(MAKE) run-integration-test TEST_NAME=TestXionSendPlatformFee

test-integration-xion-token-factory: compile-integration-tests
	$(MAKE) run-integration-test TEST_NAME=TestXionTokenFactory

test-integration-xion-treasury-grants: compile-integration-tests
	$(MAKE) run-integration-test TEST_NAME=TestTreasuryContract

test-integration-xion-update-treasury-configs: compile-integration-tests
	$(MAKE) run-integration-test TEST_NAME=TestUpdateTreasuryConfigsWithLocalAndURL configUrl="$(configUrl)"

test-integration-xion-update-treasury-configs-aa: compile-integration-tests
	$(MAKE) run-integration-test TEST_NAME=TestUpdateTreasuryConfigsWithAALocalAndURL configUrl="$(configUrl)"

test-integration-xion-update-treasury-params: compile-integration-tests
	$(MAKE) run-integration-test TEST_NAME=TestUpdateTreasuryContractParams

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

.PHONY: test test-all test-version test-unit test-race benchmark \
        compile-integration-tests test-integration run-integration-test \
        test-integration-abstract-account-migration test-integration-jwt-abstract-account \
        test-integration-min-fee test-integration-mint-module-inflation-high-fees \
        test-integration-mint-module-inflation-low-fees test-integration-mint-module-inflation-no-fees \
        test-integration-mint-module-no-inflation-no-fees test-integration-register-jwt-abstract-account \
        test-integration-simulate test-integration-single-aa-mig test-integration-treasury-contract \
        test-integration-treasury-multi test-integration-upgrade-ibc test-integration-upgrade-network \
        test-integration-web-auth-n-abstract-account test-integration-xion-abstract-account \
        test-integration-xion-abstract-account-event test-integration-xion-min-default \
        test-integration-xion-min-multi-denom test-integration-xion-min-multi-denom-ibc \
        test-integration-xion-min-zero test-integration-xion-send-platform-fee \
        test-integration-xion-token-factory test-integration-xion-treasury-grants \
        test-integration-xion-update-treasury-configs test-integration-xion-update-treasury-configs-aa \
        test-integration-xion-update-treasury-params \
        test-sim-import-export test-sim-multi-seed-short test-sim-deterministic

# Help targets for test module
help-test-brief:
	@echo "  test                       Run unit tests"

help-test:
	@echo "Test targets:"
	@echo "  test                       Run unit tests"
	@echo "  test-unit                  Run unit tests"
	@echo "  test-race                  Run tests with race detection"
	@echo "  test-integration           Run integration tests"
	@echo "  compile-integration-tests  Compile integration test binary"
	@echo "  run-integration-test       Run specific integration test"
	@echo "  test-sim                   Run simulation tests"
	@echo "  test-sim-import-export     Run simulation import/export tests"
	@echo "  test-sim-multi-seed-short  Run multi-seed simulation tests"
	@echo "  test-sim-deterministic     Run deterministic simulation tests"
	@echo ""

.PHONY: test test-unit test-race test-integration compile-integration-tests run-integration-test \
        test-sim test-sim-nondeterminism test-sim-custom-genesis-fast test-sim-import-export \
        test-sim-after-import test-sim-custom-genesis-multi-seed test-sim-multi-seed-long \
        test-sim-multi-seed-short test-integration-min-fee test-integration-mint-module-inflation-high-fees \
        test-integration-mint-module-inflation-low-fees test-integration-mint-module-inflation-no-fees \
        test-integration-mint-module-no-inflation-no-fees test-integration-register-jwt-abstract-account \
        test-integration-simulate test-integration-single-aa-mig test-integration-treasury-contract \
        test-integration-treasury-multi test-integration-upgrade-ibc test-integration-upgrade-network \
        test-integration-web-auth-n-abstract-account test-integration-xion-abstract-account \
        test-integration-xion-abstract-account-event test-integration-xion-min-default \
        test-integration-xion-min-multi-denom test-integration-xion-min-multi-denom-ibc \
        test-integration-xion-min-zero test-integration-xion-send-platform-fee \
        test-integration-xion-token-factory test-integration-xion-treasury-grants \
        test-integration-xion-update-treasury-configs test-integration-xion-update-treasury-configs-aa \
        test-integration-xion-update-treasury-params \
        test-sim-import-export test-sim-multi-seed-short test-sim-deterministic \
        help-test help-test-brief
