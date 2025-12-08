# DKIM E2E Tests

This directory contains end-to-end tests for XION's DKIM (DomainKeys Identified Mail) module.

## Overview

The DKIM module stores DKIM public keys and their Poseidon hashes on-chain, enabling ZK-Email authentication for Abstract Accounts. It mimics traditional DKIM functionality by securely storing information found in email headers, allowing verification of email authenticity on the blockchain. The Poseidon hash computation provides ZK-circuit compatibility for efficient on-chain verification.

## Test Files

### dkim_test.go

Tests core DKIM module functionality:

- **TestDKIMModule** - Basic DKIM query operations
  - Query single DKIM record by domain and selector
  - Query all DKIM records for a domain
  - Query by domain + Poseidon hash pair
  - Generate DKIM records from DNS lookup (`gdkim`)

- **TestDKIMGovernance** - Governance-controlled key management
  - Generate DKIM record from DNS
  - Add DKIM record via governance proposal
  - Verify record was added
  - Remove DKIM record via governance proposal
  - Verify record was removed

- **TestDKIMKeyRevocation** - Private key revocation flow
  - Generate RSA-2048 key pair
  - Compute Poseidon hash for public key
  - Register key via governance proposal
  - Revoke key using private key proof (`rdkim`)
  - Verify key was revoked

### zk_email_test.go

Tests ZK-Email authentication integration with Abstract Accounts:

- **TestZKEmailAuthenticator** - End-to-end ZK-Email authentication
  - Deploy Abstract Account contract
  - Add ZK-Email authenticator with email salt and allowed hosts
  - Query contract to verify authenticator was created
  - Execute bank send using ZK proof signature
  - Verify recipient received funds

## Test Data

### Genesis Records

The test chain is initialized with predefined DKIM records:

| Domain   | Selector   | Poseidon Hash |
|----------|------------|---------------|
| x.com    | dkim202406 | poseidon_hash_1 |
| xion.com | dkim202407 | poseidon_hash_2 |
| x.com    | google     | poseidon_hash_3 |

### External Records

- `account.netflix.com` / `kk6c473czcop4fqv6yhfgiqupmfz3cm2` - Netflix DKIM (fetched via DNS)

### Test Keys and Proofs

Located in `testdata/keys/`:

- `zk-auth.json` - ZK proof for authenticator registration
- `zk-transaction.json` - ZK proof for transaction signing

### Contracts

Located in `testdata/contracts/`:

- `xion_account.wasm` - Abstract Account contract with ZK-Email support

## Running Tests

```bash
# Run all DKIM module tests
make test-dkim-module

# Run governance tests
make test-dkim-governance

# Run key revocation tests
make test-dkim-key-revocation

# Run ZK-Email authenticator test
make test-zk-email

# Run specific test directly
cd e2e_tests/dkim && go test -v -run TestDKIMModule
cd e2e_tests/dkim && go test -v -run TestDKIMGovernance
cd e2e_tests/dkim && go test -v -run TestDKIMKeyRevocation
cd e2e_tests/dkim && go test -v -run TestZKEmailAuthenticator
```

## Key Concepts

### DKIM Records

Each DKIM record consists of:

- **Domain** - Email domain (e.g., `x.com`)
- **Selector** - Unique identifier for the public key in DNS records
- **PubKey** - Base64-encoded RSA public key (DER format)
- **PoseidonHash** - Hash for ZK circuit compatibility

### Poseidon Hash

The module computes Poseidon hashes of public keys, providing a secure and efficient way to verify email signatures within ZK circuits. Use `dkimTypes.ComputePoseidonHash(pubKey)` to generate hashes programmatically.

### ZK-Email Authentication

ZK-Email enables email-based authentication for Abstract Accounts:

1. User proves ownership of an email address via ZK proof
2. Proof validates against on-chain DKIM public keys
3. Abstract Account executes transactions based on valid proofs

### Query Operations

```bash
# Query single record by selector and domain
xiond query dkim dkim-pubkey <domain> <selector>

# Query all records for a domain
xiond query dkim qdkims --domain <domain>

# Query by domain and Poseidon hash
xiond query dkim qdkims --domain <domain> --hash <poseidon_hash>

# Generate DKIM record from DNS TXT lookup
xiond query dkim gdkim <domain> <selector>
```

### Governance Operations

Adding and removing DKIM keys requires governance proposals:

- **MsgAddDkimPubKeys** - Add one or more DKIM public keys
- **MsgRemoveDkimPubKey** - Remove a specific DKIM record by selector and domain

### Key Revocation

Domain owners can revoke their DKIM keys by proving ownership of the private key:

```bash
xiond tx dkim rdkim <domain> <private_key_base64> --chain-id <chain-id>
```

This enables immediate revocation without governance when a private key is compromised.

## Test Flow

### TestDKIMModule

1. Query single DKIM record by domain/selector
2. Query all records for a domain
3. Query with Poseidon hash filter
4. Generate DKIM record from DNS

### TestDKIMGovernance

1. Generate DKIM record from DNS via `gdkim`
2. Submit governance proposal to add record
3. Verify record exists on-chain
4. Submit governance proposal to remove record
5. Verify record no longer exists

### TestDKIMKeyRevocation

1. Generate RSA-2048 key pair locally
2. Compute Poseidon hash for public key
3. Submit governance proposal to add key
4. Verify key exists on-chain
5. Revoke key via `rdkim` with private key proof
6. Verify key no longer exists

### TestZKEmailAuthenticator

1. Fund deployer account
2. Store Abstract Account contract
3. Register Abstract Account with Secp256K1 authenticator
4. Add ZK-Email authenticator with email salt and allowed hosts
5. Query contract to verify ZK-Email authenticator exists
6. Build bank send transaction
7. Sign with pre-generated ZK proof
8. Broadcast transaction
9. Verify recipient received funds

## CLI Commands

| Command | Type | Description |
|---------|------|-------------|
| `query dkim dkim-pubkey` | Query | Fetch single DKIM record |
| `query dkim qdkims` | Query | Fetch multiple DKIM records with filters |
| `query dkim gdkim` | Query | Generate DKIM record from DNS lookup |
| `tx dkim rdkim` | Transaction | Revoke DKIM key with private key proof |

## gRPC Endpoints

### Query Service

- `Params` - Retrieve module parameters
- `DkimPubKey` - Fetch DKIM public key and Poseidon hash for selector/domain

### Msg Service

- `UpdateParams` - Governance-controlled parameter updates
- `AddDkimPubKey` - Add DKIM public keys (governance only)
- `RemoveDkimPubKey` - Remove DKIM public key (governance only)
- `RevokeDkimPubKey` - Revoke key with private key proof

## Test Dependencies

These tests require:

- Running XION chain with DKIM module enabled
- Governance module for proposal submission
- Network access for DNS DKIM lookups (`gdkim` command)
- Pre-funded test accounts (10B uxion for fees and deposits)
- Abstract Account WASM contracts (for ZK-Email tests)
- Pre-generated ZK proofs in `testdata/keys/`

## Common Issues

### Issue: DKIM record not found

**Solution**: Verify the domain and selector are correct; check genesis configuration includes the record

### Issue: Governance proposal fails

**Solution**: Ensure proposer has sufficient funds for deposit (500M uxion minimum in tests)

### Issue: DNS lookup fails for `gdkim`

**Solution**: Verify the domain has a valid DKIM TXT record at `<selector>._domainkey.<domain>`

### Issue: Key revocation fails

**Solution**: Ensure the private key matches the registered public key; key must be in base64-encoded PKCS1 format

### Issue: Poseidon hash mismatch

**Solution**: Verify public key is in correct format (base64, no PEM headers, no newlines) before hashing

### Issue: ZK-Email proof verification fails

**Solution**: Ensure the proof in `zk-transaction.json` matches the exact transaction being signed (sign bytes must match)

### Issue: ZK-Email authenticator not found

**Solution**: Verify the authenticator ID matches between registration and query; check email salt format

## Integration

DKIM tests integrate with:

- **Governance module** - Proposal-based key management via `MsgAddDkimPubKeys` and `MsgRemoveDkimPubKey`
- **ZK module** - Poseidon hashes enable ZK-Email circuit verification
- **Abstract Account** - DKIM keys authenticate ZK-Email based account operations
- **Bank module** - Fund transfers in ZK-Email authentication tests

## Message Types

```go
// Add DKIM public keys (governance only)
MsgAddDkimPubKeys{
    Authority:   govModuleAddress,
    DkimPubKeys: []DkimPubKey{
        {Domain, Selector, PubKey, PoseidonHash},
    },
}

// Remove DKIM public key (governance only)
MsgRemoveDkimPubKey{
    Authority: govModuleAddress,
    DkimPubKey: DkimPubKey{Domain: domain, Selector: selector},
}

// Revoke with private key proof (anyone with private key)
MsgRevokeDkimPubKey{
    Sender:     senderAddress,
    Domain:     domain,
    PrivateKey: privateKeyPEM,
}
```

## Data Structures

```go
type DkimPubKey struct {
    Domain       string // Email domain
    Selector     string // DKIM selector
    PubKey       string // Base64-encoded RSA public key
    PoseidonHash []byte // ZK-compatible hash
}

// ZK-Email Signature format
type Signature struct {
    Proof        ProofData // Groth16 proof (pi_a, pi_b, pi_c)
    PublicInputs []string  // Circuit public inputs
}
```

---

Package: `integration_tests`
Shared utilities: `../testlib/` (BuildXionChain, ExecQuery, ExecTx, GetModuleAddress, ExecBroadcast)
