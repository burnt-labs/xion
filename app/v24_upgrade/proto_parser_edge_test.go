package v24_upgrade

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
)

// Additional tests for edge cases in protobuf parsing

func TestReadFieldData_AllWireTypes(t *testing.T) {
	tests := []struct {
		name     string
		wireType int
		data     []byte
		wantLen  int
		wantErr  bool
	}{
		{
			name:     "varint",
			wireType: WireVarint,
			data:     []byte{0x96, 0x01}, // 150 as varint
			wantLen:  2,
			wantErr:  false,
		},
		{
			name:     "fixed64",
			wireType: WireFixed64,
			data:     []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
			wantLen:  8,
			wantErr:  false,
		},
		{
			name:     "fixed32",
			wireType: WireFixed32,
			data:     []byte{0x01, 0x02, 0x03, 0x04},
			wantLen:  4,
			wantErr:  false,
		},
		{
			name:     "bytes (length-delimited)",
			wireType: WireBytes,
			data:     append([]byte{0x05}, []byte("hello")...), // length 5 + "hello"
			wantLen:  6,
			wantErr:  false,
		},
		{
			name:     "invalid varint - truncated",
			wireType: WireVarint,
			data:     []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}, // Too long
			wantLen:  0,
			wantErr:  true,
		},
		{
			name:     "fixed64 - truncated",
			wireType: WireFixed64,
			data:     []byte{0x01, 0x02}, // Only 2 bytes instead of 8
			wantLen:  0,
			wantErr:  true,
		},
		{
			name:     "fixed32 - truncated",
			wireType: WireFixed32,
			data:     []byte{0x01}, // Only 1 byte instead of 4
			wantLen:  0,
			wantErr:  true,
		},
		{
			name:     "bytes - truncated data",
			wireType: WireBytes,
			data:     []byte{0x0A, 0x01, 0x02}, // Says length 10 but only has 2 bytes
			wantLen:  0,
			wantErr:  true,
		},
		{
			name:     "bytes - invalid length prefix",
			wireType: WireBytes,
			data:     []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}, // Invalid varint
			wantLen:  0,
			wantErr:  true,
		},
		{
			name:     "unsupported wire type",
			wireType: WireStartGroup, // Wire type 3 (deprecated)
			data:     []byte{0x01, 0x02},
			wantLen:  0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fieldData, bytesRead, err := readFieldData(tt.data, tt.wireType)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.wantLen, bytesRead)
			require.Len(t, fieldData, tt.wantLen)
		})
	}
}

func TestParseProtobufFields_Malformed(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "truncated tag",
			data:    []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
			wantErr: true,
		},
		{
			name:    "valid tag but truncated data",
			data:    []byte{0x3A, 0xFF}, // Field 7, wire type 2, but truncated length
			wantErr: true,
		},
		{
			name: "multiple fields with one truncated",
			data: append(
				createTestProtobuf(map[int][]byte{1: []byte("valid")}),
				[]byte{0xFF, 0xFF}..., // Invalid trailing data
			),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseProtobufFields(tt.data)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGetFieldValue_EmptyData(t *testing.T) {
	// Test with various empty/null scenarios
	tests := []struct {
		name    string
		data    []byte
		field   int
		wantNil bool
	}{
		{
			name:    "empty protobuf",
			data:    []byte{},
			field:   1,
			wantNil: true,
		},
		{
			name:    "field doesn't exist",
			data:    createTestProtobuf(map[int][]byte{7: []byte("data")}),
			field:   8,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := GetFieldValue(tt.data, tt.field)

			require.NoError(t, err)
			if tt.wantNil {
				require.Nil(t, value)
			}
		})
	}
}

func TestHasField_EdgeCases(t *testing.T) {
	// Test with corrupted data
	corruptedData := []byte{0xFF, 0xFF, 0xFF}

	result := HasField(corruptedData, 1)
	require.False(t, result, "corrupted data should return false")
}

func TestIsField8Empty_EdgeCases(t *testing.T) {
	// Test with corrupted data
	corruptedData := []byte{0xFF, 0xFF, 0xFF}

	result := IsField8Empty(corruptedData)
	require.True(t, result, "corrupted data should be treated as empty")
}

func TestSwapFields7And8_PreserveOtherFields(t *testing.T) {
	// Create a protobuf with many fields
	input := createTestProtobuf(map[int][]byte{
		1: []byte("field1"),
		2: []byte("field2"),
		3: []byte("field3"),
		4: []byte("field4"),
		5: []byte("field5"),
		6: []byte("field6"),
		7: []byte("field7"),
		8: []byte("field8"),
		9: []byte("field9"),
	})

	result, err := SwapFields7And8(input)
	require.NoError(t, err)

	// Verify all fields except 7 and 8 are preserved
	for i := 1; i <= 9; i++ {
		value, err := GetFieldValue(result, i)
		require.NoError(t, err)

		switch i {
		case 7:
			require.Equal(t, []byte("field8"), value, "field 7 should have field 8's data")
		case 8:
			require.Equal(t, []byte("field7"), value, "field 8 should have field 7's data")
		default:
			expectedValue := []byte("field" + string(rune('0'+i)))
			require.Equal(t, expectedValue, value, "field %d should be preserved", i)
		}
	}
}

func TestClearField8_PreserveOtherFields(t *testing.T) {
	// Create a protobuf with many fields
	input := createTestProtobuf(map[int][]byte{
		1: []byte("field1"),
		2: []byte("field2"),
		7: []byte("field7"),
		8: []byte("field8-to-be-cleared"),
		9: []byte("field9"),
	})

	result, err := ClearField8(input)
	require.NoError(t, err)

	// Verify field 8 is gone
	require.True(t, IsField8Empty(result))

	// Verify other fields are preserved
	fields := []int{1, 2, 7, 9}
	for _, fieldNum := range fields {
		value, err := GetFieldValue(result, fieldNum)
		require.NoError(t, err)
		require.NotNil(t, value, "field %d should be preserved", fieldNum)
	}
}

// Test protobuf with very large field numbers
func TestParseProtobufFields_LargeFieldNumbers(t *testing.T) {
	// Create a protobuf with field number 536870911 (max allowed)
	data := []byte{}

	// Encode field tag: field number 100, wire type 2
	fieldNum := 100
	wireType := WireBytes
	tag := uint64(fieldNum<<3 | wireType)

	tagBuf := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(tagBuf, tag)
	data = append(data, tagBuf[:n]...)

	// Add length and data
	testData := []byte("test")
	lenBuf := make([]byte, binary.MaxVarintLen64)
	m := binary.PutUvarint(lenBuf, uint64(len(testData)))
	data = append(data, lenBuf[:m]...)
	data = append(data, testData...)

	fields, err := ParseProtobufFields(data)
	require.NoError(t, err)
	require.Contains(t, fields, 100)
}

// Test with extremely small and large data
func TestProtobufParsing_SizeEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		fieldData []byte
	}{
		{
			name:      "single byte",
			fieldData: []byte{0x01},
		},
		{
			name:      "large data",
			fieldData: make([]byte, 10000), // 10KB
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := createTestProtobuf(map[int][]byte{
				7: tt.fieldData,
			})

			value, err := GetFieldValue(input, 7)
			require.NoError(t, err)
			require.Equal(t, tt.fieldData, value)
		})
	}
}
