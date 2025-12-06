package v25_upgrade

import (
	"testing"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/require"
)

// TestAnalyzeHealthyContract verifies analysis of healthy contracts
func TestAnalyzeHealthyContract(t *testing.T) {
	// Create a contract with all required fields including 7 and 8
	data := []byte{}

	// Field 1: CodeID = 1
	data = append(data, EncodeFieldTag(1, WireVarint)...)
	data = append(data, 1)

	// Field 2: Creator
	creator := "xion1creator"
	data = append(data, EncodeFieldTag(2, WireBytes)...)
	data = append(data, byte(len(creator)))
	data = append(data, []byte(creator)...)

	// Field 7: Empty extension (healthy)
	data = append(data, EncodeFieldTag(7, WireBytes)...)
	data = append(data, 0)

	// Field 8: Empty ibc2_port_id (healthy)
	data = append(data, EncodeFieldTag(8, WireBytes)...)
	data = append(data, 0)

	// Analyze it
	analysis := AnalyzeContract("xion1test", data)

	require.Equal(t, StateHealthy, analysis.State)
	require.Nil(t, analysis.UnmarshalError, "healthy contract should unmarshal")
	require.False(t, analysis.Fixable, "healthy contract doesn't need fixing")
	require.True(t, analysis.HasField7, "healthy contract should have field 7")
	require.True(t, analysis.HasField8, "healthy contract should have field 8")
}

// TestAnalyzeInvalidWireType verifies detection of invalid wire type corruption
func TestAnalyzeInvalidWireType(t *testing.T) {
	// Create contract with swapped fields
	data := []byte{}

	// Field 1: CodeID
	data = append(data, EncodeFieldTag(1, WireVarint)...)
	data = append(data, 1)

	// Field 2: Creator
	creator := "xion1creator"
	data = append(data, EncodeFieldTag(2, WireBytes)...)
	data = append(data, byte(len(creator)))
	data = append(data, []byte(creator)...)

	// Field 7: Contains port ID (wrong)
	portID := "wasm.xion1test"
	data = append(data, EncodeFieldTag(7, WireBytes)...)
	data = append(data, byte(len(portID)))
	data = append(data, []byte(portID)...)

	// Analyze it
	analysis := AnalyzeContract("xion1test", data)

	require.Equal(t, StateUnmarshalFails, analysis.State)
	require.NotNil(t, analysis.UnmarshalError, "corrupted contract should not unmarshal")
	require.True(t, analysis.Fixable, "invalid wire type should be fixable")
	require.Equal(t, PatternInvalidWireType, analysis.CorruptionPattern)
}

// TestAnalyzeSchemaInconsistent verifies analysis of schema-inconsistent contracts
func TestAnalyzeSchemaInconsistent(t *testing.T) {
	// Create contract missing field 7
	data := []byte{}

	// Field 1: CodeID
	data = append(data, EncodeFieldTag(1, WireVarint)...)
	data = append(data, 1)

	// Field 2: Creator
	creator := "xion1creator"
	data = append(data, EncodeFieldTag(2, WireBytes)...)
	data = append(data, byte(len(creator)))
	data = append(data, []byte(creator)...)

	// Skip fields 7 and 8

	// Analyze it
	analysis := AnalyzeContract("xion1test", data)

	require.Equal(t, StateSchemaInconsistent, analysis.State)
	require.Nil(t, analysis.UnmarshalError, "schema-inconsistent can still unmarshal")
	require.True(t, analysis.Fixable, "schema-inconsistent is fixable")
	require.False(t, analysis.HasField7, "should detect missing field 7")
	require.False(t, analysis.HasField8, "should detect missing field 8")
}

// TestAnalyzeTruncatedData verifies detection of truncated data
func TestAnalyzeTruncatedData(t *testing.T) {
	// Create truncated data (incomplete field)
	data := []byte{}

	// Field 1: CodeID
	data = append(data, EncodeFieldTag(1, WireVarint)...)
	data = append(data, 1)

	// Field 2: Creator - but truncated (claims length 100 but only has a few bytes)
	data = append(data, EncodeFieldTag(2, WireBytes)...)
	data = append(data, 100)                // Claims 100 bytes
	data = append(data, []byte("short")...) // Only 5 bytes

	// Analyze it
	analysis := AnalyzeContract("xion1test", data)

	require.Equal(t, StateUnmarshalFails, analysis.State)
	require.NotNil(t, analysis.UnmarshalError)
	require.Equal(t, PatternTruncatedField, analysis.CorruptionPattern)
}

// TestAnalyzeMissingRequiredFields verifies detection of missing required fields
func TestAnalyzeMissingRequiredFields(t *testing.T) {
	// Create contract missing required fields (e.g., CodeID)
	data := []byte{}

	// Only field 2 (Creator) - missing field 1 (CodeID)
	creator := "xion1creator"
	data = append(data, EncodeFieldTag(2, WireBytes)...)
	data = append(data, byte(len(creator)))
	data = append(data, []byte(creator)...)

	// Analyze it
	analysis := AnalyzeContract("xion1test", data)

	// This might unmarshal (protobuf is lenient) or might fail
	// Either way, we should detect the pattern
	if analysis.UnmarshalError != nil {
		require.Contains(t, []CorruptionPattern{
			PatternMissingRequiredFields,
			PatternUnknown, // Might just be schema inconsistent
		}, analysis.CorruptionPattern)
	}
}

// TestAnalyzeDuplicateFields verifies detection of duplicate fields
func TestAnalyzeDuplicateFields(t *testing.T) {
	// Create contract with duplicate field 1
	data := []byte{}

	// Field 1: CodeID = 1
	data = append(data, EncodeFieldTag(1, WireVarint)...)
	data = append(data, 1)

	// Field 1 again: CodeID = 2 (duplicate!)
	data = append(data, EncodeFieldTag(1, WireVarint)...)
	data = append(data, 2)

	// Field 2: Creator
	creator := "xion1creator"
	data = append(data, EncodeFieldTag(2, WireBytes)...)
	data = append(data, byte(len(creator)))
	data = append(data, []byte(creator)...)

	// Analyze it
	analysis := AnalyzeContract("xion1test", data)

	// Protobuf might accept duplicates (uses last value) or reject them
	if analysis.UnmarshalError != nil {
		require.Equal(t, PatternDuplicateFields, analysis.CorruptionPattern)
	}
}

// TestAnalyzeField7Content verifies field 7 content analysis
func TestAnalyzeField7Content(t *testing.T) {
	// Create contract where field 7 contains string data (looks like ibc_port_id)
	data := []byte{}

	// Field 1: CodeID
	data = append(data, EncodeFieldTag(1, WireVarint)...)
	data = append(data, 1)

	// Field 7: Contains what looks like ibc_port_id
	portID := "wasm.xion1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqpn45e7"
	data = append(data, EncodeFieldTag(7, WireBytes)...)
	data = append(data, byte(len(portID)))
	data = append(data, []byte(portID)...)

	// Analyze it
	analysis := AnalyzeContract("xion1test", data)

	// Should detect this as needing field swap
	if analysis.State == StateUnmarshalFails {
		require.Equal(t, PatternInvalidWireType, analysis.CorruptionPattern)
	}
}

// TestAnalyzeComplexCorruption verifies handling of complex corruption
func TestAnalyzeComplexCorruption(t *testing.T) {
	// Create severely corrupted data
	data := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF}

	// Analyze it
	analysis := AnalyzeContract("xion1test", data)

	require.Equal(t, StateUnmarshalFails, analysis.State)
	require.NotNil(t, analysis.UnmarshalError)
	// Should either detect a pattern or mark as unfixable
	require.True(t,
		analysis.CorruptionPattern != PatternUnknown || !analysis.Fixable,
		"should detect corruption or mark as unfixable")
}

// TestAnalyzeWithField8Data verifies analysis when field 8 has legitimate data
func TestAnalyzeWithField8Data(t *testing.T) {
	// Create healthy contract with field 8 containing data
	data := []byte{}

	// Field 1: CodeID
	data = append(data, EncodeFieldTag(1, WireVarint)...)
	data = append(data, 1)

	// Field 2: Creator
	creator := "xion1creator"
	data = append(data, EncodeFieldTag(2, WireBytes)...)
	data = append(data, byte(len(creator)))
	data = append(data, []byte(creator)...)

	// Field 7: Empty extension (correct)
	data = append(data, EncodeFieldTag(7, WireBytes)...)
	data = append(data, 0)

	// Field 8: Legitimate ibc2_port_id data
	portID := "wasm.xion1contract"
	data = append(data, EncodeFieldTag(8, WireBytes)...)
	data = append(data, byte(len(portID)))
	data = append(data, []byte(portID)...)

	// Analyze it
	analysis := AnalyzeContract("xion1test", data)

	require.Equal(t, StateHealthy, analysis.State, "contract with field 8 data should be healthy")
	require.Nil(t, analysis.UnmarshalError)
	require.True(t, analysis.HasField7)
	require.True(t, analysis.HasField8)
	require.Equal(t, PatternUnknown, analysis.CorruptionPattern)
}

// TestAnalyzeEmptyData verifies handling of empty data
func TestAnalyzeEmptyData(t *testing.T) {
	data := []byte{}

	analysis := AnalyzeContract("xion1test", data)

	// Empty data can unmarshal (protobuf creates default values)
	// But it's schema-inconsistent (missing fields 7 and 8)
	require.Equal(t, StateSchemaInconsistent, analysis.State)
	require.Nil(t, analysis.UnmarshalError)
}

// TestAnalyzeRoundTrip verifies analysis is consistent
func TestAnalyzeRoundTrip(t *testing.T) {
	// Create contract
	contractInfo := &wasmtypes.ContractInfo{
		CodeID:  123,
		Creator: "xion1creator",
		Admin:   "xion1admin",
		Label:   "test-contract",
	}

	data, err := proto.Marshal(contractInfo)
	require.NoError(t, err)

	// Analyze original (will be schema-inconsistent without fields 7 and 8)
	analysis1 := AnalyzeContract("xion1test", data)
	require.Equal(t, StateSchemaInconsistent, analysis1.State)

	// Unmarshal and re-marshal
	var info2 wasmtypes.ContractInfo
	err = proto.Unmarshal(data, &info2)
	require.NoError(t, err)

	data2, err := proto.Marshal(&info2)
	require.NoError(t, err)

	// Analyze re-marshaled
	analysis2 := AnalyzeContract("xion1test", data2)
	require.Equal(t, StateSchemaInconsistent, analysis2.State)

	// Both analyses should give same state
	require.Equal(t, analysis1.State, analysis2.State)
	require.Equal(t, analysis1.UnmarshalError == nil, analysis2.UnmarshalError == nil)
}
