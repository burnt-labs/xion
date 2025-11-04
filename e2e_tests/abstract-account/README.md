# Abstract Account E2E Tests

This directory contains end-to-end tests for XION's Abstract Account module.

## Overview

Abstract Accounts (AA) are smart contract-based accounts that support pluggable authenticators, allowing flexible authentication methods beyond traditional cryptographic signatures.

## Test Files

### abstract_account_test.go
Tests core Abstract Account functionality:
- **TestXionAbstractAccountJWTCLI** - JWT-based account registration and authentication via CLI
- **TestXionAbstractAccount** - Basic AA operations
- **TestXionAbstractAccountPanic** - Error handling and panic recovery
- **TestXionClientEvent** - Client event emission

### account_migration_test.go
Tests account migration scenarios:
- **TestAbstractAccountMigration** - Migrating accounts between contract versions
- **TestSingleAbstractAccountMigration** - Single account migration workflow

### Priority 1 Security Tests

#### multi_auth_security_test.go
- **TestMultipleAuthenticatorsSecurity** - Prevents account lockout with multiple authenticators
  - Validates two JWT authenticators work independently
  - Tests authenticator addition without disruption
  - Ensures authenticator independence (failure isolation)
  - Prevents complete lockout scenarios

## Test Data

### contracts/
- `account_updatable-aarch64.wasm` - Main updatable account contract
- `account_updatable-aarch64-previous.wasm` - Previous version for migration tests
- `account-wasm-updatable-event-aarch64.wasm` - Event-emitting account variant
- `xion-account.wasm` - Standard account contract

## Running Tests

```bash
# Run all AA tests
make test-e2e-aa

# Run all Priority 1 security tests
make test-e2e-aa-security

# Run individual security tests
make test-e2e-aa-multi-auth

# Run specific test
cd e2e_tests/aa && go test -v -run TestAbstractAccountMigration
cd e2e_tests/aa && go test -v -run TestMultipleAuthenticatorsSecurity
```

## Key Concepts

### Authenticators
AA supports multiple authentication methods:
- **JWT** - JSON Web Token authentication
- **WebAuthn** - FIDO2/Passkeys
- **Secp256k1** - Traditional cryptographic signatures
- **Secp256r1** - Alternative signature scheme
- **Custom** - Smart contract-defined authenticators

### Registration Flow
1. Deploy AA contract via `xion register`
2. Set up authenticators (JWT audience, WebAuthn credential, etc.)
3. Execute transactions using selected authenticator

### Migration
Accounts can be migrated to new contract versions while preserving:
- Account address
- Authenticator configuration
- Account state

## Test Dependencies

These tests require:
- JWT RSA keys (from `../jwk/keys/`)
- Abstract Account WASM contracts
- Funded test accounts

## Common Issues

### Issue: Contract deployment fails
**Solution**: Ensure WASM contracts are in `contracts/` directory

### Issue: JWT authentication fails
**Solution**: Check that JWK module has registered the audience with matching public key

### Issue: Migration fails
**Solution**: Verify contract versions are compatible and migration path is defined

## Integration

AA tests integrate with:
- **JWK module** - For JWT authentication
- **XION module** - For platform fees
- **Bank module** - For fund transfers

---

Package: `e2e_aa`
Shared utilities: `../testlib/`
