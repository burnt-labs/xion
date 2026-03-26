# Xion Blockchain - Copilot Instructions

## Overview

Xion is a Cosmos SDK-based blockchain optimized for consumer applications with a focus on user experience through Abstract Accounts. This is a **CosmWasm-enabled chain** forked from [CosmWasm/wasmd](https://github.com/CosmWasm/wasmd) with custom modules for Web3 authentication, zero-knowledge proofs, and platform fee management.

## Project Architecture

### Core Technology Stack
- **Language**: Go 1.25.3
- **Framework**: Cosmos SDK v0.50+
- **Smart Contracts**: CosmWasm v3.0+ (wasmvm/v3)
- **Consensus**: CometBFT
- **IBC**: v10 with wasm light clients support
- **Database**: LevelDB (pinned to specific version)

### Custom Modules (x/)

1. **x/dkim** - DKIM public key management for email authentication
   - Stores DNS-verified DKIM public keys on-chain
   - Computes Poseidon hashes for ZK email verification
   - Governance-controlled key management
   - Location: `/home/runner/work/xion/xion/x/dkim/`

2. **x/jwk** - JSON Web Key management for JWT authentication
   - Manages JWT public keys for Abstract Account authentication
   - Supports key rotation and audience validation
   - Security-critical: validates expired tokens, audience mismatch
   - Location: `/home/runner/work/xion/xion/x/jwk/`

3. **x/xion** - Core platform functionality
   - Platform fee management (percentage and minimums)
   - WebAuthn signature validation utilities
   - Multi-denomination fee support
   - Location: `/home/runner/work/xion/xion/x/xion/`

4. **x/zk** - Zero-knowledge proof verification
   - Groth16 proof verification
   - Configurable proof size limits (max 896 bytes default)
   - Public input validation
   - Location: `/home/runner/work/xion/xion/x/zk/`

5. **x/mint** - Custom minting and fee burning
   - Inflation control and supply management
   - Fee burning mechanism
   - Location: `/home/runner/work/xion/xion/x/mint/`

6. **x/globalfee** - Global minimum fee enforcement
   - Multi-denom minimum fees
   - Platform fee integration
   - Location: `/home/runner/work/xion/xion/x/globalfee/`

7. **x/feeabs** - Fee abstraction for gas payments in multiple denoms
   - Location: `/home/runner/work/xion/xion/x/feeabs/`

### Abstract Account Architecture

**CRITICAL CONTEXT**: Xion uses an **Abstract Account (AA) architecture** where:
- User accounts are **CosmWasm smart contracts** (not standard SDK accounts)
- Authentication is **contract-based** (contracts validate their own signatures)
- WebAuthn/JWT credentials are **stored in contract state**
- The `x/xion` and `x/jwk` modules provide **utility functions** for validation
- Authorization happens at the **contract level**, not module level

**Security Model**: The Abstract Account dependency (`github.com/burnt-labs/abstract-account`) implements the core AA logic. When working on authentication/authorization features, understand that:
1. Modules provide cryptographic validation utilities
2. Real authentication happens in smart contracts
3. An attacker cannot bypass AA security by manipulating module queries

## Build System

### Makefile Structure (Modular)
```
Makefile                    # Main entry point
├── make/build.mk          # Build and release targets
├── make/test.mk           # Testing targets
├── make/coverage.mk       # Coverage analysis (85% threshold required)
├── make/proto.mk          # Protobuf generation (requires Docker)
└── make/lint.mk           # Linting and formatting
```

### Essential Commands

#### Development Workflow
```bash
# Install xiond binary
make install

# Run unit tests with coverage
make test-cover              # Development: shows progress
make test-cover-ci           # CI: single report at end

# Lint code (golangci-lint)
make lint

# Format code (gofumpt + gci)
make format

# Generate protobuf files (requires Docker)
make proto-all
```

#### Testing Strategy
```bash
# Unit tests (fast, no Docker)
make test-unit               # All unit tests
make test-race               # With race detector

# E2E tests (requires Docker, slow)
make test-e2e-all           # All integration tests (1-2 hours)
make test-jwk-all           # JWK module tests
make test-dkim-all          # DKIM module tests
make test-xion-all          # XION core tests
make test-aa-all            # Abstract Account tests
```

#### Coverage Requirements
- **Threshold**: 85% total coverage (enforced in CI)
- **Excluded**: `api/` and `cmd/` packages
- **Report types**: HTML, filtered (no .pb.go files), detailed analysis
- Coverage config: `.coveragerc` (Python format, parsed by Makefile)

### Docker Requirements
- **Required for**: Protobuf generation, E2E tests
- **Image**: `xiond:local` (auto-built by `make .ensure-docker-image`)
- **Memory**: 4-8GB for E2E tests
- **Time**: Individual E2E test: 2-5 min, Full suite: 1-2 hours

## Code Style and Conventions

### Linting Configuration (`.golangci.yml`)

**Enabled Linters**:
- `errcheck`, `govet`, `staticcheck`, `ineffassign`, `unused`
- `gosec` (security, but G404, G115 excluded)
- `gocritic` (appendAssign disabled)
- `revive` (var-naming disabled)
- `misspell`, `nakedret`, `unconvert`, `unparam`
- `bodyclose`, `copyloopvar`, `dogsled`

**Disabled Checks**:
- `-SA1019`: Deprecated API usage (TODO: fix module.AppModule, MustSortJSON)
- `-ST1003`: Naming convention violations (TODO: fix)

### Import Ordering (gci)
```go
import (
    // 1. Standard library
    "context"
    "fmt"

    // 2. Default (third-party)
    "github.com/spf13/cobra"

    // 3. CometBFT
    "github.com/cometbft/cometbft/..."

    // 4. Cosmos ecosystem (generic)
    "github.com/cosmos/..."

    // 5. Cosmos SDK (cosmossdk.io)
    "cosmossdk.io/errors"
    "cosmossdk.io/math"

    // 6. Cosmos SDK (github)
    "github.com/cosmos/cosmos-sdk/types"

    // 7. Xion (this repo)
    "github.com/burnt-labs/xion/x/dkim/types"
)
```

### Code Formatting
- **Tool**: `gofumpt` (stricter than `gofmt`)
- **Command**: `make format` (runs gci + gofumpt)
- **Auto-fix**: Most linters support `--fix` flag

### Testing Conventions

#### Unit Tests
```go
// Use testify/require and testify/suite
import (
    "github.com/stretchr/testify/require"
    "github.com/stretchr/testify/suite"
)

// Test fixture pattern
type TestFixture struct {
    suite.Suite
    ctx         sdk.Context
    k           keeper.Keeper
    msgServer   types.MsgServer
    queryServer types.QueryServer
}

func SetupTest(t *testing.T) *TestFixture {
    // Setup code...
}

// Tags for build
// -tags='ledger test_ledger_mock'
```

#### E2E Tests (InterchainTest)
```go
// Chain setup
xion := BuildXionChain(t)  // Default 3 validators
// OR
spec := XionLocalChainSpec(t, 3, 1)
spec.GasPrices = "0.025uxion"
xion := BuildXionChainWithSpec(t, spec)

// Transaction execution
txHash, err := ExecTx(t, ctx, xion.GetNode(),
    user.KeyName(), "bank", "send", ...)

// Query execution
result, err := ExecQuery(t, ctx, xion.GetNode(),
    "bank", "balances", addr)
```

**E2E Test Organization** (`e2e_tests/`):
- Tests are organized by module: `aa/`, `app/`, `dkim/`, `jwk/`, `xion/`, `mint/`, `indexer/`
- Each module has its own `testdata/` with contracts, keys, proofs
- Multi-binary compilation strategy for CI parallelization
- Tests use `t.Parallel()` when possible

## Critical Patterns and Gotchas

### 1. Params Migration Pattern (x/zk, x/dkim)
**Context**: New params fields need backfill logic for existing chains.

```go
// In types/params.go
func (p Params) WithMaxLimitDefaults() Params {
    if p.MaxProofSize == 0 {
        p.MaxProofSize = 896
    }
    if p.MaxPublicInputs == 0 {
        p.MaxPublicInputs = 24
    }
    return p
}

// In keeper GetParams/SetParams
func (k Keeper) GetParams(ctx context.Context) (types.Params, error) {
    params, err := k.Params.Get(ctx)
    if err != nil {
        return types.Params{}, err
    }
    return params.WithMaxLimitDefaults(), nil
}
```

**Why**: When new params are added, stored params have zero values. Backfill ensures defaults apply without requiring migration proposals.

**Files**: `x/zk/types/params.go:64-87`, `x/zk/keeper/keeper.go:179-203`

### 2. JWT/JWS Security Pattern (x/jwk)
**Context**: Prevent JWS JSON serialization attacks.

```go
// Use compact-only JWT settings globally
jwt.Settings(jwt.WithCompactOnly(true))

// AND leading-byte check
if len(token) > 0 && token[0] == '{' {
    return errors.New("JSON serialization not allowed")
}

// For JWS verification
jws.WithCompact()
```

**Why**: JSON serialization allows algorithm confusion attacks. Enforce compact format only.

**Files**: `x/jwk/keeper/query_validate_jwt.go:47-76`, `x/jwk/keeper/query_verify_jws.go:40-60`

### 3. RSA Key Size Validation (x/dkim)
**Context**: DKIM keys need minimum 2048-bit RSA, but genesis has legacy 1024-bit keys.

```go
const MinRSAKeyBits = 2048

// ValidateRSAKeySize enforces MinRSAKeyBits
// ValidateDkimPubKeys is lax (for genesis)
// ValidateDkimPubKeysWithRevocation(enforceMinKeySize=true) is strict (for messages)
```

**Why**: Legacy keys (yahoo.com s1024) exist in genesis. New keys via governance must meet modern standards.

**Files**: `x/dkim/types/pubkey.go:12-52`, `x/dkim/types/genesis.go:90-92`

### 4. WebAuthn Security Model (x/xion)
**CRITICAL**: WebAuthn validation functions are **utilities**, NOT authentication.

```go
// These are SAFE despite client-controlled params:
// - WebAuthNVerifyRegister
// - WebAuthNVerifyAuthenticate

// Why? Real authorization happens in Abstract Account contracts.
// See x/xion/keeper/grpc_query.go for detailed security documentation.
```

**Files**: `x/xion/README.md:29-62`, `x/xion/keeper/grpc_query.go`

### 5. Store Migration Pattern
```go
// In module.go
func (am AppModule) RegisterServices(cfg module.Configurator) {
    types.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper))
    types.RegisterQueryServer(cfg.QueryServer(), keeper.NewQuerier(am.keeper))

    // Register migrator
    m := keeper.NewMigrator(am.keeper)
    if err := cfg.RegisterMigration(types.ModuleName, 1, m.Migrate1to2); err != nil {
        panic(err)
    }
}

// Consensus version in module.go
func (AppModule) ConsensusVersion() uint64 { return 2 }
```

## Protobuf and Code Generation

### Structure
```
proto/xion/
├── dkim/v1/         # DKIM module protos
├── jwk/v1/          # JWK module protos
├── feeabs/v1beta1/  # Fee abstraction protos
├── v1/              # Core xion module protos
├── indexer/         # Indexer protos
└── zk/v1/           # ZK proofs protos
```

### Generation Commands
```bash
# Full pipeline (format, lint, generate, swagger)
make proto-all

# Individual steps
make proto-format           # Format .proto files
make proto-lint            # Lint with buf
make proto-check-breaking  # Check for breaking changes
make proto-gen             # Generate Go code
make proto-gen-openapi     # Generate OpenAPI specs
```

### Important Notes
- **All protobuf commands require Docker**
- Uses `buf` for linting and breaking change detection
- Generates both gogo and pulsar protobuf bindings
- Generated files: `x/*/types/*.pb.go`, `x/*/types/*.pb.gw.go`, `api/`

## Dependencies and Forks

### Critical Forks
```go
// go.mod replace directives:
github.com/CosmWasm/wasmd => github.com/burnt-labs/wasmd v0.61.8-xion.2
  // Reason: Genesis exports for wasmd

github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10 => github.com/burnt-labs/ibc-go/modules/light-clients/08-wasm/v10 v10.5.0-xion.1
  // Reason: wasmvm3 support

github.com/strangelove-ventures/tokenfactory => github.com/burnt-labs/tokenfactory v0.53.4-xion.2
  // Reason: wasmvm3 tokenfactory fork

// Pinned for stability:
github.com/syndtr/goleveldb => github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7
  // Reason: 126854af5e6d has store query issues
```

### External Dependencies
- **Abstract Account**: `github.com/burnt-labs/abstract-account v0.1.3`
- **InterchainTest**: For E2E testing (not in go.mod, used via e2e_tests)
- **Heighliner**: For Docker image builds

## CI/CD Workflows

### GitHub Actions Structure
```
.github/workflows/
├── build-test.yaml              # Main build and unit test
├── e2e-tests.yaml              # Integration tests (parallel)
├── golangci-lint.yaml          # Linting
├── tests.yaml                  # Unit test workflow
├── docker-build.yaml           # Docker image builds
├── exec-goreleaser.yaml        # Release builds
├── binaries-linux.yaml         # Linux binary builds
├── binaries-darwin.yaml        # macOS binary builds
├── update-swagger.yaml         # Swagger generation
└── heighliner.yaml            # Heighliner builds
```

### E2E Test Strategy (CI)
**Multi-binary compilation approach**:
1. Build separate test binaries for each module: `jwk.test`, `dkim.test`, `app.test`, `xion.test`, `abstract-account.test`
2. Matrix strategy routes tests to correct binary based on test name prefix
3. Enables massive parallelization (30+ concurrent tests)

**Test Name Routing**:
- `JWK*`, `JWT*` → `jwk.test`
- `DKIM*` → `dkim.test`
- `IBC*`, `Governance*` → `app.test`
- `Platform*`, `Minimum*`, `Treasury*`, `FeeGrant*`, `WebAuthN*` → `xion.test`
- `Multiple*`, `Single*` → `abstract-account.test`

### Local E2E Test Binary Build
```bash
cd e2e_tests
for dir in abstract-account app dkim jwk xion; do
  go test -c -o ${dir}.test ./${dir}
done

# Run specific test
./jwk.test -test.v -test.run TestJWKExpiredToken
```

## Common Errors and Solutions

### 1. "proto generation failed"
- **Cause**: Docker not running or not installed
- **Fix**: Ensure Docker is running: `docker ps`

### 2. "coverage threshold not met"
- **Cause**: New code lacks sufficient test coverage (85% required)
- **Fix**: Add unit tests, check coverage with `make test-cover-html`

### 3. "store query fails after upgrade"
- **Cause**: Using wrong goleveldb version (126854af5e6d)
- **Fix**: Verify go.mod has pinned version `v1.0.1-0.20210819022825-2ae1ddf74ef7`

### 4. "E2E test timeout"
- **Cause**: System too slow or Docker resource limits
- **Fix**: Increase timeout `go test -timeout 30m`, allocate more Docker memory

### 5. "import ordering lint failure"
- **Cause**: Imports not following gci sections
- **Fix**: Run `make format` to auto-fix

### 6. "deprecated API usage (SA1019)"
- **Context**: Known issue, currently suppressed
- **Don't fix yet**: Requires coordinated SDK migration
- **Affected**: `module.AppModule`, `MustSortJSON`, etc.

## Security Considerations

### Reporting Security Issues
- **Email**: security@burnt.com
- **DO NOT** open public GitHub issues for security vulnerabilities
- Follow coordinated disclosure process (SECURITY.md)

### Security-Critical Areas
1. **JWT/JWK validation** (x/jwk) - Algorithm confusion, token expiration, audience validation
2. **WebAuthn verification** (x/xion) - Abstract Account authentication utilities
3. **ZK proof verification** (x/zk) - Proof size limits, input validation
4. **DKIM key management** (x/dkim) - RSA key size, governance-only operations
5. **Platform fees** (x/xion) - Fee bypass prevention, multi-denom validation

### Testing Security Features
- **JWT security**: `e2e_tests/jwk/` - Expired token, audience mismatch, key rotation
- **WebAuthn**: `e2e_tests/xion/webauthn_test.go`
- **ZK proofs**: `e2e_tests/dkim/zkemail_test.go`
- **Fee enforcement**: `e2e_tests/xion/minimum_fee_test.go`

## Documentation Locations

### Module Documentation
- x/dkim: `/home/runner/work/xion/xion/x/dkim/README.md`
- x/xion: `/home/runner/work/xion/xion/x/xion/README.md`
- E2E tests: `/home/runner/work/xion/xion/e2e_tests/README.md`

### Build Documentation
- Main README: `/home/runner/work/xion/xion/README.md`
- Security Policy: `/home/runner/work/xion/xion/SECURITY.md`
- Installer Guide: `/home/runner/work/xion/xion/INSTALLERS.md`

### Online Resources
- Cosmos SDK docs: https://docs.cosmos.network/
- CosmWasm docs: https://docs.cosmwasm.com/
- InterchainTest: https://github.com/strangelove-ventures/interchaintest

## Working with Agents

### When to Run Different Tests
1. **During development**: `make test-unit` (fast feedback)
2. **Before commit**: `make lint && make test-cover` (ensure coverage)
3. **Module changes**: Module-specific E2E test (e.g., `make test-jwk-all`)
4. **Before PR**: `make test-unit test-race` + affected module E2E tests
5. **Major changes**: Full `make test-e2e-all` (rare, expensive)

### Quick Validation Workflow
```bash
# 1. Format code
make format

# 2. Lint
make lint

# 3. Unit tests with coverage
make test-cover

# 4. If module-specific (e.g., x/jwk changed)
make test-jwk-all
```

### Build Artifacts
- Binary: `build/xiond` or `$GOPATH/bin/xiond` (after `make install`)
- Coverage: `coverage.out`, `coverage.html`
- Docker image: `xiond:local`
- Release binaries: `dist/` (after `make release-snapshot`)

## Troubleshooting Checklist

Before asking for help, verify:

1. **Go version**: `go version` should show `1.25.3`
2. **Docker running**: `docker ps` succeeds
3. **Dependencies updated**: `go mod download`
4. **Clean build**: `make clean && make install`
5. **Format applied**: `make format`
6. **Linter passing**: `make lint`
7. **Coverage threshold met**: `make test-cover-validate`

## Common Agent Tasks

### "Add a new module parameter"
1. Update `proto/xion/{module}/v1/{module}.proto` with new field
2. Run `make proto-all` to regenerate code
3. Add validation in `x/{module}/types/params.go`
4. Implement `WithDefaults()` pattern for backfill (see x/zk example)
5. Update `GetParams`/`SetParams` to call `WithDefaults()`
6. Add migration if needed (increment ConsensusVersion)
7. Write unit tests for validation and migration
8. Update module README.md

### "Add a new gRPC query"
1. Define query in `proto/xion/{module}/v1/query.proto`
2. Run `make proto-all`
3. Implement handler in `x/{module}/keeper/query_server.go`
4. Add CLI wrapper in `x/{module}/client/cli/query.go`
5. Write unit tests in `keeper/query_server_test.go`
6. Add E2E test if security-critical
7. Update module README with query documentation

### "Fix a security vulnerability"
1. **DO NOT** commit the fix immediately to public branches
2. Follow SECURITY.md disclosure process
3. Coordinate with CosmWasm team if affects CosmWasm components
4. Write regression test in `e2e_tests/` (security suite)
5. Document the vulnerability and fix in commit message
6. Consider if other modules have similar patterns

### "Update a dependency"
1. Check if it's a forked dependency (see go.mod replace directives)
2. If forked: update in `burnt-labs/{repo}` first, tag, then update go.mod
3. If not forked: `go get {package}@{version}`
4. Run full test suite: `make test-all`
5. Check for breaking changes: `make proto-check-breaking` if SDK updated
6. Update replace directives if needed
7. Verify E2E tests: `make test-e2e-all`

## Key Takeaways for Agents

1. **This is a Cosmos SDK chain with CosmWasm support** - follow Cosmos conventions
2. **Abstract Accounts are fundamental** - user accounts are smart contracts
3. **Security is paramount** - JWT, WebAuthn, ZK proofs are security-critical
4. **Coverage matters** - 85% threshold is enforced
5. **E2E tests are expensive** - use judiciously, prefer unit tests
6. **Protobuf requires Docker** - don't try to generate without it
7. **Multiple forks in use** - check go.mod replace directives before updating deps
8. **Module documentation is critical** - update READMEs when changing module behavior
9. **Import ordering is enforced** - use `make format` to fix automatically
10. **CI is comprehensive** - lint, test, build, E2E all run on PRs

## Need Help?

- **Build issues**: Check README.md build targets section
- **Test failures**: Check e2e_tests/README.md common issues
- **Security concerns**: Email security@burnt.com
- **Module questions**: Read module-specific README in `x/{module}/`
- **Cosmos SDK questions**: https://docs.cosmos.network/
- **Code owners**: See .github/CODEOWNERS
