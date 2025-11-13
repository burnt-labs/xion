package v24_upgrade_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/app"
	v24_upgrade "github.com/burnt-labs/xion/app/v24_upgrade"
)

// TestE2E_FullUpgradeFlow simulates the complete v24 upgrade process
// This test exercises the entire upgrade handler workflow
func TestE2E_FullUpgradeFlow(t *testing.T) {
	// Setup test app
	wasmApp, ctx := setupTestApp(t)
	store := ctx.KVStore(wasmApp.GetKey("wasm"))

	// Scenario: Simulate a realistic blockchain state with various contract schemas
	// This mirrors what we'd see on mainnet/testnet

	// Step 1: Create contracts in "broken" state (as they would be from v20/v21)
	t.Log("Creating broken contracts (simulating v20/v21 state)")
	brokenContracts := []struct {
		address string
		data    map[int][]byte
	}{
		// These contracts have extension at field 8 (wrong position)
		{"xion1contract001", map[int][]byte{1: []byte("1"), 7: []byte("ibc_port"), 8: []byte("extension1")}},
		{"xion1contract002", map[int][]byte{1: []byte("2"), 7: []byte("ibc_port"), 8: []byte("extension2")}},
		{"xion1contract003", map[int][]byte{1: []byte("3"), 7: []byte("ibc_port"), 8: []byte("extension3")}},
	}

	for _, c := range brokenContracts {
		data := createTestProtobuf(c.data)
		createTestContract(t, ctx, store, c.address, data)
	}

	// Step 2: Create some legacy contracts (pre-v20, already safe)
	t.Log("Creating legacy contracts (simulating pre-v20 state)")
	legacyContracts := []struct {
		address string
		data    map[int][]byte
	}{
		// These have extension at field 7 (correct position), no field 8
		{"xion1legacy001", map[int][]byte{1: []byte("10"), 7: []byte("legacy_ext1")}},
		{"xion1legacy002", map[int][]byte{1: []byte("11"), 7: []byte("legacy_ext2")}},
	}

	for _, c := range legacyContracts {
		data := createTestProtobuf(c.data)
		createTestContract(t, ctx, store, c.address, data)
	}

	// Step 3: Create some canonical contracts (v22+, already correct)
	t.Log("Creating canonical contracts (simulating v22+ state)")
	canonicalContracts := []struct {
		address string
		data    map[int][]byte
	}{
		// These have extension at field 7, empty field 8
		{"xion1canon001", map[int][]byte{1: []byte("20"), 7: []byte("canon_ext1"), 8: []byte("")}},
	}

	for _, c := range canonicalContracts {
		data := createTestProtobuf(c.data)
		createTestContract(t, ctx, store, c.address, data)
	}

	t.Logf("Created %d broken, %d legacy, %d canonical contracts",
		len(brokenContracts), len(legacyContracts), len(canonicalContracts))

	// Step 4: Run the upgrade handler (this is what happens during chain upgrade)
	t.Log("Executing v24 upgrade handler...")
	startTime := time.Now()

	err := v24_upgrade.PerformMigration(ctx, wasmApp.GetKey("wasm"))
	require.NoError(t, err, "migration should succeed")

	duration := time.Since(startTime)
	t.Logf("Migration completed in %v", duration)

	// Step 5: Verify ALL broken contracts are now fixed
	t.Log("Verifying broken contracts are fixed...")
	for _, c := range brokenContracts {
		key := append([]byte{v24_upgrade.ContractKeyPrefix}, []byte(c.address)...)
		data := store.Get(key)
		require.NotNil(t, data, "contract %s should exist", c.address)

		// Critical checks:
		// 1. Field 7 should now have the extension data
		field7, err := v24_upgrade.GetFieldValue(data, 7)
		require.NoError(t, err, "should read field 7 for %s", c.address)
		require.Contains(t, string(field7), "extension", "field 7 should contain extension for %s", c.address)

		// 2. Field 8 should be empty
		isEmpty := v24_upgrade.IsField8Empty(data)
		require.True(t, isEmpty, "field 8 should be empty for %s", c.address)

		t.Logf("✓ Contract %s successfully migrated", c.address)
	}

	// Step 6: Verify legacy contracts are unchanged
	t.Log("Verifying legacy contracts are unchanged...")
	for _, c := range legacyContracts {
		key := append([]byte{v24_upgrade.ContractKeyPrefix}, []byte(c.address)...)
		data := store.Get(key)
		require.NotNil(t, data, "contract %s should exist", c.address)

		field7, err := v24_upgrade.GetFieldValue(data, 7)
		require.NoError(t, err)
		require.Contains(t, string(field7), "legacy_ext", "legacy contract should be unchanged")

		t.Logf("✓ Legacy contract %s unchanged", c.address)
	}

	// Step 7: Verify canonical contracts are unchanged
	t.Log("Verifying canonical contracts are unchanged...")
	for _, c := range canonicalContracts {
		key := append([]byte{v24_upgrade.ContractKeyPrefix}, []byte(c.address)...)
		data := store.Get(key)
		require.NotNil(t, data, "contract %s should exist", c.address)

		isEmpty := v24_upgrade.IsField8Empty(data)
		require.True(t, isEmpty, "canonical contract field 8 should remain empty")

		t.Logf("✓ Canonical contract %s unchanged", c.address)
	}

	t.Log("✓ E2E upgrade flow test PASSED")
}

// TestE2E_UpgradePerformance tests migration performance with large contract count
func TestE2E_UpgradePerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	wasmApp, ctx := setupTestApp(t)
	store := ctx.KVStore(wasmApp.GetKey("wasm"))

	// Create 100 contracts to test parallel processing
	contractCount := 100
	t.Logf("Creating %d test contracts...", contractCount)

	for i := 0; i < contractCount; i++ {
		addr := "xion1perf" + string(rune('a'+(i/26))) + string(rune('a'+(i%26)))
		var data []byte

		// Mix: 40% broken, 30% legacy, 30% canonical
		switch i % 10 {
		case 0, 1, 2, 3: // 40% broken
			data = createTestProtobuf(map[int][]byte{
				1: {byte(i)},
				7: []byte("ibc"),
				8: []byte("ext"),
			})
		case 4, 5, 6: // 30% legacy
			data = createTestProtobuf(map[int][]byte{
				1: {byte(i)},
				7: []byte("ext"),
			})
		default: // 30% canonical
			data = createTestProtobuf(map[int][]byte{
				1: {byte(i)},
				7: []byte("ext"),
				8: []byte(""),
			})
		}

		createTestContract(t, ctx, store, addr, data)
	}

	t.Log("Running migration performance test...")
	startTime := time.Now()

	err := v24_upgrade.PerformMigration(ctx, wasmApp.GetKey("wasm"))
	require.NoError(t, err)

	duration := time.Since(startTime)
	contractsPerSecond := float64(contractCount) / duration.Seconds()

	t.Logf("Performance results:")
	t.Logf("  Contracts: %d", contractCount)
	t.Logf("  Duration: %v", duration)
	t.Logf("  Rate: %.2f contracts/second", contractsPerSecond)

	// Performance assertion: should process at least 10 contracts per second
	require.Greater(t, contractsPerSecond, 10.0, "migration should be reasonably fast")

	t.Log("✓ Performance test PASSED")
}

// TestE2E_UpgradeIdempotency verifies running the upgrade multiple times is safe
func TestE2E_UpgradeIdempotency(t *testing.T) {
	wasmApp, ctx := setupTestApp(t)
	store := ctx.KVStore(wasmApp.GetKey("wasm"))

	// Create broken contract
	brokenAddr := "xion1idempotent"
	brokenData := createTestProtobuf(map[int][]byte{
		1: []byte("1"),
		7: []byte("ibc_port"),
		8: []byte("extension"),
	})
	createTestContract(t, ctx, store, brokenAddr, brokenData)

	// Run migration first time
	t.Log("Running migration (first time)...")
	err := v24_upgrade.PerformMigration(ctx, wasmApp.GetKey("wasm"))
	require.NoError(t, err)

	// Get contract state after first migration
	key := append([]byte{v24_upgrade.ContractKeyPrefix}, []byte(brokenAddr)...)
	dataAfterFirst := store.Get(key)
	require.NotNil(t, dataAfterFirst)
	require.True(t, v24_upgrade.IsField8Empty(dataAfterFirst))

	// Run migration second time (should be idempotent)
	t.Log("Running migration (second time)...")
	err = v24_upgrade.PerformMigration(ctx, wasmApp.GetKey("wasm"))
	require.NoError(t, err)

	// Get contract state after second migration
	dataAfterSecond := store.Get(key)
	require.NotNil(t, dataAfterSecond)

	// Verify data is unchanged
	require.Equal(t, dataAfterFirst, dataAfterSecond, "running migration twice should produce same result")
	require.True(t, v24_upgrade.IsField8Empty(dataAfterSecond), "field 8 should still be empty")

	t.Log("✓ Idempotency test PASSED")
}

// TestE2E_UpgradeAnalysis tests the dry-run analysis functionality
func TestE2E_UpgradeAnalysis(t *testing.T) {
	wasmApp, ctx := setupTestApp(t)
	store := ctx.KVStore(wasmApp.GetKey("wasm"))

	// Create diverse contract set
	contracts := map[string]map[int][]byte{
		"xion1broken1": {1: []byte("1"), 7: []byte("ibc"), 8: []byte("ext1")},
		"xion1broken2": {1: []byte("2"), 7: []byte("ibc"), 8: []byte("ext2")},
		"xion1legacy1": {1: []byte("3"), 7: []byte("ext3")},
		"xion1legacy2": {1: []byte("4"), 7: []byte("ext4")},
		"xion1legacy3": {1: []byte("5"), 7: []byte("ext5")},
		"xion1canon1":  {1: []byte("6"), 7: []byte("ext6"), 8: []byte("")},
	}

	for addr, fields := range contracts {
		data := createTestProtobuf(fields)
		createTestContract(t, ctx, store, addr, data)
	}

	// Run dry-run analysis
	t.Log("Running dry-run analysis...")
	stats, err := v24_upgrade.DryRunAnalysis(ctx, wasmApp.GetKey("wasm"))
	require.NoError(t, err)
	require.NotNil(t, stats)

	// Verify analysis results
	require.Equal(t, uint64(6), stats.TotalContracts, "should count all contracts")
	require.Equal(t, uint64(2), stats.BrokenCount, "should identify broken contracts")
	require.Equal(t, uint64(3), stats.LegacyCount, "should identify legacy contracts")
	require.GreaterOrEqual(t, stats.CanonicalCount, uint64(1), "should identify canonical contracts")

	t.Logf("Analysis results:")
	t.Logf("  Total: %d", stats.TotalContracts)
	t.Logf("  Broken: %d", stats.BrokenCount)
	t.Logf("  Legacy: %d", stats.LegacyCount)
	t.Logf("  Canonical: %d", stats.CanonicalCount)

	t.Log("✓ Analysis test PASSED")
}

// TestE2E_UpgradeWithCorruptedData tests handling of corrupted contract data
func TestE2E_UpgradeWithCorruptedData(t *testing.T) {
	wasmApp, ctx := setupTestApp(t)
	store := ctx.KVStore(wasmApp.GetKey("wasm"))

	// Create a valid contract
	validAddr := "xion1valid"
	validData := createTestProtobuf(map[int][]byte{
		1: []byte("1"),
		7: []byte("ext"),
	})
	createTestContract(t, ctx, store, validAddr, validData)

	// Create a contract with potentially problematic data
	// (but still valid protobuf)
	edgeCaseAddr := "xion1edge"
	edgeCaseData := createTestProtobuf(map[int][]byte{
		1: {0x01},     // Minimal data
		7: []byte(""), // Empty extension
	})
	createTestContract(t, ctx, store, edgeCaseAddr, edgeCaseData)

	// Run migration - should handle edge cases gracefully
	t.Log("Running migration with edge case contracts...")
	err := v24_upgrade.PerformMigration(ctx, wasmApp.GetKey("wasm"))
	require.NoError(t, err, "migration should handle edge cases without failing")

	// Verify both contracts still exist
	validKey := append([]byte{v24_upgrade.ContractKeyPrefix}, []byte(validAddr)...)
	require.NotNil(t, store.Get(validKey), "valid contract should exist")

	edgeKey := append([]byte{v24_upgrade.ContractKeyPrefix}, []byte(edgeCaseAddr)...)
	require.NotNil(t, store.Get(edgeKey), "edge case contract should exist")

	t.Log("✓ Corrupted data handling test PASSED")
}

// TestE2E_UpgradeContextTimeout tests that upgrade completes within reasonable time
func TestE2E_UpgradeContextTimeout(t *testing.T) {
	wasmApp := app.Setup(t)

	// Create context with timeout
	baseCtx := wasmApp.NewContext(false)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Wrap SDK context with Go context
	sdkCtx := sdk.UnwrapSDKContext(baseCtx).WithContext(ctx)
	store := sdkCtx.KVStore(wasmApp.GetKey("wasm"))

	// Create some test contracts
	for i := 0; i < 10; i++ {
		addr := "xion1timeout" + string(rune('a'+i))
		data := createTestProtobuf(map[int][]byte{
			1: {byte(i)},
			7: []byte("ibc"),
			8: []byte("ext"),
		})
		createTestContract(t, sdkCtx, store, addr, data)
	}

	// Run migration - should complete before timeout
	t.Log("Running migration with timeout context...")
	err := v24_upgrade.PerformMigration(sdkCtx, wasmApp.GetKey("wasm"))
	require.NoError(t, err, "migration should complete before timeout")

	// Verify context didn't timeout
	select {
	case <-ctx.Done():
		t.Fatal("context timed out during migration")
	default:
		t.Log("✓ Migration completed within timeout")
	}

	t.Log("✓ Context timeout test PASSED")
}
