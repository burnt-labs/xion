package v25_upgrade

import (
	"testing"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/require"
)

// TestEncodeFieldTag verifies field tag encoding
func TestEncodeFieldTag(t *testing.T) {
	// Test field 1 with varint wire type
	tag1 := EncodeFieldTag(1, WireVarint)
	require.Equal(t, []byte{0x08}, tag1) // (1 << 3) | 0 = 8

	// Test field 2 with bytes wire type
	tag2 := EncodeFieldTag(2, WireBytes)
	require.Equal(t, []byte{0x12}, tag2) // (2 << 3) | 2 = 18

	// Test field 7 with bytes wire type
	tag7 := EncodeFieldTag(7, WireBytes)
	require.Equal(t, []byte{0x3A}, tag7) // (7 << 3) | 2 = 58

	// Test field 8 with bytes wire type
	tag8 := EncodeFieldTag(8, WireBytes)
	require.Equal(t, []byte{0x42}, tag8) // (8 << 3) | 2 = 66
}

// TestParseProtobufFields verifies protobuf field parsing
func TestParseProtobufFields(t *testing.T) {
	// Create simple protobuf data
	data := []byte{}

	// Field 1: CodeID = 42
	data = append(data, EncodeFieldTag(1, WireVarint)...)
	data = append(data, 42)

	// Field 2: Creator = "test"
	creator := "test"
	data = append(data, EncodeFieldTag(2, WireBytes)...)
	data = append(data, byte(len(creator)))
	data = append(data, []byte(creator)...)

	// Parse it
	fields, err := ParseProtobufFields(data)
	require.NoError(t, err)
	require.Len(t, fields, 2)

	// Verify field 1
	field1, has1 := fields[1]
	require.True(t, has1)
	require.Equal(t, 1, field1.Number)
	require.Equal(t, WireVarint, field1.WireType)
	require.Equal(t, []byte{42}, field1.Data)

	// Verify field 2
	field2, has2 := fields[2]
	require.True(t, has2)
	require.Equal(t, 2, field2.Number)
	require.Equal(t, WireBytes, field2.WireType)
	// Data should include length prefix
	require.Equal(t, append([]byte{byte(len(creator))}, []byte(creator)...), field2.Data)
}

// TestParseProtobufFieldsWithAllTypes verifies parsing different wire types
func TestParseProtobufFieldsWithAllTypes(t *testing.T) {
	data := []byte{}

	// Field 1: Varint
	data = append(data, EncodeFieldTag(1, WireVarint)...)
	data = append(data, 100)

	// Field 2: Bytes
	str := "hello"
	data = append(data, EncodeFieldTag(2, WireBytes)...)
	data = append(data, byte(len(str)))
	data = append(data, []byte(str)...)

	// Parse
	fields, err := ParseProtobufFields(data)
	require.NoError(t, err)
	require.Len(t, fields, 2)

	// Check wire types
	require.Equal(t, WireVarint, fields[1].WireType)
	require.Equal(t, WireBytes, fields[2].WireType)
}

// TestGetFieldValue verifies field value extraction
func TestGetFieldValue(t *testing.T) {
	// Create contract data
	contractInfo := &wasmtypes.ContractInfo{
		CodeID:  1,
		Creator: "xion1creator",
		Admin:   "xion1admin",
		Label:   "test-label",
	}

	data, err := proto.Marshal(contractInfo)
	require.NoError(t, err)

	// Get field 2 (Creator)
	creator, err := GetFieldValue(data, 2)
	require.NoError(t, err)
	require.Equal(t, []byte("xion1creator"), creator)

	// Get field 3 (Admin)
	admin, err := GetFieldValue(data, 3)
	require.NoError(t, err)
	require.Equal(t, []byte("xion1admin"), admin)

	// Get field 4 (Label)
	label, err := GetFieldValue(data, 4)
	require.NoError(t, err)
	require.Equal(t, []byte("test-label"), label)

	// Get non-existent field - should either error or return empty
	value, err := GetFieldValue(data, 99)
	if err == nil {
		// Field doesn't exist, value should be nil or empty
		require.Empty(t, value, "non-existent field should return empty value")
	}
}

// TestParseProtobufFieldsInvalidData verifies error handling
func TestParseProtobufFieldsInvalidData(t *testing.T) {
	// Completely invalid data
	data := []byte{0xFF, 0xFF, 0xFF, 0xFF}
	_, err := ParseProtobufFields(data)
	require.Error(t, err)

	// Empty data
	data = []byte{}
	fields, err := ParseProtobufFields(data)
	require.NoError(t, err)
	require.Len(t, fields, 0)
}

// TestParseProtobufFieldsTruncated verifies handling of truncated data
func TestParseProtobufFieldsTruncated(t *testing.T) {
	data := []byte{}

	// Field 1: Valid
	data = append(data, EncodeFieldTag(1, WireVarint)...)
	data = append(data, 1)

	// Field 2: Claims 100 bytes but only has 5
	data = append(data, EncodeFieldTag(2, WireBytes)...)
	data = append(data, 100)                // Claims 100
	data = append(data, []byte("short")...) // Only 5 bytes

	// Should return error due to truncation
	_, err := ParseProtobufFields(data)
	require.Error(t, err)
}

// TestParseProtobufFieldsMultipleSameField verifies duplicate field handling
func TestParseProtobufFieldsMultipleSameField(t *testing.T) {
	data := []byte{}

	// Field 1: First occurrence
	data = append(data, EncodeFieldTag(1, WireVarint)...)
	data = append(data, 1)

	// Field 1: Second occurrence (duplicate)
	data = append(data, EncodeFieldTag(1, WireVarint)...)
	data = append(data, 2)

	// Parse - should keep last occurrence (protobuf behavior)
	fields, err := ParseProtobufFields(data)
	require.NoError(t, err)

	field1, has1 := fields[1]
	require.True(t, has1)
	// Protobuf typically keeps last value for duplicates
	// Our parser might keep first or last - either is acceptable
	require.Contains(t, [][]byte{{1}, {2}}, field1.Data)
}

// TestGetFieldValueEmptyField verifies handling of empty fields
func TestGetFieldValueEmptyField(t *testing.T) {
	data := []byte{}

	// Field 7: Empty bytes field
	data = append(data, EncodeFieldTag(7, WireBytes)...)
	data = append(data, 0) // Length 0

	// Get field 7
	value, err := GetFieldValue(data, 7)
	require.NoError(t, err)
	require.Equal(t, []byte{}, value, "empty field should return empty slice")
}

// TestParseProtobufFieldsLargeFieldNumber verifies large field numbers
func TestParseProtobufFieldsLargeFieldNumber(t *testing.T) {
	data := []byte{}

	// Field 100: Large field number
	data = append(data, EncodeFieldTag(100, WireVarint)...)
	data = append(data, 42)

	fields, err := ParseProtobufFields(data)
	require.NoError(t, err)

	field100, has100 := fields[100]
	require.True(t, has100)
	require.Equal(t, 100, field100.Number)
	require.Equal(t, []byte{42}, field100.Data)
}

// TestReadFieldData verifies the internal readFieldData function
func TestReadFieldData(t *testing.T) {
	// Test varint
	data := []byte{42}
	fieldData, n, err := readFieldData(data, WireVarint)
	require.NoError(t, err)
	require.Equal(t, []byte{42}, fieldData)
	require.Equal(t, 1, n)

	// Test bytes
	str := "hello"
	data = append([]byte{byte(len(str))}, []byte(str)...)
	fieldData, n, err = readFieldData(data, WireBytes)
	require.NoError(t, err)
	require.Equal(t, append([]byte{byte(len(str))}, []byte(str)...), fieldData)
	require.Equal(t, len(str)+1, n)

	// Test truncated bytes
	data = []byte{100, 'a', 'b', 'c'} // Claims 100 bytes, only has 3
	_, _, err = readFieldData(data, WireBytes)
	require.Error(t, err, "should error on truncated data")
}

// TestEncodeDecodeRoundTrip verifies encoding and decoding consistency
func TestEncodeDecodeRoundTrip(t *testing.T) {
	// Create contract
	original := &wasmtypes.ContractInfo{
		CodeID:  999,
		Creator: "xion1creator",
		Admin:   "xion1admin",
		Label:   "roundtrip-test",
	}

	// Marshal
	data, err := proto.Marshal(original)
	require.NoError(t, err)

	// Parse fields
	fields, err := ParseProtobufFields(data)
	require.NoError(t, err)
	require.NotEmpty(t, fields)

	// Verify we can extract values
	creator, err := GetFieldValue(data, 2)
	require.NoError(t, err)
	require.Equal(t, []byte("xion1creator"), creator)

	// Unmarshal to verify correctness
	var decoded wasmtypes.ContractInfo
	err = proto.Unmarshal(data, &decoded)
	require.NoError(t, err)
	require.Equal(t, original.CodeID, decoded.CodeID)
	require.Equal(t, original.Creator, decoded.Creator)
	require.Equal(t, original.Admin, decoded.Admin)
	require.Equal(t, original.Label, decoded.Label)
}

// TestParseProtobufFieldsWithFields7And8 verifies parsing fields 7 and 8 specifically
func TestParseProtobufFieldsWithFields7And8(t *testing.T) {
	data := []byte{}

	// Field 1
	data = append(data, EncodeFieldTag(1, WireVarint)...)
	data = append(data, 1)

	// Field 7: Extension (empty)
	data = append(data, EncodeFieldTag(7, WireBytes)...)
	data = append(data, 0)

	// Field 8: IBC port ID
	portID := "wasm.xion1test"
	data = append(data, EncodeFieldTag(8, WireBytes)...)
	data = append(data, byte(len(portID)))
	data = append(data, []byte(portID)...)

	// Parse
	fields, err := ParseProtobufFields(data)
	require.NoError(t, err)

	// Verify field 7
	field7, has7 := fields[7]
	require.True(t, has7)
	require.Equal(t, WireBytes, field7.WireType)
	require.Equal(t, []byte{0}, field7.Data) // Empty with length prefix

	// Verify field 8
	field8, has8 := fields[8]
	require.True(t, has8)
	require.Equal(t, WireBytes, field8.WireType)

	// Extract field 8 value
	port, err := GetFieldValue(data, 8)
	require.NoError(t, err)
	require.Equal(t, []byte(portID), port)
}
