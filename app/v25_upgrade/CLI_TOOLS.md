# V25 Migration CLI Tools

This document describes the CLI tools available for analyzing and testing the v25 contract migration.

## Commands Overview

### 1. `xiond analyze-contracts`

Analyzes all contracts in the database and reports corruption patterns.

**Purpose**: Understanding the scope of contract corruption before migration.

**Usage**:
```bash
xiond analyze-contracts [flags]
```

**Flags**:
- `--limit int` - Max contracts to analyze (0 = all, default: 0)
- `--show-examples` - Show example addresses for each category (default: true)
- `--test-fixes` - Test fixes on corrupted contracts to verify repairability (default: false)
- `--home string` - Node home directory (default: ~/.xiond)

**Example**:
```bash
# Analyze all contracts with fix testing
xiond analyze-contracts --test-fixes --home ~/.xiond

# Analyze first 1000 contracts only
xiond analyze-contracts --limit 1000

# Analyze without showing examples
xiond analyze-contracts --show-examples=false
```

**Output**:
- Total contract count
- State distribution (Healthy, SchemaInconsistent, UnmarshalFails, Unfixable)
- Corruption pattern breakdown (InvalidWireType, TruncatedField, etc.)
- Fix success rate (if --test-fixes enabled)
- Recommendations for migration

**When to use**:
- Before planning the v25 upgrade
- To understand how many contracts need fixing
- To validate that the fix logic works correctly

---

### 2. `xiond v25-dry-run`

Performs a dry-run of the v25 migration without modifying the database.

**Purpose**: Validating the migration logic on actual data before deploying.

**Usage**:
```bash
xiond v25-dry-run [flags]
```

**Flags**:
- `--verbose` - Show detailed information for each contract (default: false)
- `--limit int` - Max contracts to process (0 = all, default: 0)
- `--home string` - Node home directory (default: ~/.xiond)

**Example**:
```bash
# Run full dry-run migration
xiond v25-dry-run --home ~/.xiond

# Verbose output showing each contract
xiond v25-dry-run --verbose

# Test on first 100 contracts only
xiond v25-dry-run --limit 100
```

**Output**:
- Contract scan summary
- Working contracts count (will be skipped during migration)
- Corrupted contracts count (will be fixed)
- Fix success rate (should be 100%)
- Migration validation result
- Deployment readiness assessment

**When to use**:
- Before submitting governance proposal for v25 upgrade
- To verify migration will succeed on production data
- To confirm 100% fix success rate
- As final validation before deployment

---

## Usage Workflow

### Pre-Migration Phase

1. **Initial Analysis**:
   ```bash
   xiond analyze-contracts --test-fixes
   ```
   This gives you the full picture of contract health and validates fix logic.

2. **Review Results**:
   - Check total contract count
   - Verify corruption patterns are as expected (mostly InvalidWireType)
   - Confirm fix success rate is 100%

3. **Dry-Run Validation**:
   ```bash
   xiond v25-dry-run
   ```
   This simulates the actual migration and confirms readiness.

4. **Review Dry-Run**:
   - Verify all corrupted contracts are fixable
   - Confirm working contracts will be skipped
   - Check for any unfixable contracts (should be 0)

### Example Session

```bash
# Step 1: Analyze contracts
$ xiond analyze-contracts --test-fixes

Contract Corruption Analysis
=============================

Opening database at: /Users/greg/.xiond/data
✓ Database opened

Scanning contracts...
  Analyzed 1000 contracts...
  ...
  Analyzed 14000 contracts...

=================================
ANALYSIS REPORT
=================================

Total Contracts: 14,486

Contract State Distribution:
-----------------------------
  StateHealthy: 0 (0.00%)

  StateSchemaInconsistent: 13,380 (92.37%)

  StateUnmarshalFails: 1,106 (7.63%)

Corruption Pattern Distribution:
--------------------------------
  InvalidWireType: 1,106 (100.00%)

Fix Validation:
  ✓ Successfully fixable: 1,106 (100.0%)

=================================
RECOMMENDATIONS
=================================

⚠️  MIGRATION REQUIRED
------------------
1,106 contracts CANNOT unmarshal and must be fixed.

Corruption patterns:
  • Invalid wire type: 1,106 contracts (field swap from v20/v21)

→ Run 'xiond v25-dry-run' to validate the migration

# Step 2: Dry-run validation
$ xiond v25-dry-run

V25 Migration Dry Run
=====================

⚠️  DRY RUN MODE - No changes will be written to the database

Opening database at: /Users/greg/.xiond/data
✓ Database opened (read-only)

Scanning contracts...
  Processed 1000 contracts...
  ...
  Processed 14000 contracts...

=================================
DRY RUN REPORT
=================================

Total Contracts: 14,486

Working Contracts (can unmarshal): 13,380 (92.37%)
  ~ Schema inconsistent (functional): 13,380 (92.37%)

Corrupted Contracts (need fixing): 1,106 (7.63%)
  ✓ Fixable: 1,106 (100.00% of corrupted, 100.0% overall success rate)

=================================
MIGRATION VALIDATION
=================================

✅ ALL CORRUPTED CONTRACTS ARE FIXABLE!

Migration validation: SUCCESS
  • Total contracts: 14,486
  • Working contracts (will be skipped): 13,380 (92.37%)
  • Corrupted contracts (will be fixed): 1,106 (7.63%)
  • Fix success rate: 100.00%

✓ The v25 migration is READY to deploy
✓ All 1,106 corrupted contracts will be successfully fixed
✓ 13,380 working contracts will be left unchanged

Next steps:
1. Submit governance proposal for v25 upgrade
2. Migration will run automatically during chain upgrade
3. Monitor logs during upgrade to verify success

⚠️  Remember: This was a DRY RUN - no changes were made
The actual migration will run during the v25 chain upgrade
```

---

## Understanding the Output

### Contract States

1. **StateHealthy** (Canonical Schema)
   - Contract has all required fields including 7 and 8
   - Can unmarshal successfully
   - Schema is in canonical form
   - **Action**: None - leave as-is

2. **StateSchemaInconsistent** (Functional but Non-Canonical)
   - Missing fields 7 and/or 8
   - Can still unmarshal successfully (protobuf is lenient)
   - Contract is FUNCTIONAL
   - **Action**: Skip during migration (no fix needed)

3. **StateUnmarshalFails** (Corrupted)
   - Cannot unmarshal as ContractInfo
   - Chain cannot read contract metadata
   - **MUST BE FIXED**
   - **Action**: Apply fix during migration

4. **StateUnfixable**
   - Corruption is too severe to auto-fix
   - Manual intervention required
   - **Action**: Investigate, possibly delete

### Corruption Patterns

- **InvalidWireType**: Field 7/8 swap from v20/v21 bug (most common)
- **TruncatedField**: Incomplete field data
- **MalformedLength**: Invalid length delimiter
- **MissingRequiredFields**: Missing CodeID, Creator, etc.
- **DuplicateFields**: Same field number appears multiple times

---

## Safety Notes

- Both commands are **READ-ONLY** and safe to run on production data
- `analyze-contracts` only reads and reports
- `v25-dry-run` only reads and simulates (doesn't write)
- No changes are made to the database
- The actual migration runs during the v25 chain upgrade via governance proposal

---

## Troubleshooting

### "failed to open database"
- Check that `--home` points to correct node directory
- Verify database exists at `$HOME/data/application.db`
- Ensure you have read permissions

### "version mismatch"
- This is expected when testing on snapshot data
- Not an issue for production deployment
- Actual migration runs during chain upgrade with proper versioning

### "fix success rate < 100%"
- Investigate unfixable contracts
- Review corruption patterns
- May need manual intervention for some contracts
- Consider whether to proceed with partial migration

---

## Production Deployment

Once dry-run shows 100% success:

1. **Submit Governance Proposal**:
   - Propose v25 upgrade at specific block height
   - Include migration details in proposal

2. **Monitor During Upgrade**:
   - Watch for v25 migration logs
   - Verify fix counts match expectations
   - Confirm no errors

3. **Post-Upgrade Validation**:
   - Run `analyze-contracts` again
   - Verify all contracts can now unmarshal
   - Confirm StateUnmarshalFails count is 0

---

## Additional Resources

- Main README: `app/v25_upgrade/README.md`
- Migration validation: `V25_MIGRATION_VALIDATION.md`
- Test suite: `app/v25_upgrade/*_test.go`
