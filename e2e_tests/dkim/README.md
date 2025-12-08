# DKIM E2E Tests

This directory contains end-to-end tests for XION's DKIM (DomainKeys Identified Mail) module.

## Overview

The DKIM module stores DKIM public keys and their Poseidon hashes on-chain, enabling ZK-Email authentication for Abstract Accounts. It mimics traditional DKIM functionality by securely storing information found in email headers, allowing verification of email authenticity on the blockchain. The Poseidon hash computation provides ZK-circuit compatibility for efficient on-chain verification.

## Test Files

### dkim_test.go

Tests core DKIM module functionality:

- **TestDKIMModule** - Comprehensive DKIM operations test
  - Query single DKIM record by domain and selector
  - Query all DKIM records for a domain
  - Query by domain + Poseidon hash pair
  - Generate DKIM records from DNS lookup (`gdkim`)
  - Add DKIM records via governance proposal
  - Remove DKIM records via governance proposal
  - RSA key pair generation with Poseidon hash computation
  - Revoke DKIM keys using private key proof

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

## Running Tests

```bash
# Run DKIM tests
cd e2e_tests && go test -v -run TestDKIMModule

# Run with make target (if configured)
make test-e2e-dkim
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

1. **Genesis Verification** - Confirm pre-seeded DKIM records exist
2. **Query Tests** - Verify single record, domain-wide, and hash-filtered queries
3. **DNS Generation** - Fetch real DKIM record from DNS via `gdkim`
4. **Governance Add** - Submit proposal to add Netflix DKIM record
5. **Governance Remove** - Submit proposal to remove Netflix record
6. **Key Generation** - Create RSA-2048 key pair and compute Poseidon hash
7. **Governance Add** - Register generated key via governance
8. **Direct Revocation** - Revoke key using private key proof via `rdkim`

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

## Integration

DKIM tests integrate with:

- **Governance module** - Proposal-based key management via `MsgAddDkimPubKeys` and `MsgRemoveDkimPubKey`
- **ZK module** - Poseidon hashes enable ZK-Email circuit verification
- **Abstract Account** - DKIM keys authenticate ZK-Email based account operations

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
```

---

Package: `integration_tests`
Shared utilities: `../testlib/` (BuildXionChain, ExecQuery, ExecTx, GetModuleAddress)
