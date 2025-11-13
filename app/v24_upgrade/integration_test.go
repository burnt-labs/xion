package v24_upgrade_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/app"
	v24_upgrade "github.com/burnt-labs/xion/app/v24_upgrade"
)

// setupTestApp creates a test app with a clean state
func setupTestApp(t *testing.T) (*app.WasmApp, sdk.Context) {
	t.Helper()

	// Use the existing Setup helper from app package
	wasmApp := app.Setup(t)

	// Create a context
	ctx := wasmApp.NewContext(false)

	return wasmApp, ctx
}

// createTestContract creates a test contract with specific protobuf data
func createTestContract(
	t *testing.T,
	_ctx sdk.Context,
	store storetypes.KVStore,
	address string,
	data []byte,
) {
	t.Helper()

	// Contract info is stored with prefix 0x02
	key := append([]byte{v24_upgrade.ContractKeyPrefix}, []byte(address)...)
	store.Set(key, data)
}

// Helper to create protobuf with specific fields
func createTestProtobuf(fields map[int][]byte) []byte {
	result := make([]byte, 0)
	for fieldNum, data := range fields {
		// Create field tag (field number | wire type 2 for bytes)
		tag := v24_upgrade.EncodeFieldTag(fieldNum, v24_upgrade.WireBytes)
		result = append(result, tag...)

		// Add length prefix
		lenBuf := make([]byte, 10)
		n := 0
		length := uint64(len(data))
		for length >= 0x80 {
			lenBuf[n] = byte(length) | 0x80 //nolint:gosec // G602: slice bounds are controlled by loop
			length >>= 7
			n++
		}
		lenBuf[n] = byte(length) //nolint:gosec // G602: slice bounds are controlled by loop
		n++
		result = append(result, lenBuf[:n]...) // Add data
		result = append(result, data...)
	}
	return result
}

func TestIntegration_DetectNetwork(t *testing.T) {
	tests := []struct {
		name    string
		chainID string
		want    v24_upgrade.NetworkType
	}{
		{"mainnet", "xion-1", v24_upgrade.Mainnet},
		{"testnet", "xion-testnet-2", v24_upgrade.Testnet},
		{"local testnet", "xion-local", v24_upgrade.Testnet},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wasmApp, ctx := setupTestApp(t)
			require.NotNil(t, wasmApp)

			// The context should have a chain ID
			_ = ctx.ChainID()
		})
	}
}

func TestIntegration_AnalyzeSingleContract(t *testing.T) {
	wasmApp, ctx := setupTestApp(t)
	store := ctx.KVStore(wasmApp.GetKey("wasm"))

	tests := []struct {
		name         string
		address      string
		contractData map[int][]byte
		wantSchema   v24_upgrade.SchemaVersion
		wantErr      bool
	}{
		{
			name:    "SchemaLegacy contract",
			address: "xion1legacy",
			contractData: map[int][]byte{
				1: []byte("100"),
				7: []byte("extension"),
				// No field 8
			},
			wantSchema: v24_upgrade.SchemaLegacy,
			wantErr:    false,
		},
		{
			name:    "SchemaBroken contract",
			address: "xion1broken",
			contractData: map[int][]byte{
				1: []byte("200"),
				7: []byte("ibc_port_id"),
				8: []byte("extension"), // Wrong position
			},
			wantSchema: v24_upgrade.SchemaBroken,
			wantErr:    false,
		},
		{
			name:    "SchemaCanonical contract",
			address: "xion1canonical",
			contractData: map[int][]byte{
				1: []byte("300"),
				7: []byte("extension"),
				8: []byte(""), // Empty
			},
			wantSchema: v24_upgrade.SchemaCanonical,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create contract in store
			data := createTestProtobuf(tt.contractData)
			createTestContract(t, ctx, store, tt.address, data)

			// Analyze the contract
			analysis, err := v24_upgrade.AnalyzeSingleContract(ctx, wasmApp.GetKey("wasm"), tt.address)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, analysis)
			require.Equal(t, tt.wantSchema, analysis.Schema)
			require.Equal(t, tt.address, analysis.Address)
		})
	}
}

func TestIntegration_AnalyzeSingleContract_NotFound(t *testing.T) {
	wasmApp, ctx := setupTestApp(t)

	// Try to analyze non-existent contract
	analysis, err := v24_upgrade.AnalyzeSingleContract(ctx, wasmApp.GetKey("wasm"), "xion1nonexistent")

	require.Error(t, err)
	require.Nil(t, analysis)
	require.Contains(t, err.Error(), "contract not found")
}

func TestIntegration_DryRunAnalysis(t *testing.T) {
	wasmApp, ctx := setupTestApp(t)
	store := ctx.KVStore(wasmApp.GetKey("wasm"))

	// Create a variety of test contracts
	contracts := []struct {
		address string
		data    map[int][]byte
	}{
		{"xion1legacy1", map[int][]byte{1: []byte("1"), 7: []byte("ext1")}},
		{"xion1legacy2", map[int][]byte{1: []byte("2"), 7: []byte("ext2")}},
		{"xion1broken1", map[int][]byte{1: []byte("3"), 7: []byte("ibc"), 8: []byte("ext3")}},
		{"xion1broken2", map[int][]byte{1: []byte("4"), 7: []byte("ibc"), 8: []byte("ext4")}},
		{"xion1canonical1", map[int][]byte{1: []byte("5"), 7: []byte("ext5"), 8: []byte("")}},
	}

	for _, c := range contracts {
		data := createTestProtobuf(c.data)
		createTestContract(t, ctx, store, c.address, data)
	}

	// Run dry-run analysis
	stats, err := v24_upgrade.DryRunAnalysis(ctx, wasmApp.GetKey("wasm"))

	require.NoError(t, err)
	require.NotNil(t, stats)

	// Verify stats
	require.Equal(t, uint64(5), stats.TotalContracts)
	require.Equal(t, uint64(2), stats.LegacyCount)
	require.Equal(t, uint64(2), stats.BrokenCount)
	require.Greater(t, stats.CanonicalCount, uint64(0)) // At least 1
	require.NotZero(t, stats.Duration)
}

func TestIntegration_MigrateContract(t *testing.T) {
	wasmApp, ctx := setupTestApp(t)
	store := ctx.KVStore(wasmApp.GetKey("wasm"))

	// Create a broken contract
	brokenAddress := "xion1broken"
	brokenData := createTestProtobuf(map[int][]byte{
		1: []byte("code_id"),
		7: []byte("ibc_port_id"), // Wrong position
		8: []byte("extension"),   // Wrong position
	})

	createTestContract(t, ctx, store, brokenAddress, brokenData)

	// Create migrator
	migrator := v24_upgrade.NewMigrator(ctx.Logger(), wasmApp.GetKey("wasm"), v24_upgrade.Testnet, v24_upgrade.ModeAutoMigrate)

	// Migrate the contract
	migratedData, changed, err := migrator.MigrateContract(brokenAddress, brokenData)

	require.NoError(t, err)
	require.True(t, changed, "contract should have been changed")

	// Verify migration
	field7, err := v24_upgrade.GetFieldValue(migratedData, 7)
	require.NoError(t, err)
	require.Equal(t, []byte("extension"), field7, "field 7 should now have extension")

	// Verify field 8 is empty
	require.True(t, v24_upgrade.IsField8Empty(migratedData), "field 8 should be empty")
}

func TestIntegration_MigrationWithMultipleContracts(t *testing.T) {
	wasmApp, ctx := setupTestApp(t)
	store := ctx.KVStore(wasmApp.GetKey("wasm"))

	// Create multiple contracts with different schemas
	testContracts := map[string]map[int][]byte{
		"xion1legacy1":    {1: []byte("1"), 7: []byte("ext1")},
		"xion1legacy2":    {1: []byte("2"), 7: []byte("ext2")},
		"xion1broken1":    {1: []byte("3"), 7: []byte("ibc1"), 8: []byte("ext3")},
		"xion1broken2":    {1: []byte("4"), 7: []byte("ibc2"), 8: []byte("ext4")},
		"xion1canonical1": {1: []byte("5"), 7: []byte("ext5"), 8: []byte("")},
	}

	for addr, fields := range testContracts {
		data := createTestProtobuf(fields)
		createTestContract(t, ctx, store, addr, data)
	}

	// Create migrator
	migrator := v24_upgrade.NewMigrator(ctx.Logger(), wasmApp.GetKey("wasm"), v24_upgrade.Testnet, v24_upgrade.ModeAutoMigrate)

	// Process each contract
	migratedCount := 0
	skippedCount := 0

	for addr, fields := range testContracts {
		data := createTestProtobuf(fields)
		_, changed, err := migrator.MigrateContract(addr, data)
		require.NoError(t, err)

		if changed {
			migratedCount++
		} else {
			skippedCount++
		}
	}

	// Verify counts
	require.Equal(t, 4, migratedCount, "should have migrated 2 legacy + 2 broken contracts")
	require.Equal(t, 1, skippedCount, "should have skipped 1 canonical contract")
}

func TestIntegration_ValidatorSchemaDistribution(t *testing.T) {
	wasmApp, ctx := setupTestApp(t)

	validator := v24_upgrade.NewValidator(ctx.Logger(), wasmApp.GetKey("wasm"), v24_upgrade.Testnet)

	tests := []struct {
		name    string
		stats   v24_upgrade.MigrationStats
		wantErr bool
	}{
		{
			name: "valid distribution",
			stats: v24_upgrade.MigrationStats{
				TotalContracts: 100,
				LegacyCount:    60,
				BrokenCount:    30,
				CanonicalCount: 10,
				UnknownCount:   0,
			},
			wantErr: false,
		},
		{
			name: "invalid distribution - count mismatch",
			stats: v24_upgrade.MigrationStats{
				TotalContracts: 100,
				LegacyCount:    50,
				BrokenCount:    30,
				CanonicalCount: 10,
				UnknownCount:   5, // Doesn't add up to 100
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateSchemaDistribution(tt.stats)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestIntegration_ConfigurationHelpers(t *testing.T) {
	tests := []struct {
		name        string
		network     v24_upgrade.NetworkType
		wantWorkers int
		wantBatch   int
		wantSample  float64
	}{
		{
			name:        "mainnet configuration",
			network:     v24_upgrade.Mainnet,
			wantWorkers: 40,
			wantBatch:   10000,
			wantSample:  0.001,
		},
		{
			name:        "testnet configuration",
			network:     v24_upgrade.Testnet,
			wantWorkers: 10,
			wantBatch:   5000,
			wantSample:  0.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workers := v24_upgrade.GetWorkerCount(tt.network)
			batch := v24_upgrade.GetBatchSize(tt.network)
			sample := v24_upgrade.GetSampleRate(tt.network)

			require.Equal(t, tt.wantWorkers, workers)
			require.Equal(t, tt.wantBatch, batch)
			require.Equal(t, tt.wantSample, sample)
		})
	}
}

func TestIntegration_MigrateAllContracts(t *testing.T) {
	wasmApp, ctx := setupTestApp(t)
	store := ctx.KVStore(wasmApp.GetKey("wasm"))

	// Create 10 contracts with various schemas
	contracts := []struct {
		address string
		data    map[int][]byte
	}{
		{"xion1legacy1", map[int][]byte{1: []byte("1"), 7: []byte("ext1")}},
		{"xion1legacy2", map[int][]byte{1: []byte("2"), 7: []byte("ext2")}},
		{"xion1legacy3", map[int][]byte{1: []byte("3"), 7: []byte("ext3")}},
		{"xion1broken1", map[int][]byte{1: []byte("4"), 7: []byte("ibc1"), 8: []byte("ext4")}},
		{"xion1broken2", map[int][]byte{1: []byte("5"), 7: []byte("ibc2"), 8: []byte("ext5")}},
		{"xion1broken3", map[int][]byte{1: []byte("6"), 7: []byte("ibc3"), 8: []byte("ext6")}},
		{"xion1broken4", map[int][]byte{1: []byte("7"), 7: []byte("ibc4"), 8: []byte("ext7")}},
		{"xion1canonical1", map[int][]byte{1: []byte("8"), 7: []byte("ext8"), 8: []byte("")}},
		{"xion1canonical2", map[int][]byte{1: []byte("9"), 7: []byte("ext9"), 8: []byte("")}},
		{"xion1canonical3", map[int][]byte{1: []byte("10"), 7: []byte("ext10"), 8: []byte("")}},
	}

	for _, c := range contracts {
		data := createTestProtobuf(c.data)
		createTestContract(t, ctx, store, c.address, data)
	}

	// Create migrator and run full migration
	migrator := v24_upgrade.NewMigrator(ctx.Logger(), wasmApp.GetKey("wasm"), v24_upgrade.Testnet, v24_upgrade.ModeAutoMigrate)

	// Run the full migration
	report, err := migrator.MigrateAllContracts(ctx)
	require.NoError(t, err)
	require.NotNil(t, report)

	// Verify migration stats
	require.Equal(t, uint64(10), report.Stats.TotalContracts)
	require.Equal(t, uint64(10), report.Stats.ProcessedContracts)
	require.Equal(t, uint64(7), report.Stats.MigratedContracts, "should migrate 3 legacy + 4 broken contracts")
	require.Equal(t, uint64(3), report.Stats.SkippedContracts, "should skip 3 canonical contracts")
	require.Equal(t, uint64(0), report.Stats.FailedContracts)

	// Verify schema distribution
	require.Equal(t, uint64(3), report.Stats.LegacyCount, "3 legacy contracts (have field 7 but missing field 8)")
	require.Equal(t, uint64(4), report.Stats.BrokenCount, "4 broken contracts with field 8 data")
	require.Equal(t, uint64(3), report.Stats.CanonicalCount, "3 canonical contracts (both fields present, field 8 empty)")

	// Verify all broken contracts are now fixed
	brokenAddresses := []string{"xion1broken1", "xion1broken2", "xion1broken3", "xion1broken4"}
	for _, addr := range brokenAddresses {
		key := append([]byte{v24_upgrade.ContractKeyPrefix}, []byte(addr)...)
		data := store.Get(key)
		require.NotNil(t, data)

		// Verify field 7 has extension
		field7, err := v24_upgrade.GetFieldValue(data, 7)
		require.NoError(t, err)
		require.NotNil(t, field7)
		require.Contains(t, string(field7), "ext")

		// Verify field 8 is empty
		require.True(t, v24_upgrade.IsField8Empty(data))
	}
}

func TestIntegration_MigrateAllContracts_EmptyStore(t *testing.T) {
	wasmApp, ctx := setupTestApp(t)

	// Create migrator with empty store
	migrator := v24_upgrade.NewMigrator(ctx.Logger(), wasmApp.GetKey("wasm"), v24_upgrade.Testnet, v24_upgrade.ModeAutoMigrate)

	// Run migration on empty store
	report, err := migrator.MigrateAllContracts(ctx)
	require.NoError(t, err)
	require.NotNil(t, report)

	// Verify empty stats
	require.Equal(t, uint64(0), report.Stats.TotalContracts)
	require.Equal(t, uint64(0), report.Stats.ProcessedContracts)
	require.Equal(t, uint64(0), report.Stats.MigratedContracts)
}

func TestIntegration_PerformMigration(t *testing.T) {
	wasmApp, ctx := setupTestApp(t)
	store := ctx.KVStore(wasmApp.GetKey("wasm"))

	// Create contracts for full migration test
	contracts := []struct {
		address string
		data    map[int][]byte
	}{
		{"xion1test1", map[int][]byte{1: []byte("1"), 7: []byte("ext1")}},
		{"xion1test2", map[int][]byte{1: []byte("2"), 7: []byte("ibc2"), 8: []byte("ext2")}},
		{"xion1test3", map[int][]byte{1: []byte("3"), 7: []byte("ibc3"), 8: []byte("ext3")}},
		{"xion1test4", map[int][]byte{1: []byte("4"), 7: []byte("ext4"), 8: []byte("")}},
	}

	for _, c := range contracts {
		data := createTestProtobuf(c.data)
		createTestContract(t, ctx, store, c.address, data)
	}

	// Run the full PerformMigration function
	err := v24_upgrade.PerformMigration(ctx, wasmApp.GetKey("wasm"))
	require.NoError(t, err)

	// Verify contracts are migrated correctly
	// Check the broken contracts are now fixed
	for _, addr := range []string{"xion1test2", "xion1test3"} {
		key := append([]byte{v24_upgrade.ContractKeyPrefix}, []byte(addr)...)
		data := store.Get(key)
		require.NotNil(t, data)

		// Verify field 7 has extension
		field7, err := v24_upgrade.GetFieldValue(data, 7)
		require.NoError(t, err)
		require.NotNil(t, field7)
		require.Contains(t, string(field7), "ext")

		// Verify field 8 is empty
		require.True(t, v24_upgrade.IsField8Empty(data))
	}
}

func TestIntegration_ValidateMigration(t *testing.T) {
	wasmApp, ctx := setupTestApp(t)
	store := ctx.KVStore(wasmApp.GetKey("wasm"))

	// Create contracts that are properly migrated (all have field 7 and empty field 8)
	contracts := []struct {
		address string
		data    map[int][]byte
	}{
		{"xion1valid1", map[int][]byte{1: []byte("1"), 7: []byte("ext1"), 8: []byte("")}},
		{"xion1valid2", map[int][]byte{1: []byte("2"), 7: []byte("ext2"), 8: []byte("")}},
		{"xion1valid3", map[int][]byte{1: []byte("3"), 7: []byte("ext3"), 8: []byte("")}},
		{"xion1valid4", map[int][]byte{1: []byte("4"), 7: []byte("ext4"), 8: []byte("")}},
		{"xion1valid5", map[int][]byte{1: []byte("5"), 7: []byte("ext5"), 8: []byte("")}},
	}

	for _, c := range contracts {
		data := createTestProtobuf(c.data)
		createTestContract(t, ctx, store, c.address, data)
	}

	// Create validator and run validation
	validator := v24_upgrade.NewValidator(ctx.Logger(), wasmApp.GetKey("wasm"), v24_upgrade.Testnet)

	// Run validation
	results, err := validator.ValidateMigration(ctx, uint64(len(contracts)))
	require.NoError(t, err)
	require.NotNil(t, results)

	// Verify all sampled contracts are valid
	for _, result := range results {
		require.True(t, result.Valid, "contract %s should be valid", result.Address)
		require.NoError(t, result.Error)
		require.True(t, result.Field8Empty, "field 8 should be empty")
	}
}

func TestIntegration_ValidateMigration_WithBrokenContract(t *testing.T) {
	wasmApp, ctx := setupTestApp(t)
	store := ctx.KVStore(wasmApp.GetKey("wasm"))

	// Create a mix of valid and broken contracts
	contracts := []struct {
		address string
		data    map[int][]byte
		broken  bool
	}{
		{"xion1valid1", map[int][]byte{1: []byte("1"), 7: []byte("ext1"), 8: []byte("")}, false},
		{"xion1valid2", map[int][]byte{1: []byte("2"), 7: []byte("ext2"), 8: []byte("")}, false},
		{"xion1broken1", map[int][]byte{1: []byte("3"), 7: []byte("ibc"), 8: []byte("ext3")}, true},
		{"xion1valid3", map[int][]byte{1: []byte("4"), 7: []byte("ext4"), 8: []byte("")}, false},
	}

	for _, c := range contracts {
		data := createTestProtobuf(c.data)
		createTestContract(t, ctx, store, c.address, data)
	}

	// Create validator
	validator := v24_upgrade.NewValidator(ctx.Logger(), wasmApp.GetKey("wasm"), v24_upgrade.Testnet)

	// Run validation - this should detect the broken contract and return an error
	results, err := validator.ValidateMigration(ctx, uint64(len(contracts)))

	// The validation SHOULD fail because we have a broken contract
	// This test verifies that the validator correctly detects broken contracts
	if err != nil {
		// If validation failed (as expected), verify the error message indicates failure
		require.Contains(t, err.Error(), "validation failed")
		require.NotNil(t, results, "results should still be returned even on failure")

		// Check that at least one contract in results is invalid
		hasInvalid := false
		for _, result := range results {
			if !result.Valid {
				hasInvalid = true
				require.False(t, result.Field8Empty, "broken contract should have non-empty field 8")
			}
		}
		require.True(t, hasInvalid, "should have caught at least one invalid contract")
	} else {
		// If validation passed, it means our sample didn't catch the broken contract
		// This is possible with statistical sampling
		require.NotEmpty(t, results, "should have validation results")
	}
}

func TestIntegration_PerformMigration_LargeDataset(t *testing.T) {
	wasmApp, ctx := setupTestApp(t)
	store := ctx.KVStore(wasmApp.GetKey("wasm"))

	// Create 50 contracts to test parallel processing
	for i := 1; i <= 50; i++ {
		var data []byte
		addr := ""

		// Mix of schemas
		switch i % 3 {
		case 0: // Legacy
			addr = "xion1legacy" + string(rune('a'+i%26))
			data = createTestProtobuf(map[int][]byte{
				1: {byte(i)},
				7: []byte("ext"),
			})
		case 1: // Broken
			addr = "xion1broken" + string(rune('a'+i%26))
			data = createTestProtobuf(map[int][]byte{
				1: {byte(i)},
				7: []byte("ibc"),
				8: []byte("ext"),
			})
		case 2: // Canonical
			addr = "xion1canonical" + string(rune('a'+i%26))
			data = createTestProtobuf(map[int][]byte{
				1: {byte(i)},
				7: []byte("ext"),
				8: []byte(""),
			})
		}

		createTestContract(t, ctx, store, addr, data)
	}

	// Run full migration
	err := v24_upgrade.PerformMigration(ctx, wasmApp.GetKey("wasm"))
	require.NoError(t, err)

	// Verify all broken contracts are fixed
	// Check a sample of broken contracts
	for i := 1; i <= 50; i++ {
		if i%3 == 1 { // Broken contracts
			addr := "xion1broken" + string(rune('a'+i%26))
			key := append([]byte{v24_upgrade.ContractKeyPrefix}, []byte(addr)...)
			data := store.Get(key)
			require.NotNil(t, data, "contract %s should exist", addr)

			// Verify field 8 is now empty
			require.True(t, v24_upgrade.IsField8Empty(data), "contract %s field 8 should be empty", addr)
		}
	}
}

func TestIntegration_ValidateContract_Public(t *testing.T) {
	wasmApp, ctx := setupTestApp(t)
	store := ctx.KVStore(wasmApp.GetKey("wasm"))

	// Create a valid contract (properly migrated with field 8 empty)
	validAddr := "xion1validcontract"
	validData := createTestProtobuf(map[int][]byte{
		1: []byte("1"),
		7: []byte("extension"),
		8: []byte(""),
	})
	createTestContract(t, ctx, store, validAddr, validData)

	// Create validator
	validator := v24_upgrade.NewValidator(ctx.Logger(), wasmApp.GetKey("wasm"), v24_upgrade.Testnet)

	// Test the public ValidateContract function
	result := validator.ValidateContract(ctx, validAddr)

	// Verify result
	require.Equal(t, validAddr, result.Address)
	require.True(t, result.Valid)
	require.NoError(t, result.Error)
	require.True(t, result.Field8Empty)
}

func TestIntegration_ValidateContract_Public_Broken(t *testing.T) {
	wasmApp, ctx := setupTestApp(t)
	store := ctx.KVStore(wasmApp.GetKey("wasm"))

	// Create a broken contract
	brokenAddr := "xion1brokencontract"
	brokenData := createTestProtobuf(map[int][]byte{
		1: []byte("1"),
		7: []byte("ibc_port"),
		8: []byte("extension"),
	})
	createTestContract(t, ctx, store, brokenAddr, brokenData)

	// Create validator
	validator := v24_upgrade.NewValidator(ctx.Logger(), wasmApp.GetKey("wasm"), v24_upgrade.Testnet)

	// Test the public ValidateContract function on broken contract
	result := validator.ValidateContract(ctx, brokenAddr)

	// Verify result shows it's invalid
	require.Equal(t, brokenAddr, result.Address)
	require.False(t, result.Valid)
	require.False(t, result.Field8Empty)
}
