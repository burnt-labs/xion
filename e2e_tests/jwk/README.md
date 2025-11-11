# JWK E2E Tests

This directory contains end-to-end tests for XION's JSON Web Key (JWK) module.

## Overview

The JWK module manages public keys for JWT (JSON Web Token) authentication, enabling Web2-style authentication for blockchain accounts. This is a core component of Abstract Account authentication.

## Test Files

### jwt_aa_test.go
Basic JWT authentication:
- **TestJWTAbstractAccount** - Full JWT authentication flow with Abstract Accounts

### advanced_test.go
Security and edge case testing:
- **TestJWKExpiredToken** - Expired JWT token rejection
- **TestJWKAudienceMismatch** - Cross-audience token isolation
- **TestJWKKeyRotation** - Key update mechanism
- **TestJWKMultipleAudiences** - Multi-tenant independence

### Priority 1 Security Tests

#### algorithm_confusion_security_test.go
- **TestJWKAlgorithmConfusionSecurity** - Prevents [CVE-2015-9235](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2015-9235) algorithm confusion attacks
  - Blocks RS256 → HS256 algorithm switching
  - Prevents asymmetric key used as symmetric key
  - Tests "none" algorithm rejection
  - Validates algorithm header immutability

#### key_rotation_security_test.go
- **TestJWKKeyRotationSecurity** - Ensures secure key rotation without service disruption
  - Validates old tokens rejected after key update
  - Tests grace period for key transitions
  - Prevents unauthorized key modifications
  - Ensures atomic key updates

#### token_expiration_security_test.go
- **TestJWKTokenExpirationSecurity** - Enforces strict token expiration
  - Rejects expired tokens immediately
  - Validates exp claim presence and format
  - Prevents exp claim manipulation
  - Tests clock skew tolerance

#### token_replay_security_test.go
- **TestJWKTokenReplaySecurity** - Prevents token replay attacks
  - Validates nonce uniqueness per transaction
  - Tests transaction hash binding
  - Prevents cross-account token reuse
  - Ensures one-time token usage

#### signature_validation_security_test.go
- **TestJWKSignatureValidationSecurity** - Comprehensive signature verification
  - Rejects tampered JWT payloads
  - Validates signature against registered public key
  - Tests malformed signature rejection
  - Prevents signature stripping attacks

### test_helpers.go
Shared test utilities:
- JWT token generation
- JWK audience setup
- Key loading helpers

## Test Data

### keys/
- `jwtRS256.key` - RSA private key for test JWT signing
- `jwtRS256.key.pub` - RSA public key
- `public.json` - JWK-formatted public key

## Running Tests

```bash
# Run all JWK tests
make test-e2e-jwk

# Run all Priority 1 security tests
make test-e2e-jwk-security

# Run individual security tests
make test-e2e-jwk-algorithm-confusion
make test-e2e-jwk-key-rotation
make test-e2e-jwk-token-expiration
make test-e2e-jwk-token-replay
make test-e2e-jwk-signature-validation

# Run specific tests directly
cd e2e_tests/jwk && go test -v -run TestJWKExpiredToken
cd e2e_tests/jwk && go test -v -run TestJWKKeyRotation
```

## Key Concepts

### Audiences
An "audience" represents a JWT issuer (e.g., auth0.com, google.com):
- Each audience has a registered public key
- Tokens are validated against audience's public key
- Audiences provide multi-tenancy

### JWT Authentication Flow
1. User obtains JWT from identity provider
2. JWT includes:
   - `aud` (audience) - identifies the issuer
   - `sub` (subject) - user identifier
   - `exp` (expiration) - token validity
   - Transaction hash for replay protection
3. XION validates JWT against registered JWK
4. Transaction executes if valid

### Security Guarantees
- **Token Expiration** - Expired tokens rejected
- **Audience Isolation** - Tokens only valid for their audience
- **Replay Protection** - Each token tied to specific transaction
- **Key Rotation** - Audiences can update public keys

## Test Coverage

### Positive Cases
- ✅ Valid JWT authentication
- ✅ Account registration with JWT
- ✅ Transaction execution with JWT

### Negative Cases (Security)
- ✅ Expired token rejection
- ✅ Wrong audience token rejection
- ✅ Invalid signature rejection
- ✅ Missing claims rejection

### Operational
- ✅ Key rotation without downtime
- ✅ Multiple independent audiences
- ✅ Audience deletion

## Test Dependencies

These tests require:
- RSA key pair for JWT signing
- Abstract Account contracts (from `../aa/contracts/`)
- Funded test accounts

## Common Issues

### Issue: JWT signature validation fails
**Solution**: Ensure public key in JWK module matches private key used for signing

### Issue: "audience not found" error
**Solution**: Register audience with `MsgCreateAudience` before using

### Issue: Token always rejected as expired
**Solution**: Check system clock; ensure `exp` claim is in the future

## Integration

JWK tests integrate with:
- **Abstract Account module** - JWT is an AA authenticator
- **Bank module** - For test transactions
- **Gov module** - For audience management

## Helper Functions

### LoadJWTTestKey()
Loads the test RSA key pair

### CreateValidJWTToken()
Creates a properly formatted JWT with valid claims

### CreateExpiredJWTToken()
Creates an expired JWT for negative testing

### SetupJWKAudience()
Registers a new audience with public key

### UpdateJWKAudience()
Updates audience public key (key rotation)

---

Package: `e2e_jwk`
Shared utilities: `../testlib/`
