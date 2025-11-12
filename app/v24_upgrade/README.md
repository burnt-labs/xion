# V24 Upgrade: Comprehensive ContractInfo Migration

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Background](#background)
3. [Migration Strategy](#migration-strategy)
4. [Implementation](#implementation)
5. [Deployment Guide](#deployment-guide)
6. [Testing](#testing)
7. [Post-Upgrade](#post-upgrade)
8. [Troubleshooting](#troubleshooting)
9. [Technical Details](#technical-details)
10. [References](#references)

## Executive Summary

The v24 upgrade performs a comprehensive migration of all CosmWasm contracts to a canonical ContractInfo protobuf structure. This migration:

- **Fixes** contracts affected by wasmd v0.61.0-v0.61.4 field ordering bugs
- **Mainnet (v21→v24)**: Migrates 6M contracts (mix of SchemaLegacy and SchemaBroken)
- **Testnet (v23→v24)**: Migrates ~100K of 500K contracts (mixed schemas)
- **Enables** removal of forked wasmd dependency in future upgrades
- **Completes** within 2-hour time constraint with automatic rollback on failure

## Background

### The Protobuf Field Ordering Bug

During XION's evolution from v18 to v23, different versions of wasmd introduced breaking changes to the `ContractInfo` protobuf structure:

1. **wasmd PR #2123 (March 2025)**: Introduced field reordering bug
   - Moved `extension` field from position 7 to position 8
   - Added `ibc_port_id` at position 7

2. **wasmd PR #2390 (November 2025)**: Fixed field ordering
   - Restored `extension` to position 7
   - Moved `ibc_port_id_v2` to position 8

### Timeline of Wasmd Versions

#### Mainnet Path (Simplified with IBCv2 Never Used)
| Version | Wasmd Version | Schema | Field 8 Status | Action Needed |
|---------|---------------|--------|----------------|---------------|
| Pre-v20 | < v0.61.0 | SchemaLegacy | Empty (correct) | None - already safe |
| v20 | v0.61.0 | SchemaBroken | Has extension data | Swap fields 7-8 |
| v21 | v0.61.4 | SchemaBroken | Has extension data | Swap fields 7-8 |
| **v24** | **Target** | **SchemaCanonical** | Empty (nullified) | **Migration target** |

**Key Points**:
- Pre-v20 contracts are **already safe** - they have extension at position 7 and empty field 8
- v20 & v21 contracts have extension incorrectly at position 8 (needs swapping back)
- Detection is **simple**: just check if field 8 has data
- After migration, ALL contracts will have null field 8 (since IBCv2 is never used)

#### Testnet Path (Simplified with IBCv2 Never Used)
| Version | Wasmd Version | Schema | Field 8 Status | Action Needed |
|---------|---------------|--------|----------------|---------------|
| Pre-v20 | < v0.61.0 | SchemaLegacy | Empty (correct) | None - already safe |
| v20 | v0.61.0 | SchemaBroken | Has extension data | Swap fields 7-8 |
| v21 | v0.61.4 | SchemaBroken | Has extension data | Swap fields 7-8 |
| v22 | v0.61.6 | SchemaCanonical | Empty (correct) | None - already safe |
| v23 | v0.61.6-xion.3 | Mixed schemas | Varies | Check field 8 |
| **v24** | **Target** | **SchemaCanonical** | Empty (nullified) | **Migration target** |

**Key Points**:
- Same simple detection as mainnet: check if field 8 has data
- Pre-v20 and v22 contracts are already safe (empty field 8)
- v20 & v21-era contracts need field swapping

### Known Issues

- **Error signature**: `proto: illegal wireType 7`
- **Impact**: Contracts with corrupted `ContractInfo` cannot be deserialized
- **Example corrupted contract**: `xion1nfd77m9nuhhpsj2a7pcky6kza7htqnt605563d8wuszxnxt3vrkssv4rtg`
- **Creation block**: 9,137,467 (v22 window)

**Critical**: Some contracts created on v21 (and possibly before) still have corruption issues on v23, meaning block height alone cannot determine which contracts need migration.

### Important: IBCv2 Never Used

**XION has never used IBCv2 functionality**, which significantly simplifies our migration:
- The `ibc2_port_id` field (position 8) should always be empty/null for all legitimate contracts
- Any non-null value in field 8 indicates SchemaBroken (fields were swapped during v20/v21)
- We can safely null the `ibc2_port_id` field for ALL contracts after migration
- Pre-v20 contracts are "safe" - they have the correct field positions and null field 8

### Validation: IBCv2 Field Verification

We empirically validated that IBCv2 was never used by checking contracts across testnet:

**Contracts Tested (November 2025)**:
| Contract Code ID | Block Height | Era | ibc2_port_id Status |
|------------------|--------------|-----|-------------------|
| 33 | 541,017 | Pre-v20 | `null` ✅ |
| 33 | 1,557,888 | Pre-v20 | `null` ✅ |
| 95 | 3,487,478 | v20/v21 | `null` ✅ |
| 33 | 5,889,737 | v21 | `null` ✅ |
| 1 | 9,456,656 | v23 (post-fix) | `null` ✅ |
| 1260 | 9,452,571 | v23 (post-fix) | `null` ✅ |

**Key Findings**:
- All queryable contracts show `ibc2_port_id` as `null` or omitted (empty)
- Corrupted contract (`xion1nfd77...`) returns "illegal wireType 7" error (has data in wrong field)
- This confirms our detection strategy: any data in field 8 = corrupted contract

## Migration Strategy

### Network Differences (Simplified with IBCv2 Never Used)

Both networks use the same simple detection: check if field 8 has data.

| Aspect | Mainnet (v21→v24) | Testnet (v23→v24) |
|--------|-------------------|-------------------|
| **Total Contracts** | 6,000,000 | 500,000 |
| **Need Migration** | v20 & v21 contracts | v20 & v21-era contracts |
| **Detection Method** | Check field 8 | Check field 8 |
| **Migration Action** | Swap if field 8 has data | Swap if field 8 has data |
| **Already Safe** | Pre-v20 contracts | Pre-v20 & v22 contracts |

**Key Simplification**:
- **Universal detection**: Just check if field 8 has data (indicates SchemaBroken)
- **Single migration**: Swap fields 7-8 if needed, then null field 8
- **Pre-v20 contracts don't need changes** - already have correct structure

### Why App-Only Migration?

We chose an app-level migration (vs wasmd module migration) for:

- **Simplicity**: All migration logic in one place
- **Control**: No wasmd fork modifications needed
- **Flexibility**: Can adjust without maintaining fork
- **Future-proof**: Clean path to upstream wasmd

### Schema Versions Handled

The migration detects and handles four different schema versions:

#### 1. SchemaLegacy (Pre-v0.61.0)
```protobuf
message ContractInfo {
  // Fields 1-6: Standard fields
  google.protobuf.Any extension = 7;  // Correct position
  // No ibc2_port_id field
}
```

#### 2. SchemaBroken (v0.61.0 - v0.61.4)
```protobuf
message ContractInfo {
  // Fields 1-6: Standard fields
  string ibc_port_id = 7;              // WRONG position
  google.protobuf.Any extension = 8;   // WRONG position
}
```

#### 3. SchemaIntermediate (v0.61.5 - v0.61.6)
```protobuf
message ContractInfo {
  // Fields 1-6: Standard fields
  google.protobuf.Any extension = 7;   // Correct position
  string ibc2_port_id = 8;             // May have type variations
}
```

#### 4. SchemaCanonical (Target)
```protobuf
message ContractInfo {
  uint64 code_id = 1;
  string creator = 2;
  string admin = 3;
  string label = 4;
  AbsoluteTxPosition created = 5;
  string ibc_port_id = 6;
  google.protobuf.Any extension = 7;   // Correct position
  string ibc2_port_id = 8;             // Correct position & type
}
```

### Migration Phases by Network

#### Mainnet Phases (v21→v24, 6M contracts, Simplified)

**Phase 1: Preparation** (5 minutes)
- Count total contracts (6M)
- Initialize 40 worker goroutines
- Set up simple field 8 detection

**Phase 2: Backup Strategy** (15-20 minutes)
- Option A: Full state export before upgrade (recommended)
- Option B: Streaming backup during migration
- Critical: 6M contracts at risk

**Phase 3: Simplified Migration** (85-90 minutes)
- **Single detection**: Check if field 8 has data
- **Single transformation**: Swap fields 7-8 if needed, null field 8
- **Pre-v20 contracts**: Automatically skipped (field 8 already empty)
- **Required rate**: 1,200 contracts/second
- **Batch size**: 10,000 contracts
- **Workers**: 40 parallel goroutines
- **Progress**: Log every 250,000 contracts

**Phase 4: Validation** (10 minutes)
- Sample 0.1% (6,000 contracts)
- Verify all have empty field 8
- Verify extension at field 7

**Phase 5: Cleanup** (5 minutes)
- Database compaction
- Generate migration report

#### Testnet Phases (v23→v24, 500K contracts, Simplified)

**Phase 1: Preparation** (5 minutes)
- Count total contracts (500K)
- Initialize 10 worker goroutines
- Set up simple field 8 detection

**Phase 2: Backup** (10 minutes)
- Selective backup of contracts with non-empty field 8
- Skip pre-v20 and v22 contracts (already safe)

**Phase 3: Simplified Migration** (30-40 minutes)
- **Single detection**: Check if field 8 has data
- **Single transformation**: Swap fields 7-8 if needed, null field 8
- **Auto-skip safe contracts**: Pre-v20 and v22 (field 8 already empty)
- **Required rate**: 250 contracts/second
- **Batch size**: 5,000 contracts
- **Workers**: 10 parallel goroutines

**Phase 4: Validation** (10 minutes)
- Sample 1% (5,000 contracts)
- Verify all have empty field 8

**Phase 5: Cleanup** (5 minutes)
- Remove backup
- Final reporting

## Implementation

### Architecture

The v24 upgrade uses an app-only approach with all logic contained in the `app/v24_upgrade/` directory.

### Core Components

| File | Purpose | Status |
|------|---------|--------|
| `handler.go` | Main upgrade handler orchestration | 🔄 To enhance |
| `types.go` | Schema type definitions for all versions | 🔄 To create |
| `detector.go` | Schema detection from raw bytes | 🔄 To create |
| `migrator.go` | Core migration logic for each schema | 🔄 To create |
| `backup.go` | Backup/restore functionality | 🔄 To create |
| `validator.go` | Post-migration validation | 🔄 To create |
| `report.go` | Migration reporting and metrics | 🔄 To create |
| `proto_parser.go` | Low-level protobuf parsing | 🔄 To create |
| `constants.go` | Configuration constants | 🔄 To create |
| `corrupted.go` | Manual intervention tools | ✅ Exists |
| `example.go` | Integration examples | ✅ Exists |

### Migration Algorithm

#### Simplified Detection Algorithm (IBCv2 Never Used)
```go
// Since IBCv2 was never used, detection is simple:
// - If field 8 has ANY value, it's SchemaBroken (needs swap)
// - If field 8 is null/empty, it's already correct (SchemaLegacy or SchemaCanonical)

func DetectAndMigrate(data []byte) ([]byte, error) {
    field8Value := GetFieldValue(data, 8)

    if field8Value != nil && len(field8Value) > 0 {
        // Field 8 has data - this is SchemaBroken
        // The extension was incorrectly placed at position 8
        // Swap fields 7 and 8, then null field 8
        data = SwapFields7And8(data)
    }
    // else: Already correct (pre-v20 or post-fix contracts)

    // Ensure field 8 is null for all contracts (we never use IBCv2)
    data = NullifyField8(data)

    return data, nil
}

func ProcessContracts(ctx sdk.Context) error {
    // Simple detection based on field 8 presence
    for batch := range GetContractBatches(10000) {
        ParallelMigrate(batch, DetectAndMigrate)
    }
    return nil
}
```

#### Universal Algorithm (Both Networks - IBCv2 Simplification)
```go
// Both mainnet and testnet use the same simplified logic
// since IBCv2 was never used on either network

func MigrateContract(data []byte) ([]byte, error) {
    // Check if field 8 has any data (indicates SchemaBroken)
    field8Value := ExtractFieldValue(data, 8)

    if field8Value != nil && len(field8Value) > 0 {
        // This is SchemaBroken - extension was moved to field 8
        // Swap it back to field 7
        data = SwapFields7And8(data)
    }

    // Ensure field 8 (ibc2_port_id) is empty for all contracts
    // This handles both:
    // 1. Contracts that just had fields swapped
    // 2. Any future-proofing for contracts that somehow have garbage in field 8
    data = ClearField8(data)

    return data, nil
}

// Same algorithm works for both networks
func ProcessAllContracts(ctx sdk.Context, isMainnet bool) error {
    batchSize := 10000
    workers := 40 // For mainnet's 6M contracts
    if !isMainnet {
        workers = 10 // Testnet needs fewer workers
    }

    pool := NewWorkerPool(workers)
    for batch := range GetContractBatches(batchSize) {
        pool.Process(batch, MigrateContract)
    }
    return pool.Wait()
}
```

### Configuration Options

The v24 upgrade handler supports three operational modes (configured in handler.go):

#### Option 1: Fail on Detection (Conservative)
**Status**: Commented out by default

```go
// Uncomment to enable:
return nil, fmt.Errorf("upgrade halted: %d corrupted contracts found", len(corrupted))
```

#### Option 2: Auto-Migration (Recommended)
**Status**: To be implemented

```go
// Full migration of all schemas to canonical format
report, err := MigrateAllContracts(ctx, cdc, storeKey, MaxMigrationTime)
if err != nil {
    return nil, fmt.Errorf("migration failed: %w", err)
}
```

#### Option 3: Log and Continue (Current Default)
**Status**: Active ✅

Logs corrupted contracts and continues. Manual remediation required post-upgrade.

## Deployment Guide

### Pre-Upgrade Checklist

- [ ] Verify wasmd has canonical ContractInfo structure
- [ ] Confirm 2+ hour maintenance window available
- [ ] Backup chain data
- [ ] Prepare rollback procedure
- [ ] Test on local network

### Integration Steps

1. **Ensure v24_upgrade directory exists**
   ```bash
   ls -la app/v24_upgrade/
   ```

2. **Verify upgrades.go configuration**
   ```go
   const UpgradeName = "v24"  // Simplified name
   ```

3. **Build the upgraded binary**
   ```bash
   make build
   ```

4. **Test locally (optional)**
   ```bash
   xiond start --home ~/.xiond-test
   ```

### Upgrade Proposal

```bash
# Submit upgrade proposal for v24
xiond tx gov submit-proposal software-upgrade v24 \
  --title "V24: Comprehensive ContractInfo Migration" \
  --description "Migrates all contracts to canonical protobuf structure" \
  --upgrade-height <TARGET_HEIGHT> \
  --deposit 10000000uxion \
  --from validator \
  --chain-id xion-testnet-2
```

## Testing

### Test Strategy

1. **Unit tests** for each migration path
2. **Integration tests** with sample contracts
3. **Performance tests** with large contract sets
4. **Rollback tests** for failure scenarios

### Expected Log Output

#### Successful Migration
```
INF Starting v24 upgrade name=v24
INF Phase 1: Discovering contracts...
INF Schema distribution schema=Legacy count=5000
INF Schema distribution schema=Broken count=1000
INF Schema distribution schema=Canonical count=19000
INF Phase 2: Creating backup... total=25000
INF Phase 3: Starting migration...
INF Migration progress processed=10000 total=25000 success=10000
INF Phase 4: Validating migrations...
INF Phase 5: Cleaning up backup...
INF Migration Report duration=1h23m total=25000 successful=6000 skipped=19000
INF v24 upgrade complete name=v24
```

#### Migration with Issues
```
INF Starting v24 upgrade name=v24
ERR Found corrupted contract address=xion1nfd77m... error="proto: illegal wireType 7"
INF Migration progress processed=10000 total=25000 success=9999 failed=1
WRN Migration completed with errors failed=1
INF Continuing with upgrade...
```

### Performance Expectations

#### Scale Requirements
| Network | Total Contracts | Must Migrate | Required Rate | Workers | Batch Size |
|---------|----------------|--------------|---------------|---------|------------|
| **Mainnet** | 6,000,000 | 100% (6M) | 1,200/sec | 40 | 10,000 |
| **Testnet** | 500,000 | ~20% (100K) | 70/sec | 10 | 5,000 |

#### Key Performance Strategies
- **Mainnet**: Optimize for throughput with single transformation function
- **Testnet**: Optimize for accuracy with detection and multiple paths
- **Memory**: Bounded at 2GB using streaming iterators
- **Progress**: Log every 250K (mainnet) or 50K (testnet) contracts
- **Validation**: Statistical sampling (0.1% for mainnet, 1% for testnet)

## Post-Upgrade

### Verification Steps

1. **Check migration report in logs**
   ```bash
   journalctl -u xiond --since "2 hours ago" | grep "Migration Report"
   ```

2. **Verify random contracts**
   ```bash
   # Test random contracts can be queried
   xiond query wasm contract <ADDRESS>
   ```

3. **Check for any failed migrations**
   ```bash
   grep "failed=" ~/.xiond/xiond.log
   ```

### Path to Upstream Wasmd

After successful v24 migration:

1. ✅ All contracts in canonical format
2. ✅ Compatible with upstream wasmd v0.61.6+
3. ✅ Can remove xion fork in v25
4. ✅ Future upgrades use standard wasmd

### Manual Remediation (if needed)

If any contracts fail migration:

1. **Extract failed addresses from logs**
   ```bash
   grep "Failed to migrate" ~/.xiond/xiond.log | grep -o "xion[a-z0-9]*"
   ```

2. **Investigate each contract**
   ```bash
   xiond query wasm contract-state all <ADDRESS>
   ```

3. **Choose remediation strategy**:
   - **Delete**: If contract is unused
   - **Reconstruct**: Manual protobuf fix
   - **Redeploy**: Contact owner for new deployment

## Troubleshooting

### Mainnet-Specific: Backup Strategies for 6M Contracts

#### Option A: Pre-Upgrade State Export (Recommended)
```bash
# Before upgrade - export at specific height
xiond export --height <pre-upgrade-height> > mainnet_v21_backup.json
```
- **Pros**: Complete backup, can restore if needed
- **Cons**: Large file (100+ GB), takes 30-60 minutes

#### Option B: Streaming Backup During Migration
- Backup contracts as they're processed
- Only backup original data before transformation
- Can resume from checkpoint if interrupted

#### Option C: Trust in Validation
- Rely on extensive testnet testing
- Use statistical validation (0.1% sample)
- Have emergency response plan ready

### Common Issues

#### Issue: Migration timeout (Mainnet Risk)
**Solution**:
- Ensure 40+ workers configured
- Consider 3-hour maintenance window
- Have checkpoint/resume capability

#### Issue: Memory exhaustion with 6M contracts
**Solution**:
- Use streaming iterators, not bulk loading
- Process in 10,000 contract batches
- Monitor memory usage, add swap if needed

#### Issue: Contracts still corrupted after migration
**Solution**: Check logs for specific errors, may need manual fix

### Rollback Plan

If upgrade fails:

```bash
# Stop node
systemctl stop xiond

# Rollback to v23 binary
cp xiond-v23 /usr/local/bin/xiond

# Restart (will use pre-upgrade state)
systemctl start xiond
```

## Technical Details

### Parallel Processing Architecture for 6M Contracts

To achieve the required 1,400 contracts/second for mainnet (with detection overhead):

```go
type MigrationWorkerPool struct {
    workers   int              // 40 for mainnet
    jobQueue  chan []Contract  // Batches of 10,000
    results   chan Result
}

func (p *MigrationWorkerPool) ProcessMainnet() {
    // Launch workers
    for i := 0; i < 40; i++ {
        go p.worker()
    }

    // Feed batches
    for batch := range GetBatches(10000) {
        p.jobQueue <- batch
    }
}

// Each worker processes batches
func (p *MigrationWorkerPool) worker() {
    for batch := range p.jobQueue {
        for _, contract := range batch {
            // Detect schema type for each contract
            schema := DetectSchemaVersion(contract.Data)

            var migrated []byte
            switch schema {
            case SchemaBroken:
                migrated = SwapFields7And8(contract.Data)
            case SchemaLegacy:
                migrated = AddIBC2PortField(contract.Data)
            default:
                // Handle error or skip
                continue
            }

            SaveContract(contract.Addr, migrated)
        }
    }
}
```

### Why This Approach?

1. **Direct KV store access**: Most efficient for scanning contracts
2. **Iterator pattern**: Minimal memory overhead
3. **Parallel processing**: Essential for 6M contracts in 2 hours
4. **Schema detection**: Required to handle both SchemaLegacy and SchemaBroken
5. **Binary manipulation**: Direct protobuf field swapping and addition

### Protobuf Parsing

The migration uses low-level protobuf parsing to detect schema versions:

```go
// Parse raw protobuf without schema
func ParseProtobufFields(data []byte) map[int]FieldInfo {
    // Extract field numbers and wire types
    // Identify field positions
    // Return field map for analysis
}
```

### Contract Storage Layout

- **Store**: Wasm module KV store
- **Prefix**: `0x02` (ContractKeyPrefix)
- **Key**: prefix + contract address
- **Value**: protobuf-encoded ContractInfo
- **Backup prefix**: `0xF0` (temporary during migration)

### Migration Safety

- **Atomic operations**: All or nothing migration
- **Backup verification**: Checksums before proceeding
- **Rollback capability**: Automatic on failure
- **Validation phase**: Ensures data integrity

## References

- **Wasmd PR #2123**: Introduced field reordering bug (March 2025)
- **Wasmd PR #2390**: Fixed field ordering (November 2025)
- **Cosmos SDK Migrations**: [Module Migration Guide](https://docs.cosmos.network/main/building-modules/upgrade)
- **Protobuf Wire Format**: [Protocol Buffers Encoding](https://protobuf.dev/programming-guides/encoding/)

## Support

For questions or issues:
- **GitHub Issues**: https://github.com/burnt-labs/xion/issues
- **Discord**: #validator-support channel
- **Emergency contacts**: [Your team's emergency contact info]

---

## Appendix: Implementation Checklist

### Pre-Implementation
- [x] Verify wasmd ContractInfo structure (extension@7, ibc2_port_id@8)
- [x] Confirm app-only migration approach
- [ ] Create comprehensive test suite
- [ ] Document all schema versions

### Implementation
- [ ] Implement schema detector
- [ ] Create migration functions for each path
- [ ] Add backup/restore functionality
- [ ] Implement validation logic
- [ ] Add progress reporting
- [ ] Create integration tests

### Deployment
- [ ] Test on local network
- [ ] Deploy to testnet
- [ ] Monitor migration performance
- [ ] Verify all contracts migrated
- [ ] Document any issues
- [ ] Deploy to mainnet (if applicable)

### Post-Deployment
- [ ] Verify migration success
- [ ] Clean up backup data
- [ ] Plan v25 upgrade to upstream wasmd
- [ ] Archive migration code for reference

---

## Critical Summary: Mainnet v21→v24

### The Simplified Migration (IBCv2 Never Used)

**Simple detection and migration for all contracts:**
- **Detection**: Check if field 8 has ANY data (indicates SchemaBroken)
- **Pre-v20 contracts**: Already safe (field 8 empty, extension at field 7)
- **v20 & v21 contracts**: Need field swap (extension at field 8 → move to field 7)
- **Universal action**: Null field 8 for ALL contracts after migration

### Key Success Factors

1. **Parallel Processing**: 40+ workers for 1,200/sec throughput
2. **Simple Detection**: Just check field 8 for data
3. **Single Migration Path**: Swap if needed, null field 8
4. **Batch Operations**: 10,000 contracts per batch
5. **State Export Backup**: Before upgrade for safety
6. **Statistical Validation**: 0.1% sample (6,000 contracts)

### Risk Mitigation

- **Time**: Consider 3-hour window for safety margin
- **Memory**: Use streaming, never load all 6M contracts
- **Backup**: Full state export recommended
- **Testing**: Validate with mainnet data snapshot

### Expected Timeline (Mainnet - Simplified)

| Phase | Duration | Cumulative |
|-------|----------|------------|
| Preparation | 5 min | 5 min |
| Backup | 15-20 min | 25 min |
| Migration (6M) | 85-90 min | 115 min |
| Validation | 10 min | 125 min |
| Cleanup | 5 min | 130 min |
| **Total** | **~2h 10m** | **Well under 3h target** |

**Bottom Line**: The v24 upgrade will successfully migrate mainnet's 6M contracts using a simple field 8 check. Pre-v20 contracts are already safe and v20/v21 contracts get their fields swapped. All contracts end with null field 8.