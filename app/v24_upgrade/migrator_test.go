package v24_upgrade

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"
)

func TestMigratorMigrateContract(t *testing.T) {
	logger := log.NewNopLogger()
	migrator := NewMigrator(logger, nil, Mainnet, ModeAutoMigrate)

	tests := []struct {
		name            string
		address         string
		input           map[int][]byte
		wantChanged     bool
		wantField7Data  []byte
		wantField8Empty bool
	}{
		{
			name:    "SchemaLegacy - no changes needed",
			address: "xion1legacy",
			input: map[int][]byte{
				1: []byte("100"),
				7: []byte("extension"),
				// No field 8
			},
			wantChanged:     false,
			wantField7Data:  []byte("extension"),
			wantField8Empty: true,
		},
		{
			name:    "SchemaBroken - needs migration",
			address: "xion1broken",
			input: map[int][]byte{
				1: []byte("200"),
				7: []byte("ibc_port_id"),
				8: []byte("extension"),
			},
			wantChanged:     true,
			wantField7Data:  []byte("extension"),
			wantField8Empty: true,
		},
		{
			name:    "SchemaCanonical - no changes needed",
			address: "xion1canonical",
			input: map[int][]byte{
				1: []byte("300"),
				7: []byte("extension"),
				8: []byte(""),
			},
			wantChanged:     false,
			wantField7Data:  []byte("extension"),
			wantField8Empty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := createTestProtobuf(tt.input)

			result, changed, err := migrator.MigrateContract(tt.address, input)

			require.NoError(t, err)
			require.Equal(t, tt.wantChanged, changed)

			// Verify field 7
			field7, err := GetFieldValue(result, 7)
			require.NoError(t, err)
			require.Equal(t, tt.wantField7Data, field7)

			// Verify field 8 is empty
			require.Equal(t, tt.wantField8Empty, IsField8Empty(result))
		})
	}
}

func TestMigratorSchemaCount(t *testing.T) {
	logger := log.NewNopLogger()
	migrator := NewMigrator(logger, nil, Testnet, ModeAutoMigrate)

	// Simulate processing various schemas
	migrator.updateSchemaCount(SchemaLegacy)
	migrator.updateSchemaCount(SchemaLegacy)
	migrator.updateSchemaCount(SchemaBroken)
	migrator.updateSchemaCount(SchemaCanonical)
	migrator.updateSchemaCount(SchemaUnknown)

	require.Equal(t, uint64(2), migrator.stats.LegacyCount)
	require.Equal(t, uint64(1), migrator.stats.BrokenCount)
	require.Equal(t, uint64(1), migrator.stats.CanonicalCount)
	require.Equal(t, uint64(1), migrator.stats.UnknownCount)
}

func TestNewMigrator(t *testing.T) {
	tests := []struct {
		name    string
		network NetworkType
		mode    MigrationMode
	}{
		{
			name:    "mainnet auto-migrate",
			network: Mainnet,
			mode:    ModeAutoMigrate,
		},
		{
			name:    "testnet fail-on-corruption",
			network: Testnet,
			mode:    ModeFailOnCorruption,
		},
		{
			name:    "testnet log-and-continue",
			network: Testnet,
			mode:    ModeLogAndContinue,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := log.NewNopLogger()
			migrator := NewMigrator(logger, nil, tt.network, tt.mode)

			require.NotNil(t, migrator)
			require.Equal(t, tt.network, migrator.network)
			require.Equal(t, tt.mode, migrator.mode)
			require.NotNil(t, migrator.stats)
			require.NotNil(t, migrator.failedAddrs)
		})
	}
}

// Test migration statistics tracking
func TestMigrationStatsTracking(t *testing.T) {
	logger := log.NewNopLogger()
	migrator := NewMigrator(logger, nil, Mainnet, ModeAutoMigrate)

	// Process some contracts
	testContracts := []struct {
		address string
		input   map[int][]byte
	}{
		{
			"xion1legacy1",
			map[int][]byte{7: []byte("ext")},
		},
		{
			"xion1legacy2",
			map[int][]byte{7: []byte("ext")},
		},
		{
			"xion1broken1",
			map[int][]byte{7: []byte("ibc"), 8: []byte("ext")},
		},
		{
			"xion1canonical",
			map[int][]byte{7: []byte("ext"), 8: []byte("")},
		},
	}

	for _, tc := range testContracts {
		data := createTestProtobuf(tc.input)
		_, _, err := migrator.MigrateContract(tc.address, data)
		require.NoError(t, err)
	}

	// Check stats
	require.Equal(t, uint64(2), migrator.stats.LegacyCount)
	require.Equal(t, uint64(1), migrator.stats.BrokenCount)
	require.Equal(t, uint64(1), migrator.stats.CanonicalCount)
}

func TestMigrateContract_ErrorHandling(t *testing.T) {
	logger := log.NewNopLogger()
	migrator := NewMigrator(logger, nil, Mainnet, ModeAutoMigrate)

	// Test with corrupted data
	corruptedData := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF}

	result, changed, err := migrator.MigrateContract("xion1corrupted", corruptedData)

	// Should handle gracefully
	require.NoError(t, err) // Corrupted data is detected as SchemaUnknown, which doesn't need migration
	require.False(t, changed)
	require.NotNil(t, result)
}

// Test constants and configuration
func TestGetWorkerCount(t *testing.T) {
	tests := []struct {
		network NetworkType
		want    int
	}{
		{Mainnet, MainnetWorkers},
		{Testnet, TestnetWorkers},
	}

	for _, tt := range tests {
		t.Run(string(tt.network), func(t *testing.T) {
			result := GetWorkerCount(tt.network)
			require.Equal(t, tt.want, result)
		})
	}
}

func TestGetBatchSize(t *testing.T) {
	tests := []struct {
		network NetworkType
		want    int
	}{
		{Mainnet, BatchSize},
		{Testnet, TestnetBatch},
	}

	for _, tt := range tests {
		t.Run(string(tt.network), func(t *testing.T) {
			result := GetBatchSize(tt.network)
			require.Equal(t, tt.want, result)
		})
	}
}

func TestGetProgressInterval(t *testing.T) {
	tests := []struct {
		network NetworkType
		want    int
	}{
		{Mainnet, ProgressLogEvery},
		{Testnet, TestnetProgressLog},
	}

	for _, tt := range tests {
		t.Run(string(tt.network), func(t *testing.T) {
			result := GetProgressInterval(tt.network)
			require.Equal(t, tt.want, result)
		})
	}
}

func TestGetSampleRate(t *testing.T) {
	tests := []struct {
		network NetworkType
		want    float64
	}{
		{Mainnet, MainnetSampleRate},
		{Testnet, TestnetSampleRate},
	}

	for _, tt := range tests {
		t.Run(string(tt.network), func(t *testing.T) {
			result := GetSampleRate(tt.network)
			require.Equal(t, tt.want, result)
		})
	}
}

// Test ContractMigrationResult
func TestContractMigrationResult(t *testing.T) {
	result := ContractMigrationResult{
		Address:        "xion1test",
		OriginalSchema: SchemaBroken,
		Success:        true,
		Error:          nil,
		Migrated:       true,
		SkipReason:     "",
	}

	require.Equal(t, "xion1test", result.Address)
	require.Equal(t, SchemaBroken, result.OriginalSchema)
	require.True(t, result.Success)
	require.True(t, result.Migrated)
	require.Empty(t, result.SkipReason)
}

// Test MigrationMode enum
func TestMigrationMode(t *testing.T) {
	modes := []MigrationMode{
		ModeFailOnCorruption,
		ModeAutoMigrate,
		ModeLogAndContinue,
	}

	// Just verify they're different values
	for i, mode1 := range modes {
		for j, mode2 := range modes {
			if i != j {
				require.NotEqual(t, mode1, mode2)
			}
		}
	}
}

// Test NetworkType
func TestNetworkType(t *testing.T) {
	require.Equal(t, NetworkType("mainnet"), Mainnet)
	require.Equal(t, NetworkType("testnet"), Testnet)
	require.NotEqual(t, Mainnet, Testnet)
}
