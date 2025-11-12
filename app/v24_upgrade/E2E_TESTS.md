# V24 Upgrade E2E Tests - Quick Reference

This document provides a quick reference for running the v24 upgrade e2e tests via Makefile.

## Quick Start

### Run All V24 Tests
```bash
make test-app-v24-upgrade-all
```

### Run Individual Tests

```bash
# Full upgrade workflow (6 contracts: 3 broken, 2 legacy, 1 canonical)
make test-app-v24-upgrade-full-flow

# Performance test (100 contracts)
make test-app-v24-upgrade-performance

# Idempotency test (safe to run upgrade multiple times)
make test-app-v24-upgrade-idempotency

# Dry-run analysis test
make test-app-v24-upgrade-analysis

# Edge case handling (corrupted/minimal data)
make test-app-v24-upgrade-corrupted-data

# Timeout/context management
make test-app-v24-upgrade-timeout

# Schema detection logic
make test-app-v24-upgrade-schema-detection
```

## Test Locations

The v24 upgrade tests are organized in two locations:

### 1. Integration-Style E2E Tests
**Location:** `app/v24_upgrade/e2e_upgrade_test.go`

These tests use `app.Setup(t)` for fast execution:
- `TestE2E_FullUpgradeFlow` - Complete workflow
- `TestE2E_UpgradePerformance` - Performance with 100 contracts
- `TestE2E_UpgradeIdempotency` - Idempotent behavior
- `TestE2E_UpgradeAnalysis` - Dry-run functionality
- `TestE2E_UpgradeWithCorruptedData` - Edge cases
- `TestE2E_UpgradeContextTimeout` - Timeout handling

**Run directly:**
```bash
cd app/v24_upgrade
go test -v -run TestE2E
```

### 2. Lightweight E2E Test
**Location:** `e2e_tests/app/v24_upgrade_test.go`

Unit-style test in e2e package:
- `TestV24Upgrade_SchemaDetection` - Schema detection

**Run directly:**
```bash
cd e2e_tests/app
go test -v -run TestV24Upgrade
```

## Test Summary

| Test | Purpose | Contracts | Runtime |
|------|---------|-----------|---------|
| Full Flow | End-to-end workflow | 6 | ~140ms |
| Performance | Parallel processing | 100 | ~350µs |
| Idempotency | Safe re-runs | 1 | ~150ms |
| Analysis | Dry-run stats | 6 | ~130ms |
| Corrupted Data | Edge cases | 2 | ~110ms |
| Timeout | Context management | 10 | ~130ms |
| Schema Detection | Detection logic | 3 cases | <1ms |

**Total:** 7 test scenarios, ~700ms total runtime

## Performance Benchmarks

From `TestE2E_UpgradePerformance`:
- **Throughput:** ~280,000 contracts/second
- **100 contracts:** 356µs
- **Expected mainnet (6M):** ~21 seconds

## Coverage

All v24 upgrade tests contribute to:
- **89.9% code coverage** overall
- **All functions ≥50%** individual coverage
- **227 total tests** across all test files

## CI/CD Integration

### GitHub Actions Example

```yaml
name: V24 Upgrade Tests

on: [push, pull_request]

jobs:
  v24-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Run V24 Upgrade Tests
        run: make test-app-v24-upgrade-all

      - name: Check Coverage
        run: |
          cd app/v24_upgrade
          go test -coverprofile=coverage.out
          go tool cover -func=coverage.out | grep total
```

## Debugging

### Enable Verbose Output

```bash
# For Makefile targets (automatically verbose)
make test-app-v24-upgrade-full-flow

# For direct test runs
cd app/v24_upgrade
go test -v -run TestE2E_FullUpgradeFlow
```

### Run Specific Scenarios

```bash
# Test only broken contract migration
cd app/v24_upgrade
go test -v -run TestE2E_FullUpgradeFlow

# Test only performance
cd app/v24_upgrade
go test -v -run TestE2E_UpgradePerformance
```

### Skip Performance Tests

```bash
cd app/v24_upgrade
go test -short -v
```

## Test Data

Each test creates realistic protobuf data:

```go
// Broken contract (v20/v21 schema)
map[int][]byte{
    1: []byte("code_id"),
    7: []byte("ibc_port_id"),  // Wrong!
    8: []byte("extension"),     // Wrong!
}

// Legacy contract (pre-v20 schema)
map[int][]byte{
    1: []byte("code_id"),
    7: []byte("extension"),     // Correct
    // No field 8
}

// Canonical contract (v22+ schema)
map[int][]byte{
    1: []byte("code_id"),
    7: []byte("extension"),     // Correct
    8: []byte(""),              // Empty
}
```

## Expected Results

### Full Flow Test
```
✓ Contract xion1contract001 successfully migrated
✓ Contract xion1contract002 successfully migrated
✓ Contract xion1contract003 successfully migrated
✓ Legacy contract xion1legacy001 unchanged
✓ Legacy contract xion1legacy002 unchanged
✓ Canonical contract xion1canon001 unchanged
✓ E2E upgrade flow test PASSED
```

### Performance Test
```
Performance results:
  Contracts: 100
  Duration: 356.958µs
  Rate: 280145.00 contracts/second
✓ Performance test PASSED
```

### Analysis Test
```
Analysis results:
  Total: 6
  Broken: 2
  Legacy: 3
  Canonical: 1
✓ Analysis test PASSED
```

## See Also

- [TESTING.md](./TESTING.md) - Comprehensive testing guide
- [README.md](./README.md) - V24 upgrade overview
- [make/test.mk](/make/test.mk) - Makefile test targets

## Troubleshooting

### Test Fails with "contract not found"
- Ensure test app is properly initialized
- Check that `app.Setup(t)` is called
- Verify store key is correct ("wasm")

### Performance Test Too Slow
- Check if running in debug mode
- Verify parallel workers are enabled
- Ensure no resource constraints

### Coverage Below Expected
- Run all tests: `go test ./app/v24_upgrade -cover`
- Check specific function coverage: `go tool cover -func=coverage.out`
- Review untested edge cases

## Support

For questions or issues:
1. Check test output logs
2. Run with `-v` flag for verbose output
3. Review [TESTING.md](./TESTING.md) for detailed information
4. Check individual test files for inline documentation
