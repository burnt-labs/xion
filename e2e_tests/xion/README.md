# XION Core E2E Tests

This directory contains end-to-end tests for XION core module functionality.

## Overview

The XION core module provides platform-level features including platform fees, multi-denomination support, and WebAuthn integration.

## Test Files

### minimum_fee_test.go
Platform fee testing:
- **TestXionMinimumFeeDefault** - Default platform fee behavior
- **TestXionMinimumFeeZero** - Zero platform fee scenario
- **TestMultiDenomMinGlobalFee** - Multi-denomination fee support
- **TestMultiDenomMinGlobalFeeIBC** - IBC token fees

### webauthn_test.go
WebAuthn/Passkeys testing:
- **TestWebAuthNAbstractAccount** - WebAuthn authentication with Abstract Accounts
- FIDO2/Passkey integration
- Biometric authentication

### Priority 1 Security Tests

#### platform_fee_bypass_security_test.go
- **TestPlatformFeeBypassSecurity** - Prevents platform fee evasion attacks
  - Validates all transaction types pay fees
  - Tests multi-msg transaction fee calculation
  - Prevents fee field manipulation
  - Ensures treasury receives correct amounts

#### min_fee_bypass_test.go
- **TestMinimumFeeBypassSecurity** - Enforces minimum fee requirements
  - Rejects transactions below minimum threshold
  - Tests fee denom validation
  - Prevents zero-fee transactions
  - Validates fee-grant abuse prevention

#### platform_fee_security_test.go
- **TestPlatformFeeCalculationSecurity** - Ensures accurate platform fee calculations
  - Validates percentage-based fee math
  - Tests rounding behavior (prevent truncation exploits)
  - Ensures overflow protection
  - Validates multi-denom conversion rates

#### fee_grant_exploit_test.go
- **TestFeeGrantExploitSecurity** - Prevents fee grant abuse
  - Validates allowance limits enforced
  - Tests expiration time enforcement
  - Prevents grant reuse after revocation
  - Ensures per-message allowance tracking

#### treasury_manipulation_test.go
- **TestTreasuryManipulationSecurity** - Protects treasury fund security
  - Validates only governance can modify treasury
  - Tests unauthorized withdrawal prevention
  - Prevents treasury address manipulation
  - Ensures fee routing integrity

## Running Tests

```bash
# Run all XION core tests
make test-e2e-xion

# Run all Priority 1 security tests
make test-e2e-xion-security

# Run individual security tests
make test-e2e-xion-platform-fee-bypass
make test-e2e-xion-min-fee-bypass
make test-e2e-xion-platform-fee-calculation
make test-e2e-xion-fee-grant-exploit
make test-e2e-xion-treasury-manipulation

# Run specific tests directly
cd e2e_tests/xion && go test -v -run TestXionMinimumFeeDefault
cd e2e_tests/xion && go test -v -run TestWebAuthNAbstractAccount
```

## Key Concepts

### Platform Fees
XION supports governance-controlled platform fees:
- **Minimum fee threshold** - Transactions must pay at least this amount
- **Percentage-based fees** - Fee as percentage of transaction amount
- **Multi-denom support** - Fees in various tokens (XION, IBC tokens)
- **Fee distribution** - Fees sent to treasury

### Multi-Denomination Fees
Users can pay fees in different tokens:
- Native XION token (uxion)
- IBC transferred tokens (e.g., ATOM, USDC)
- Custom tokens from token factory

Conversion rates set by governance.

### WebAuthn
FIDO2/Passkey authentication for blockchain:
- **Biometric auth** - Fingerprint, Face ID
- **Hardware keys** - YubiKey, etc.
- **Platform authenticators** - Device secure enclave
- **Cross-device** - Sync across devices

## Test Coverage

### Platform Fees
- ✅ Default fee collection
- ✅ Zero fee scenario
- ✅ Fee enforcement
- ✅ Multi-denomination fees
- ✅ IBC token fees
- ✅ Fee distribution to treasury

### WebAuthn
- ✅ Account registration with WebAuthn
- ✅ Transaction signing with passkey
- ✅ Credential management
- ✅ Challenge/response flow

## Test Dependencies

These tests require:
- Governance permissions (for fee parameter updates)
- Abstract Account contracts (for WebAuthn)
- IBC setup (for multi-denom IBC tests)
- WebAuthn credential generation utilities

## Common Issues

### Issue: Platform fee not collected
**Solution**: Ensure platform fee parameters are set via governance

### Issue: Multi-denom fee rejected
**Solution**: Check that alternative denomination is registered with conversion rate

### Issue: WebAuthn verification fails
**Solution**: Ensure credential public key matches the signature

### Issue: IBC token not accepted as fee
**Solution**: Register IBC token with fee parameters

## Integration

XION core tests integrate with:
- **Bank module** - For fee collection
- **IBC module** - For cross-chain token fees
- **Abstract Account module** - For WebAuthn auth
- **Gov module** - For fee parameter management

## WebAuthn Details

### Registration Flow
1. Generate WebAuthn credential (attestation)
2. Register Abstract Account with credential public key
3. Store credential ID and public key on-chain

### Authentication Flow
1. Build transaction
2. Sign transaction hash with WebAuthn credential
3. Submit transaction with WebAuthn signature
4. Chain verifies signature against registered public key

### Supported Algorithms
- ES256 (secp256r1)
- RS256 (RSA)

## Platform Fee Configuration

Governance can set:
- `minimum_fee_percentage` - Percentage of transaction value
- `minimum_fee_amount` - Absolute minimum fee
- `accepted_denoms` - Which tokens can pay fees
- `conversion_rates` - Exchange rates for alternative denoms

---

Package: `e2e_xion`
Shared utilities: `../testlib/`
