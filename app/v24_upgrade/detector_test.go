package v24_upgrade

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDetectSchemaVersion(t *testing.T) {
	tests := []struct {
		name  string
		input map[int][]byte
		want  SchemaVersion
	}{
		{
			name: "SchemaLegacy - field 7 present, no field 8",
			input: map[int][]byte{
				1: []byte("code_id"),
				7: []byte("extension"),
				// No field 8
			},
			want: SchemaLegacy,
		},
		{
			name: "SchemaCanonical - field 7 present, field 8 empty",
			input: map[int][]byte{
				1: []byte("code_id"),
				7: []byte("extension"),
				8: []byte(""), // Empty
			},
			want: SchemaCanonical,
		},
		{
			name: "SchemaBroken - field 8 has data",
			input: map[int][]byte{
				1: []byte("code_id"),
				7: []byte("ibc_port_id"),
				8: []byte("extension"), // Has data - indicates corruption
			},
			want: SchemaBroken,
		},
		{
			name: "SchemaLegacy - no field 7, no field 8",
			input: map[int][]byte{
				1: []byte("code_id"),
				// No field 7
				// No field 8
			},
			want: SchemaLegacy,
		},
		{
			name:  "SchemaLegacy - completely empty",
			input: map[int][]byte{},
			want:  SchemaLegacy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := createTestProtobuf(tt.input)

			result := DetectSchemaVersion(input)
			require.Equal(t, tt.want, result)
		})
	}
}

func TestDetectSchemaVersion_Corrupted(t *testing.T) {
	// Test with corrupted data (invalid protobuf)
	corruptedData := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF}

	result := DetectSchemaVersion(corruptedData)
	require.Equal(t, SchemaCorrupted, result, "Invalid protobuf should be detected as SchemaCorrupted")
}

func TestNeedsMigration(t *testing.T) {
	tests := []struct {
		name   string
		schema SchemaVersion
		want   bool
	}{
		{"SchemaLegacy needs migration (add field 7)", SchemaLegacy, true},
		{"SchemaCanonical doesn't need migration", SchemaCanonical, false},
		{"SchemaBroken needs migration", SchemaBroken, true},
		{"SchemaCorrupted needs attention (will fail)", SchemaCorrupted, true},
		{"SchemaUnknown doesn't need migration", SchemaUnknown, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NeedsMigration(tt.schema)
			require.Equal(t, tt.want, result)
		})
	}
}

func TestGetMigrationAction(t *testing.T) {
	tests := []struct {
		name   string
		schema SchemaVersion
		want   string
	}{
		{
			name:   "SchemaLegacy",
			schema: SchemaLegacy,
			want:   "Add missing fields (field 7 extension and/or field 8 ibc2_port_id)",
		},
		{
			name:   "SchemaBroken",
			schema: SchemaBroken,
			want:   "Swap fields 7 and 8, then null field 8",
		},
		{
			name:   "SchemaCanonical",
			schema: SchemaCanonical,
			want:   "None - already correct",
		},
		{
			name:   "SchemaCorrupted",
			schema: SchemaCorrupted,
			want:   "Cannot fix - data corruption (invalid wire types, truncated data, etc.)",
		},
		{
			name:   "SchemaUnknown",
			schema: SchemaUnknown,
			want:   "Unknown - manual inspection required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetMigrationAction(tt.schema)
			require.Equal(t, tt.want, result)
		})
	}
}

func TestAnalyzeContractData(t *testing.T) {
	tests := []struct {
		name               string
		address            string
		input              map[int][]byte
		wantSchema         SchemaVersion
		wantHasField7      bool
		wantHasField8      bool
		wantField8HasData  bool
		wantNeedsMigration bool
	}{
		{
			name:    "SchemaLegacy contract",
			address: "xion1test1",
			input: map[int][]byte{
				1: []byte("code_id"),
				// No field 7 - true legacy
			},
			wantSchema:         SchemaLegacy,
			wantHasField7:      false,
			wantHasField8:      false,
			wantField8HasData:  false,
			wantNeedsMigration: true,
		},
		{
			name:    "SchemaBroken contract",
			address: "xion1broken",
			input: map[int][]byte{
				1: []byte("code_id"),
				7: []byte("ibc_port_id"),
				8: []byte("extension"),
			},
			wantSchema:         SchemaBroken,
			wantHasField7:      true,
			wantHasField8:      true,
			wantField8HasData:  true,
			wantNeedsMigration: true,
		},
		{
			name:    "SchemaCanonical contract",
			address: "xion1canonical",
			input: map[int][]byte{
				1: []byte("code_id"),
				7: []byte("extension"),
				8: []byte(""),
			},
			wantSchema:         SchemaCanonical,
			wantHasField7:      true,
			wantHasField8:      true,
			wantField8HasData:  false,
			wantNeedsMigration: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := createTestProtobuf(tt.input)

			analysis := AnalyzeContractData(tt.address, input)

			require.Equal(t, tt.address, analysis.Address)
			require.Equal(t, tt.wantSchema, analysis.Schema)
			require.Equal(t, tt.wantHasField7, analysis.HasField7)
			require.Equal(t, tt.wantHasField8, analysis.HasField8)
			require.Equal(t, tt.wantField8HasData, analysis.Field8HasData)
			require.Equal(t, tt.wantNeedsMigration, analysis.NeedsMigration)
			require.NoError(t, analysis.Error)
		})
	}
}

func TestAnalyzeContractData_Corrupted(t *testing.T) {
	corruptedData := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF}

	analysis := AnalyzeContractData("xion1corrupted", corruptedData)

	require.Equal(t, "xion1corrupted", analysis.Address)
	require.Equal(t, SchemaUnknown, analysis.Schema)
	require.Error(t, analysis.Error)
}

func TestDetectCorruption(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		wantErr bool
	}{
		{
			name:    "valid protobuf",
			input:   createTestProtobuf(map[int][]byte{1: []byte("data")}),
			wantErr: false,
		},
		{
			name:    "corrupted protobuf",
			input:   []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
			wantErr: true,
		},
		{
			name:    "empty protobuf",
			input:   []byte{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := DetectCorruption(tt.input)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSchemaVersion_String(t *testing.T) {
	tests := []struct {
		schema SchemaVersion
		want   string
	}{
		{SchemaLegacy, "SchemaLegacy"},
		{SchemaBroken, "SchemaBroken"},
		{SchemaCanonical, "SchemaCanonical"},
		{SchemaUnknown, "SchemaUnknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			result := tt.schema.String()
			require.Equal(t, tt.want, result)
		})
	}
}

// Integration test: Detect various real-world scenarios
func TestDetectSchemaVersion_RealWorldScenarios(t *testing.T) {
	tests := []struct {
		name        string
		description string
		input       map[int][]byte
		wantSchema  SchemaVersion
	}{
		{
			name:        "Pre-v20 contract (ancient)",
			description: "Contract created before wasmd v0.61.0 - no extension field",
			input: map[int][]byte{
				1: []byte("100"),          // code_id
				2: []byte("xion1creator"), // creator
				3: []byte("xion1admin"),   // admin
				4: []byte("my-contract"),  // label
				5: {1, 2, 3},              // created (AbsoluteTxPosition)
				6: []byte(""),             // ibc_port_id (empty)
				// No field 7 (extension) - true pre-v20 contract
				// No field 8
			},
			wantSchema: SchemaLegacy,
		},
		{
			name:        "v20/v21 broken contract",
			description: "Contract with fields swapped by bug",
			input: map[int][]byte{
				1: []byte("200"),
				2: []byte("xion1creator"),
				3: []byte("xion1admin"),
				4: []byte("broken-contract"),
				5: {1, 2, 3},
				6: []byte(""),
				7: []byte("wasm.xxxxx"),        // ibc_port_id (wrong position)
				8: {10, 5, 116, 101, 115, 116}, // extension (wrong position)
			},
			wantSchema: SchemaBroken,
		},
		{
			name:        "v22+ correct contract",
			description: "Contract with correct field ordering",
			input: map[int][]byte{
				1: []byte("300"),
				2: []byte("xion1creator"),
				3: []byte("xion1admin"),
				4: []byte("correct-contract"),
				5: {1, 2, 3},
				6: []byte(""),
				7: {10, 5, 116, 101, 115, 116}, // extension (correct)
				8: []byte(""),                  // ibc2_port_id (empty - correct)
			},
			wantSchema: SchemaCanonical,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := createTestProtobuf(tt.input)

			result := DetectSchemaVersion(input)
			require.Equal(t, tt.wantSchema, result, tt.description)
		})
	}
}
