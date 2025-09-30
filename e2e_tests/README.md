# XION End-to-End (E2E) Tests

This directory contains comprehensive end-to-end integration tests for the XION blockchain, organized by module.

## Directory Structure

```
e2e_tests/
├── README.md                    # This file
├── setup.go                     # Chain setup and configuration
├── utils.go                     # Shared test utilities
├── testdata/                    # Shared test data (legacy, being phased out)
│   ├── contracts/              # Shared WASM contracts
│   ├── keys/                   # Shared test keys
│   └── unsigned_msgs/          # Unsigned message templates
├── aa/                         # Abstract Account tests
│   ├── README.md
│   ├── testdata/               # AA-specific test data
│   │   └── contracts/         # Account WASM contracts
│   ├── abstract_account_test.go
│   └── account_migration_test.go
├── app/                        # Application-level tests
│   ├── README.md
│   ├── testdata/               # App-specific test data
│   │   └── contracts/         # Treasury, token factory contracts
│   ├── send_test.go
│   ├── simulate_test.go
│   ├── token_factory_test.go
│   ├── treasury_test.go
│   ├── update_treasury_configs_test.go
│   ├── update_treasury_params_test.go
│   ├── upgrade_test.go
│   └── upgrade_ibc_test.go
├── dkim/                       # DKIM module tests
│   ├── README.md
│   ├── testdata/               # DKIM-specific test data
│   │   ├── contracts/         # ZK email contracts
│   │   └── keys/              # ZK proofs and keys
│   ├── dkim_test.go
│   └── zkemail_test.go
├── jwk/                        # JWK module tests
│   ├── README.md
│   ├── testdata/               # JWK-specific test data
│   │   └── keys/              # JWT RSA keys
│   ├── jwt_aa_test.go
│   ├── jwk_advanced_test.go
│   ├── jwk_test_helpers.go
│   └── IMPLEMENTATION_SUMMARY.md
├── mint/                       # Mint module tests
│   ├── README.md
│   └── mint_test.go
├── xion/                       # XION core module tests
│   ├── README.md
│   ├── minimum_fee_test.go
│   └── webauthn_test.go
└── indexer/                    # Indexer E2E tests (planned)
    └── INDEXER_INTEGRATION_TESTS.md
```

## Test Categories

### Module Tests

#### ✅ Abstract Account (AA)
**Location**: [aa/](aa/)
**Tests**: JWT authentication, account migration
**Coverage**: Basic flows, CLI operations

#### ✅ DKIM
**Location**: [dkim/](dkim/)
**Tests**: Key management, ZK email authentication
**Coverage**: Governance operations, DNS generation, ZK proofs

#### ✅ JWK
**Location**: [jwk/](jwk/)
**Tests**: JWT authentication, token validation, key rotation
**Coverage**: Comprehensive security testing including:
- Expired token rejection
- Audience mismatch protection
- Key rotation
- Multi-audience isolation

#### ✅ Mint
**Location**: [mint/](mint/)
**Tests**: Inflation, fee burning, supply management
**Coverage**: Economic scenarios (0% inflation, high fees, low fees)

#### ✅ XION Core
**Location**: [xion/](xion/)
**Tests**: Platform fees, WebAuthn, multi-denom fees
**Coverage**: Fee mechanisms, IBC tokens, security regressions

#### ✅ App (Application-Level)
**Location**: [app/](app/)
**Tests**: Transaction sends, simulation, token factory, treasury, upgrades
**Coverage**: Cross-cutting concerns, infrastructure, IBC upgrades

#### ⚠️ Indexer (Planned)
**Location**: [INDEXER_INTEGRATION_TESTS.md](INDEXER_INTEGRATION_TESTS.md)
**Tests**: State streaming, authz/feegrant indexing
**Status**: Unit tests exist, E2E tests needed

## Running Tests

### Quick Start

```bash
# Run all E2E tests
make test-e2e

# Compile E2E test binary
make compile-e2e-tests

# Run specific module tests
go test -v ./e2e_tests/jwk/... -timeout 20m
go test -v ./e2e_tests/dkim/... -timeout 15m
go test -v ./e2e_tests/mint/... -timeout 15m
go test -v ./e2e_tests/xion/... -timeout 20m
go test -v ./e2e_tests/abstract-account/... -timeout 15m
go test -v ./e2e_tests/app/... -timeout 20m
```

### Running Individual Tests

```bash
# Mint module tests
make test-e2e-mint-module-inflation-high-fees
make test-e2e-mint-module-inflation-low-fees
make test-e2e-mint-module-inflation-no-fees
make test-e2e-mint-module-no-inflation-no-fees

# JWK module tests
make test-e2e-jwt-abstract-account
go test -v ./e2e_tests/jwk -run TestJWKExpiredToken
go test -v ./e2e_tests/jwk -run TestJWKAudienceMismatch

# XION module tests
make test-e2e-xion-min-default
make test-e2e-xion-min-zero
make test-e2e-xion-min-multi-denom
make test-e2e-web-auth-n-abstract-account

# Abstract Account tests
make test-e2e-abstract-account-migration
make test-e2e-xion-abstract-account
```

### With Coverage

```bash
# Generate coverage for all E2E tests
go test -v ./e2e_tests/... -coverprofile=e2e_coverage.out -timeout 60m
go tool cover -html=e2e_coverage.out -o e2e_coverage.html

# Module-specific coverage
go test -v ./e2e_tests/jwk/... -coverprofile=jwk_e2e.out
go test -v ./e2e_tests/mint/... -coverprofile=mint_e2e.out
```

## Test Requirements

### System Requirements
- **Docker**: Required for chain instantiation
- **Memory**: 4-8GB available RAM
- **CPU**: 2+ cores recommended
- **Disk**: 5-10GB temporary storage
- **Go**: 1.21+

### Time Requirements
- **Individual test**: 2-5 minutes
- **Module test suite**: 10-20 minutes
- **Full E2E suite**: 1-2 hours

### Environment
All tests use interchaintest to spin up local chains in Docker containers. No external services required.

## Test Data Organization

### Shared Test Data
Located in [testdata/](testdata/):
- `contracts/` - WASM contracts for AA tests
- `keys/` - JWT keys, ZK proofs

### Module-Specific Data
Each module directory may contain its own testdata if needed.

## Test Patterns

### Chain Setup
```go
// Use BuildXionChain for default 3 validators
xion := BuildXionChain(t)

// Use BuildXionChainWithSpec for custom configuration
spec := XionLocalChainSpec(t, 3, 1)
spec.GasPrices = "0.025uxion"
xion := BuildXionChainWithSpec(t, spec)
```

### User Accounts
```go
// Create and fund test users
fundAmount := math.NewInt(10_000_000)
users := ibctest.GetAndFundTestUsers(t, ctx, "default", fundAmount, xion)
user := users[0]
```

### Transaction Execution
```go
// Execute CLI transaction
txHash, err := ExecTx(t, ctx, xion.GetNode(),
    user.KeyName(),
    "bank", "send", user.KeyName(),
    "--chain-id", xion.Config().ChainID,
    recipientAddr, "1000uxion",
)

// Query blockchain state
result, err := ExecQuery(t, ctx, xion.GetNode(),
    "bank", "balances", addr)
```

## Module Test Coverage

| Module | Tests | Coverage Status | Priority |
|--------|-------|-----------------|----------|
| **JWK** | 5 tests | ✅ Excellent | Complete |
| **DKIM** | 2 tests | ✅ Good | Enhancement recommended |
| **Mint** | 4 tests | ✅ Excellent | Complete |
| **XION** | 7 tests | ✅ Excellent | Complete |
| **AA** | 2+ tests | ✅ Good | Enhancement recommended |
| **Indexer** | 0 tests | ❌ **None** | **CRITICAL** |
| **FeeAbs** | 0 tests | ❌ **None** | **CRITICAL** |

### Test Gap Analysis
See [INDEXER_INTEGRATION_TESTS.md](INDEXER_INTEGRATION_TESTS.md) for detailed recommendations.

## Common Issues & Solutions

### "Docker daemon not running"
```bash
# Start Docker Desktop or docker service
systemctl start docker  # Linux
open -a Docker          # macOS
```

### "Port already in use"
```bash
# Kill existing chains
docker ps | grep xiond | awk '{print $1}' | xargs docker kill
```

### "Test timeout"
```bash
# Increase timeout for slow systems
go test -v ./e2e_tests/... -timeout 120m
```

### "Out of memory"
```bash
# Run tests sequentially instead of parallel
go test -v ./e2e_tests/... -p 1
```

## CI/CD Integration

### GitHub Actions Workflow

The CI/CD pipeline uses a **multi-binary compilation strategy** for efficient parallel test execution:

#### Build Phase (`.github/workflows/binaries-test.yaml`)

Compiles separate test binaries for each module:

```bash
# Build individual test binaries for each module
for dir in abstract-account app dkim jwk xion; do
  go test -c -o ../dist/${dir}.test ./${dir}
done
```

This creates:
- `jwk.test` - JWK and JWT tests
- `dkim.test` - DKIM and ZK email tests
- `app.test` - Application-level tests (IBC, governance, treasury)
- `xion.test` - XION core tests (fees, WebAuthn)
- `abstract-account.test` - Abstract Account tests

#### Test Phase (`.github/workflows/integration-tests.yaml`)

Routes individual tests to the correct binary based on test name:

```yaml
- name: Determine test binary
  run: |
    TEST_NAME="${{ matrix.test_type }}"
    case "$TEST_NAME" in
      JWK*|JWT*)        BINARY="jwk.test" ;;
      DKIM*)            BINARY="dkim.test" ;;
      IBC*|Governance*) BINARY="app.test" ;;
      Platform*|Minimum*|Treasury*|FeeGrant*|WebAuthN*)
                        BINARY="xion.test" ;;
      Multiple*|Single*)BINARY="abstract-account.test" ;;
      *)                BINARY="app.test" ;;  # default
    esac
    echo "binary=$BINARY" >> $GITHUB_OUTPUT

- name: Run integration test
  run: |
    BINARY="dist/${{ steps.determine-binary.outputs.binary }}"
    "$BINARY" -test.failfast -test.v -test.run Test${{ matrix.test_type }}
```

#### Test Name Routing

| Test Prefix | Binary | Examples |
|------------|--------|----------|
| `JWK*`, `JWT*` | `jwk.test` | TestJWKAlgorithmConfusionSecurity |
| `DKIM*` | `dkim.test` | TestDKIMKeyRotationSecurity |
| `IBC*`, `Governance*` | `app.test` | TestIBCTokenTransferSecurity |
| `Platform*`, `Minimum*`, `Treasury*`, `FeeGrant*`, `WebAuthN*` | `xion.test` | TestPlatformFeeBypassSecurity |
| `Multiple*`, `Single*` | `abstract-account.test` | TestMultipleAuthenticatorsSecurity |
| Others (legacy) | `app.test` | TestDungeonTransferBlock |

### Building Test Binaries Locally

Replicate the CI build process:

```bash
cd e2e_tests

# Build all binaries
for dir in abstract-account app dkim jwk xion; do
  go test -c -o ${dir}.test ./${dir}
done

# Run specific test
./jwk.test -test.v -test.run TestJWKAlgorithmConfusionSecurity
./app.test -test.v -test.run TestIBCTokenTransferSecurity
```

### Selective Test Execution

```yaml
# Run only changed module tests
- name: Test JWK Module
  if: contains(github.event.head_commit.modified, 'x/jwk')
  run: go test -v ./e2e_tests/jwk/...
```

## Contributing

### Adding New Tests

1. **Choose appropriate module directory**
   ```bash
   cd e2e_tests/jwk  # or dkim, mint, xion, aa
   ```

2. **Create test file**
   ```bash
   touch my_feature_test.go
   ```

3. **Follow existing patterns**
   - Use `BuildXionChain(t)` for chain setup
   - Use `ExecTx` and `ExecQuery` helpers
   - Add descriptive logging with `t.Logf()`
   - Mark long tests with `t.Parallel()`

4. **Update module README**
   - Document what the test covers
   - Add running instructions
   - Note any special requirements

5. **Add Makefile target** (optional)
   Edit `make/test.mk`:
   ```makefile
   test-e2e-my-feature: compile-e2e-tests
       $(MAKE) run-e2e-test TEST_NAME=TestMyFeature
   ```

### Test Quality Guidelines

✅ **Good Test Practices**
- Clear test name describing what is tested
- Descriptive logging at each step
- Proper error messages
- Cleanup resources (containers, files)
- Independent (no shared state)
- Deterministic (no flaky behavior)

❌ **Anti-Patterns**
- Tests depending on other test order
- Hardcoded addresses/hashes
- No error checking
- Inadequate timeouts
- Missing cleanup

## Performance Optimization

### Parallel Execution
Tests with `t.Parallel()` run concurrently:
```go
func TestMyFeature(t *testing.T) {
    t.Parallel()  // Enables parallel execution
    // ...
}
```

### Resource Management
```go
// Use short mode for quick validation
if testing.Short() {
    t.Skip("skipping in short mode")
}

// Clean up containers
t.Cleanup(func() {
    // Cleanup code
})
```

## Security Testing

E2E tests verify security-critical functionality:

### Authentication
- ✅ JWT token expiration (jwk/)
- ✅ Audience isolation (jwk/)
- ✅ Key rotation (jwk/)
- ✅ WebAuthn signature validation (xion/)
- ✅ ZK email proof verification (dkim/)

### Authorization
- ✅ Governance-only operations (dkim/, xion/)
- ✅ Platform fee enforcement (xion/)
- ⚠️ Authz grants (indexer/ - needs E2E tests)
- ⚠️ Feegrant allowances (indexer/ - needs E2E tests)

### Economic Security
- ✅ Inflation bounds (mint/)
- ✅ Fee burning (mint/)
- ✅ Supply conservation (mint/)
- ✅ Multi-denom fees (xion/)

## Debugging

### View Chain Logs
```bash
# Find running container
docker ps | grep xiond

# View logs
docker logs -f <container-id>
```

### Interactive Chain Access
```bash
# Exec into container
docker exec -it <container-id> /bin/sh

# Run queries
xiond query bank balances <address>
xiond query auth account <address>
```

### Test-Specific Logging
Tests log extensively. Run with `-v` flag:
```bash
go test -v ./e2e_tests/jwk -run TestJWKExpiredToken
```

## References

- [InterchainTest Documentation](https://github.com/strangelove-ventures/interchaintest)
- [Cosmos SDK Testing](https://docs.cosmos.network/main/build/building-modules/testing)
- [Module Documentation](../x/)
- [Test Gap Analysis](./INDEXER_INTEGRATION_TESTS.md)

---

**Last Updated**: 2025-09-30
**Total Tests**: 20+ end-to-end scenarios
**Test Coverage**: Core Features ✅ | Edge Cases ⚠️ | Indexer ❌
**Maintainer**: XION Development Team
