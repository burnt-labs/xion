package v24_upgrade

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"
)

func TestNewValidator(t *testing.T) {
	logger := log.NewNopLogger()

	tests := []struct {
		name    string
		network NetworkType
	}{
		{"mainnet validator", Mainnet},
		{"testnet validator", Testnet},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewValidator(logger, nil, tt.network)

			require.NotNil(t, validator)
			require.Equal(t, tt.network, validator.network)
		})
	}
}

func TestValidationResult(t *testing.T) {
	result := ValidationResult{
		Address:     "xion1test",
		Valid:       true,
		Error:       nil,
		Field7Type:  WireBytes,
		Field8Empty: true,
	}

	require.Equal(t, "xion1test", result.Address)
	require.True(t, result.Valid)
	require.NoError(t, result.Error)
	require.Equal(t, WireBytes, result.Field7Type)
	require.True(t, result.Field8Empty)
}

func TestValidateSchemaDistribution(t *testing.T) {
	logger := log.NewNopLogger()
	validator := NewValidator(logger, nil, Mainnet)

	tests := []struct {
		name    string
		stats   MigrationStats
		wantErr bool
	}{
		{
			name: "valid distribution",
			stats: MigrationStats{
				TotalContracts: 1000,
				LegacyCount:    400,
				BrokenCount:    300,
				CanonicalCount: 200,
				UnknownCount:   100,
			},
			wantErr: false,
		},
		{
			name: "count mismatch",
			stats: MigrationStats{
				TotalContracts: 1000,
				LegacyCount:    400,
				BrokenCount:    300,
				CanonicalCount: 200,
				UnknownCount:   50, // Should be 100 to match total
			},
			wantErr: true,
		},
		{
			name: "all legacy",
			stats: MigrationStats{
				TotalContracts: 100,
				LegacyCount:    100,
				BrokenCount:    0,
				CanonicalCount: 0,
				UnknownCount:   0,
			},
			wantErr: false,
		},
		{
			name: "all broken",
			stats: MigrationStats{
				TotalContracts: 100,
				LegacyCount:    0,
				BrokenCount:    100,
				CanonicalCount: 0,
				UnknownCount:   0,
			},
			wantErr: false,
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

func TestSampleAddresses(t *testing.T) {
	logger := log.NewNopLogger()
	validator := NewValidator(logger, nil, Testnet)

	addresses := []string{
		"xion1addr1",
		"xion1addr2",
		"xion1addr3",
		"xion1addr4",
		"xion1addr5",
		"xion1addr6",
		"xion1addr7",
		"xion1addr8",
		"xion1addr9",
		"xion1addr10",
	}

	tests := []struct {
		name       string
		n          int
		wantLength int
	}{
		{"sample half", 5, 5},
		{"sample all", 10, 10},
		{"sample more than available", 15, 10},
		{"sample one", 1, 1},
		{"sample zero", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sampled := validator.sampleAddresses(addresses, tt.n)

			require.Equal(t, tt.wantLength, len(sampled))

			// Verify all sampled addresses are from original list
			for _, addr := range sampled {
				found := false
				for _, origAddr := range addresses {
					if addr == origAddr {
						found = true
						break
					}
				}
				require.True(t, found, "sampled address should be from original list")
			}
		})
	}
}

func TestSampleAddresses_Randomness(t *testing.T) {
	logger := log.NewNopLogger()
	validator := NewValidator(logger, nil, Mainnet)

	addresses := make([]string, 100)
	for i := 0; i < 100; i++ {
		addresses[i] = string(rune('a' + i))
	}

	// Sample multiple times and verify we get different results (randomness)
	sample1 := validator.sampleAddresses(addresses, 10)
	sample2 := validator.sampleAddresses(addresses, 10)

	// They should be different (with very high probability)
	// Compare as strings to check if order is different
	same := true
	for i := 0; i < 10; i++ {
		if sample1[i] != sample2[i] {
			same = false
			break
		}
	}

	// It's technically possible they're the same, but extremely unlikely
	// If this fails repeatedly, there's an issue with randomness
	require.False(t, same, "samples should be different (check if test fails repeatedly)")
}

// Test MigrationStats structure
func TestMigrationStats(t *testing.T) {
	stats := MigrationStats{
		TotalContracts:     1000,
		ProcessedContracts: 1000,
		MigratedContracts:  300,
		SkippedContracts:   700,
		FailedContracts:    0,
		LegacyCount:        400,
		BrokenCount:        300,
		CanonicalCount:     300,
		UnknownCount:       0,
	}

	require.Equal(t, uint64(1000), stats.TotalContracts)
	require.Equal(t, uint64(1000), stats.ProcessedContracts)
	require.Equal(t, uint64(300), stats.MigratedContracts)
	require.Equal(t, uint64(700), stats.SkippedContracts)
	require.Equal(t, uint64(0), stats.FailedContracts)

	// Verify schema counts add up to total
	totalSchemas := stats.LegacyCount + stats.BrokenCount + stats.CanonicalCount + stats.UnknownCount
	require.Equal(t, stats.TotalContracts, totalSchemas)
}

// Test MigrationReport structure
func TestMigrationReport(t *testing.T) {
	report := MigrationReport{
		Stats: MigrationStats{
			TotalContracts:    100,
			MigratedContracts: 50,
			SkippedContracts:  50,
		},
		FailedAddresses: []string{"xion1failed1", "xion1failed2"},
		NetworkType:     Mainnet,
		Mode:            ModeAutoMigrate,
	}

	require.Equal(t, uint64(100), report.Stats.TotalContracts)
	require.Equal(t, uint64(50), report.Stats.MigratedContracts)
	require.Equal(t, 2, len(report.FailedAddresses))
	require.Equal(t, Mainnet, report.NetworkType)
	require.Equal(t, ModeAutoMigrate, report.Mode)
}

// Test validation with various contract states
func TestValidateContractData(t *testing.T) {
	tests := []struct {
		name            string
		input           map[int][]byte
		wantValid       bool
		wantField7Type  int
		wantField8Empty bool
	}{
		{
			name: "valid canonical contract",
			input: map[int][]byte{
				1: []byte("code_id"),
				7: []byte("extension"), // Wire type 2 (bytes)
				// Field 8 absent/empty
			},
			wantValid:       true,
			wantField7Type:  WireBytes,
			wantField8Empty: true,
		},
		{
			name: "valid with explicit empty field 8",
			input: map[int][]byte{
				1: []byte("code_id"),
				7: []byte("extension"),
				8: []byte(""), // Explicitly empty
			},
			wantValid:       true,
			wantField7Type:  WireBytes,
			wantField8Empty: true,
		},
		{
			name: "invalid - field 8 has data",
			input: map[int][]byte{
				1: []byte("code_id"),
				7: []byte("extension"),
				8: []byte("should-be-empty"),
			},
			wantValid:       false,
			wantField7Type:  WireBytes,
			wantField8Empty: false,
		},
		{
			name: "invalid - field 7 missing",
			input: map[int][]byte{
				1: []byte("code_id"),
				// Field 7 missing
			},
			wantValid:       false,
			wantField7Type:  0,
			wantField8Empty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := createTestProtobuf(tt.input)

			// Parse to get field info
			fields, err := ParseProtobufFields(data)
			require.NoError(t, err)

			// Check field 7
			field7, hasField7 := fields[7]
			if tt.wantValid && tt.name != "invalid - field 7 missing" {
				require.True(t, hasField7)
				require.Equal(t, tt.wantField7Type, field7.WireType)
			}

			// Check field 8
			field8Empty := IsField8Empty(data)
			require.Equal(t, tt.wantField8Empty, field8Empty)
		})
	}
}

// Test sample size calculation
func TestValidationSampleSize(t *testing.T) {
	tests := []struct {
		name           string
		network        NetworkType
		totalContracts uint64
		minSample      int
	}{
		{
			name:           "mainnet 6M contracts",
			network:        Mainnet,
			totalContracts: 6000000,
			minSample:      6000, // 0.1% of 6M
		},
		{
			name:           "testnet 500K contracts",
			network:        Testnet,
			totalContracts: 500000,
			minSample:      5000, // 1% of 500K
		},
		{
			name:           "small testnet",
			network:        Testnet,
			totalContracts: 1000,
			minSample:      100, // Minimum sample size applies
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sampleRate := GetSampleRate(tt.network)
			expectedSample := int(float64(tt.totalContracts) * sampleRate)

			// Minimum sample size is 100
			if expectedSample < 100 {
				expectedSample = 100
			}

			require.GreaterOrEqual(t, expectedSample, tt.minSample)
		})
	}
}

// Test validation error scenarios
func TestValidationErrors(t *testing.T) {
	logger := log.NewNopLogger()
	validator := NewValidator(logger, nil, Testnet)

	tests := []struct {
		name    string
		stats   MigrationStats
		wantErr bool
	}{
		{
			name: "mismatch in counts",
			stats: MigrationStats{
				TotalContracts: 100,
				LegacyCount:    50,
				BrokenCount:    30,
				CanonicalCount: 10,
				UnknownCount:   5, // Total is 95, not 100
			},
			wantErr: true,
		},
		{
			name: "perfect match",
			stats: MigrationStats{
				TotalContracts: 100,
				LegacyCount:    50,
				BrokenCount:    30,
				CanonicalCount: 15,
				UnknownCount:   5,
			},
			wantErr: false,
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

// TestValidateContract_SchemaDetection tests that validation correctly handles different schemas
func TestValidateContract_SchemaDetection(t *testing.T) {
	tests := []struct {
		name      string
		hasField7 bool
		hasField8Data bool
		expectedSchema SchemaVersion
		expectedValid bool
		description string
	}{
		{
			name:      "Legacy contract (no field 7, no field 8)",
			hasField7: false,
			hasField8Data: false,
			expectedSchema: SchemaLegacy,
			expectedValid: true,
			description: "Pre-v20 contracts without extension field",
		},
		{
			name:      "Broken contract (field 8 has data)",
			hasField7: true,
			hasField8Data: true,
			expectedSchema: SchemaBroken,
			expectedValid: false,
			description: "v20/v21 contracts with extension at field 8 (should be migrated)",
		},
		{
			name:      "Canonical/migrated contract (field 7 present, field 8 empty)",
			hasField7: true,
			hasField8Data: false,
			expectedSchema: SchemaLegacy, // Detector groups both safe schemas as SchemaLegacy
			expectedValid: true,
			description: "Post-migration or v22 contracts - validator differentiates by field 7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build test contract data
			fields := map[int][]byte{
				1: []byte{0x01}, // code_id
			}

			if tt.hasField7 {
				fields[7] = []byte("extension-data")
			}

			if tt.hasField8Data {
				fields[8] = []byte("field8-data")
			}

			data := createTestProtobuf(fields)

			// Verify schema detection
			schema := DetectSchemaVersion(data)
			require.Equal(t, tt.expectedSchema, schema, tt.description)

			// Verify migration decision is correct
			needsMigration := NeedsMigration(schema)
			// Both SchemaLegacy and SchemaBroken need migration
			shouldMigrate := (tt.expectedSchema == SchemaBroken || tt.expectedSchema == SchemaLegacy)
			require.Equal(t, shouldMigrate, needsMigration,
				"SchemaLegacy and SchemaBroken should need migration")

			// Note: Actual validation requires a context and store,
			// which is tested in integration tests
		})
	}
}
