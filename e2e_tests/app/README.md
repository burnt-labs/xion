# Application-Level E2E Tests

This directory contains end-to-end tests for application-level functionality that spans multiple modules or tests core blockchain operations.

## Overview

These tests verify cross-cutting concerns, infrastructure, and core blockchain functionality that doesn't belong to a specific custom module.

## Test Files

### send_test.go
Platform fee testing:
- **TestXionSendPlatformFee** - Platform fee collection on transactions

### simulate_test.go
Transaction simulation:
- **TestSimulate** - Gas estimation and simulation accuracy

### token_factory_test.go
Token factory operations:
- **TestXionTokenFactory** - Creating and managing custom tokens

### treasury_test.go
Treasury contract testing:
- **TestTreasuryContract** - Basic treasury operations
- **TestTreasuryMulti** - Multi-signature treasury

### update_treasury_configs_test.go
Treasury configuration updates:
- **TestUpdateTreasuryConfigsWithLocalAndURL** - Config updates from local and remote sources
- **TestUpdateTreasuryConfigsWithAALocalAndURL** - Config updates with Abstract Accounts

### update_treasury_params_test.go
Treasury parameter management:
- **TestUpdateTreasuryContractParams** - Parameter modification

### upgrade_test.go
Chain upgrade testing:
- **TestXionUpgradeNetwork** - Chain software upgrades

### upgrade_ibc_test.go
IBC upgrade testing:
- **TestXionUpgradeIBC** - IBC protocol upgrades
- **TestDungeonTransferBlock** - Cross-chain transfers

### Priority 1 Security Tests

#### ibc_security_test.go
- **TestIBCTokenTransferSecurity** - Comprehensive IBC cross-chain security
  - Validates proper IBC denom tracing
  - Tests escrow account balance verification
  - Ensures packet ordering enforcement
  - Prevents source chain spoofing
  - Tests double-spend prevention across chains

#### ibc_timeout_test.go
- **TestIBCTimeoutHandling** - IBC timeout and refund security
  - Validates timeout height/timestamp refunds
  - Tests escrow release on timeout
  - Ensures network partition recovery
  - Prevents fund loss during timeouts
  - Tests valid transfers complete without timeout

#### governance_security_test.go
- **TestGovernanceProposalSecurity** - Governance security mechanisms
  - Validates minimum deposit requirements
  - Tests voting period enforcement
  - Ensures quorum and threshold requirements
  - Prevents proposal spam attacks
  - Tests deposit refund mechanism
  - Validates authority-only operations
  - Ensures parameter change validation

## Test Data

### contracts/
- `treasury-aarch64.wasm` - Treasury smart contract
- `tokenfactory_core.wasm` - Token factory contract
- `user_map.wasm` - User mapping helper contract

### unsigned_msgs/
- `bank_send_unsigned.json` - Unsigned bank send message template
- `config.json` - Configuration template
- `plain_config.json` - Plain configuration template

## Running Tests

```bash
# Run all app tests
make test-e2e-app

# Run all Priority 1 security tests
make test-e2e-app-security

# Run individual security tests
make test-e2e-app-ibc-security
make test-e2e-app-ibc-timeout
make test-e2e-app-governance-security

# Run specific tests directly
cd e2e_tests/app && go test -v -run TestTreasuryContract
cd e2e_tests/app && go test -v -run TestXionTokenFactory
cd e2e_tests/app && go test -v -run TestIBCTokenTransferSecurity
```

## Key Concepts

### Platform Fees
XION supports governance-controlled platform fees:
- Percentage-based fees on transactions
- Minimum fee thresholds
- Fee distribution to treasury

### Token Factory
Allows creating custom tokens without deploying contracts:
- Create denominations
- Mint/burn tokens
- Transfer admin rights
- Set metadata

### Treasury
Multi-signature fund management:
- Configurable signers
- Proposal-based operations
- User mapping for access control

### Chain Upgrades
Testing blockchain version updates:
- Software upgrade proposals
- State migration
- Binary compatibility
- IBC protocol upgrades

## Test Dependencies

These tests require:
- Treasury and token factory WASM contracts
- Funded test accounts
- Governance permissions (for some tests)

## Common Issues

### Issue: Platform fee test fails
**Solution**: Ensure governance has set platform fee parameters

### Issue: Token factory operations fail
**Solution**: Check that token factory module is enabled in genesis

### Issue: Treasury deployment fails
**Solution**: Verify user_map contract is deployed first

### Issue: Upgrade test hangs
**Solution**: Ensure upgrade height is reachable and upgrade handler is registered

## Integration

App tests integrate with:
- **Bank module** - For transfers
- **Gov module** - For proposals
- **IBC module** - For cross-chain operations
- **XION module** - For platform fees

---

Package: `e2e_app`
Shared utilities: `../testlib/`
