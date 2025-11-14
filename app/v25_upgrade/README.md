# V25 Upgrade - Contract Schema Migration

## Overview

The v25 upgrade fixes corrupted contract metadata that cannot be unmarshaled. This corruption stems from a field swap bug in v20/v21 where fields 7 (extension) and 8 (ibc2_port_id) were incorrectly ordered.

## Corruption Statistics

- **Total Contracts:** 14,486
- **Working (can unmarshal):** 13,380 (92.37%)
  - These have non-canonical schemas but are fully functional
  - **Do NOT need fixing** - left as-is
- **Broken (cannot unmarshal):** 1,106 (7.63%)
  - Have "illegal wireType 7" errors from field swap
  - **MUST be fixed** - migration targets these
  - **100% fixable** with field swapping logic

## Architecture

### Core Components

1. **detector.go** - Unmarshal-based corruption detection
   - Primary test: `proto.Unmarshal()` success/failure
   - States: Healthy, SchemaInconsistent, UnmarshalFails, Unfixable

2. **analyzer.go** - Deep contract analysis
   - Pattern detection (InvalidWireType, Truncated, etc.)
   - Field-level inspection
   - Fix strategy determination

3. **fixer.go** - Contract repair logic
   - Field swapping for InvalidWireType corruption
   - Schema normalization helpers
   - Validation after fixes

4. **migrator.go** - Bulk migration orchestration
   - Called from upgrade handler
   - Processes all contracts in database
   - Only fixes StateUnmarshalFails contracts
   - Progress logging every 100 fixes

5. **proto_utils.go** - Protobuf manipulation
   - Field parsing and encoding
   - Tag construction
   - Hex dumping for debugging

## Migration Strategy

### What Gets Fixed

**ONLY** contracts with `StateUnmarshalFails`:

- Cannot unmarshal to `ContractInfo`
- Have `proto: illegal wireType 7` errors
- Field 7 contains ibc_port_id data (should be in field 8)
- Field 8 is missing

### What Does NOT Get Fixed

Contracts with `StateSchemaInconsistent`:

- **Can successfully unmarshal** ✓
- Missing fields 7/8 but still functional
- Chain can read and use them without issues
- Left as-is per "if it ain't broke, don't fix it" principle

### Fix Process

For each broken contract:

1. Parse protobuf fields manually
2. Extract field 7 data (the misplaced ibc_port_id)
3. Create empty field 7 (google.protobuf.Any with length 0)
4. Move old field 7 data to field 8 (ibc2_port_id string)
5. Rebuild protobuf with corrected field order
6. Verify fix with `proto.Unmarshal()`
7. Write fixed data to store

## CLI Tools

See [CLI_TOOLS.md](CLI_TOOLS.md) for detailed usage instructions.

### Analysis Tool

```bash
# Full database analysis with corruption patterns
xiond analyze-contracts

# Test that fixes work (100% success rate expected)
xiond analyze-contracts --test-fixes

# Analyze subset of contracts
xiond analyze-contracts --limit 1000
```

### Migration Validation Tool

```bash
# Validate migration logic (dry-run, safe on production data)
xiond v25-dry-run

# Verbose output showing each contract
xiond v25-dry-run --verbose

# Test on subset
xiond v25-dry-run --limit 100
```

## Running the Upgrade

### For Testing (Local Node)

```bash
# 1. Validate migration logic first
xiond v25-dry-run
# Should show 100% fix success rate

# 2. For actual testing, use governance upgrade mechanism
# The migration runs automatically during chain upgrade
# triggered by upgrade module

# 3. Monitor logs during upgrade for:
#   - "v25: starting contract migration"
#   - "v25 migration progress" (every 100 fixes)
#   - "v25 migration complete" (final stats)

# 4. Verify all contracts fixed after upgrade
xiond analyze-contracts --test-fixes
# Should show: 0 broken contracts!
```

### For Production (Governance)

1. Create governance proposal for v25 upgrade
2. Proposal passes and upgrade height is set
3. At upgrade height, chain automatically:
   - Halts consensus
   - Runs v25 upgrade handler
   - Executes `MigrateContracts()`
   - Resumes with fixed contracts

## File Structure

```sh
app/v25_upgrade/
├── README.md                  # This file
├── CLI_TOOLS.md               # Detailed CLI tool documentation
├── types.go                   # Core types, states, patterns
├── detector.go                # Unmarshal-based detection
├── analyzer.go                # Deep analysis and patterns
├── fixer.go                   # Repair logic with validation
├── migrator.go                # Bulk migration for upgrade handler
├── proto_utils.go             # Protobuf field manipulation
├── cmd_analyze_contracts.go   # CLI: analyze-contracts command
├── cmd_v25_dry_run.go         # CLI: v25-dry-run command
└── *_test.go                  # Test suite (53 tests)
```

## Testing Strategy

1. **Unit Tests** - 53 comprehensive tests covering all components
2. **CLI Validation** - Test fix logic with `analyze-contracts --test-fixes`
3. **Dry-run Migration** - Validate with `v25-dry-run` (100% success required)
4. **Testnet Validation** - Run on testnet before mainnet
5. **Mainnet Upgrade** - Execute via governance proposal

## Success Criteria

- ✅ All 1,106 broken contracts can unmarshal after migration
- ✅ 13,380 working contracts unchanged (not modified)
- ✅ 0 contracts in StateUnmarshalFails after migration
- ✅ Chain can read all contract metadata
- ✅ No data loss or corruption introduced

## Key Design Principles

1. **Unmarshal-First Detection** - Only `proto.Unmarshal()` matters for functionality
2. **Fix Only What's Broken** - Don't modify working contracts
3. **Validate Every Fix** - Verify unmarshal succeeds after repair
4. **Fail Fast** - Panic if any contract cannot be fixed
5. **Comprehensive Logging** - Track progress and issues

## Lessons from v24

v24 upgrade failed because:

- Used `ParseProtobufFields()` which could succeed when `proto.Unmarshal()` failed
- Fixed contracts that didn't need fixing
- Incorrect assumptions (e.g., field 8 must always be empty)

v25 improvements:

- **Unmarshal is the only test** - if unmarshal works, contract is functional
- **Only fix truly broken contracts** - skip the 92% that already work
- **No incorrect assumptions** - field 8 can have data OR be empty
- **Proper field swapping** - correctly handles v20/v21 bug
