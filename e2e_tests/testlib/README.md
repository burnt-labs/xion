# Test Library (testlib)

Shared utilities for XION e2e tests.

## Overview

The `testlib` package provides common functions and helpers used across all e2e test modules, reducing code duplication and ensuring consistency.

## Files

### setup.go
Chain setup and configuration:
- `BuildXionChain()` - Create test chain with default config
- `BuildXionChainWithSpec()` - Create chain with custom spec
- `XionLocalChainSpec()` - Default chain specification
- `XionChainSpec()` - Customizable chain spec
- `GetXionImage()` - Docker image for chain

### utils.go
Test execution helpers:
- `ExecTx()` - Execute transaction
- `ExecQuery()` - Query chain state
- `ExecBin()` - Execute binary command
- `IntegrationTestPath()` - Resolve resource paths
- `GetModuleAddress()` - Get module account address
- Token factory helpers
- WebAuthn utilities

### git_release.go
Release and version helpers:
- `GetLatestGithubRelease()` - Fetch latest release
- `GetGHCRPackageName()` - Container registry paths

## Key Functions

### Chain Setup

```go
// Build chain with defaults
xion := testlib.BuildXionChain(t)

// Build with custom spec
spec := testlib.XionLocalChainSpec()
spec.NumValidators = 3
xion := testlib.BuildXionChainWithSpec(t, spec)
```

### Transaction Execution

```go
// Execute transaction
txHash, err := testlib.ExecTx(t, ctx, xion.GetNode(),
    userKeyName, "bank", "send", sender, recipient, "1000uxion")

// Query state
result, err := testlib.ExecQuery(t, ctx, xion.GetNode(),
    "bank", "balances", address)
```

### Resource Paths

```go
// Load contract (paths relative to test module)
contractPath := testlib.IntegrationTestPath("contracts", "account.wasm")

// Load keys
keyPath := testlib.IntegrationTestPath("keys", "jwtRS256.key")
```

### Token Factory

```go
// Create token
denom := testlib.CreateTokenFactoryDenom(t, ctx, chain, creator, subdenom)

// Mint tokens
testlib.MintTokenFactoryDenom(t, ctx, chain, creator, amount, denom)

// Get admin
admin := testlib.GetTokenFactoryAdmin(t, ctx, chain, denom)
```

### WebAuthn

```go
// Create WebAuthn credential
cred := testlib.CreateWebAuthNAttestationCred(t)

// Generate signature
sig := testlib.CreateWebAuthNSignature(t, cred, txHash)
```

## Configuration

### Default Genesis Modifications

`DefaultGenesisKVMods` sets:
- Gov params (short voting period for tests)
- Mint params (inflation config)
- Abstract account params
- Packet forward middleware config

### Chain Specification

```go
type ChainSpec struct {
    Name          string
    ChainID       string
    Denom         string
    NumValidators int
    NumFullNodes  int
    GasAdjustment float64
    // ...
}
```

## Best Practices

### DO
- ✅ Use `BuildXionChain()` for standard tests
- ✅ Use `IntegrationTestPath()` for resource loading
- ✅ Use `ExecTx()`/`ExecQuery()` for chain interaction
- ✅ Check errors with `require.NoError(t, err)`
- ✅ Use `t.Parallel()` for independent tests

### DON'T
- ❌ Don't hardcode paths - use `IntegrationTestPath()`
- ❌ Don't use `os.Exec` directly - use helper functions
- ❌ Don't share state between parallel tests
- ❌ Don't skip error checking

## Import

```go
import "github.com/burnt-labs/xion/e2e_tests/testlib"
```

## Dependencies

- interchaintest v10
- Cosmos SDK
- Docker (for chain instances)

---

Package: `testlib`
Used by: All e2e test modules
