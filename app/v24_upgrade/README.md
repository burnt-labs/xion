# V24 Upgrade: ContractInfo Migration

## Overview

The v24 upgrade migrates CosmWasm contracts to fix a protobuf field ordering bug introduced in wasmd v0.61.0-v0.61.4. This enables XION to use upstream wasmd without maintaining a fork.

**Key Points:**

- **Problem**: Extension field incorrectly stored at position 8 instead of 7
- **Solution**: Swap fields 7↔8 for affected contracts
- **Detection**: Check if field 8 has data (IBCv2 was never used)
- **Networks**: Mainnet (6M contracts), Testnet (500K contracts)

## Background: The Field Ordering Bug

### What Happened

Wasmd versions v0.61.0-v0.61.4 accidentally swapped the positions of `extension` and `ibc_port_id` fields in the ContractInfo protobuf structure:

```protobuf
// Correct Structure (pre-v20, v24 target)
message ContractInfo {
  // Fields 1-6...
  google.protobuf.Any extension = 7;
  string ibc2_port_id = 8;
}

// Broken Structure (v20-v21)
message ContractInfo {
  // Fields 1-6...
  string ibc_port_id = 7;
  google.protobuf.Any extension = 8;   // WRONG position
}
```

### Why It Matters

- Contracts created during v20-v21 have extension data at field 8
- XION never used IBCv2, so field 8 should always be empty
- Any data in field 8 indicates a contract needs migration

★ Insight ─────────────────────────────────────
**Wire Type Compatibility**

Both `string` and `google.protobuf.Any` use protobuf wire type 2 (length-delimited), which allowed the misplaced fields to work accidentally. The migration preserves wire types while correcting field positions.
─────────────────────────────────────────────────

## Migration Strategy

### Detection Algorithm

```go
// Simple detection: Check if field 8 has any data
func NeedsMigration(data []byte) bool {
    field8Value := GetFieldValue(data, 8)
    return field8Value != nil && len(field8Value) > 0
}
```

### Migration Process

1. **Detect**: Check if field 8 contains data
2. **Swap**: Move data from field 8 → field 7
3. **Clear**: Null field 8 (IBCv2 never used)
4. **Validate**: Verify extension at field 7

### Performance Requirements

| Network | Contracts | Time Limit | Rate Required | Workers |
|---------|-----------|------------|---------------|---------|
| Mainnet | 6,000,000 | 2 hours | 1,200/sec | 40 |
| Testnet | 500,000 | 2 hours | 70/sec | 10 |

## Implementation

### Core Files

| File | Purpose |
|------|---------|
| `handler.go` | Main upgrade handler |
| `detector.go` | Schema detection logic |
| `migrator.go` | Migration execution |
| `proto_parser.go` | Protobuf field manipulation |
| `validator.go` | Post-migration validation |
| `types.go` | Type definitions |
| `corrupted.go` | Corrupted contract utilities |
| `example.go` | Integration examples |

### Test Coverage

- **89.9%** coverage with comprehensive wire type testing
- Validates all schema migration paths
- Includes performance benchmarks for 6M contracts

## Deployment

### Pre-Upgrade Checklist

- [ ] Backup chain state (critical for mainnet's 6M contracts)
- [ ] Verify 2+ hour maintenance window
- [ ] Build v24 binary with migration code
- [ ] Test on local network with production data

### Upgrade Command

```bash
# Submit upgrade proposal
xiond tx gov submit-proposal software-upgrade v24 \
  --title "V24: ContractInfo Migration" \
  --description "Fix protobuf field ordering for all contracts" \
  --upgrade-height <TARGET_HEIGHT> \
  --deposit 10000000uxion \
  --from validator \
  --chain-id <CHAIN_ID>
```

### Expected Timeline

| Phase | Duration | Description |
|-------|----------|-------------|
| Preparation | 5 min | Count contracts, init workers |
| Backup | 15-20 min | State export (mainnet) |
| Migration | 85-90 min | Process all contracts |
| Validation | 10 min | Sample verification |
| Cleanup | 5 min | Report generation |
| **Total** | **~2 hours** | Under 3-hour safety margin |

## Monitoring

### Success Indicators

```log
INF Migration Report duration=1h45m total=6000000 migrated=1000000 skipped=5000000
INF All contracts validated successfully
INF v24 upgrade complete
```

### Error Handling

```log
ERR Failed to migrate contract address=xion1... error="proto: illegal wireType 7"
WRN Migration completed with errors failed=1 total=6000000
```

## Post-Upgrade

### Verification

```bash
# Check migration report
journalctl -u xiond | grep "Migration Report"

# Test random contract queries
xiond query wasm contract <ADDRESS>

# Verify no field 8 data remains
xiond query wasm list-contracts --limit 100
```

### Next Steps

After successful v24 migration:

- ✅ All contracts in canonical format
- ✅ Compatible with upstream wasmd v0.61.6+
- ✅ Can remove XION fork dependency
- ✅ Future upgrades use standard wasmd

## Technical Details

### Wire Type Analysis

The bug worked accidentally because both affected fields use wire type 2:

- `string ibc_port_id`: Wire type 2 (length-delimited)
- `google.protobuf.Any extension`: Wire type 2 (length-delimited)

This binary compatibility allowed misplaced fields to function until the schema was corrected.

### Solution Alignment

Our implementation:

1. Uses field 8 presence for detection (not block height)
2. Preserves wire types during field swap
3. Works universally across all contract creation times
4. Achieves 89.9% test coverage

## Troubleshooting

### Common Issues

#### Migration Timeout

- Increase workers (40+ for mainnet)
- Extend maintenance window to 3 hours
- Enable checkpoint/resume capability

#### Memory Exhaustion

- Use streaming iterators
- Process in 10K contract batches
- Add swap space if needed

#### Rollback Procedure

```bash
# Stop node
systemctl stop xiond

# Restore previous binary
cp xiond-v23 /usr/local/bin/xiond

# Restart with pre-upgrade state
systemctl start xiond
```

## References

- [Wasmd PR #2123](https://github.com/CosmWasm/wasmd/pull/2123) - Introduced bug
- [Wasmd PR #2390](https://github.com/CosmWasm/wasmd/pull/2390) - Fixed ordering
- [Protobuf Wire Format](https://protobuf.dev/programming-guides/encoding/)

## Support

- GitHub Issues: <https://github.com/burnt-labs/xion/issues>
- Discord: #validator-support channel
