package v24_upgrade

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
)

// Helper to create a simple protobuf with specific fields
func createTestProtobuf(fields map[int][]byte) []byte {
	result := make([]byte, 0)
	for fieldNum, data := range fields {
		// Create field tag (field number | wire type)
		tag := EncodeFieldTag(fieldNum, WireBytes)
		result = append(result, tag...)

		// Add length prefix
		lenBuf := make([]byte, binary.MaxVarintLen64)
		n := binary.PutUvarint(lenBuf, uint64(len(data)))
		result = append(result, lenBuf[:n]...)

		// Add data
		result = append(result, data...)
	}
	return result
}

func TestParseProtobufFields(t *testing.T) {
	tests := []struct {
		name       string
		input      []byte
		wantFields []int
		wantErr    bool
	}{
		{
			name:       "empty protobuf",
			input:      []byte{},
			wantFields: []int{},
			wantErr:    false,
		},
		{
			name: "single field",
			input: createTestProtobuf(map[int][]byte{
				1: []byte("test"),
			}),
			wantFields: []int{1},
			wantErr:    false,
		},
		{
			name: "multiple fields",
			input: createTestProtobuf(map[int][]byte{
				1: []byte("field1"),
				7: []byte("extension"),
				8: []byte("ibc2"),
			}),
			wantFields: []int{1, 7, 8},
			wantErr:    false,
		},
		{
			name:       "corrupted data",
			input:      []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
			wantFields: []int{},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields, err := ParseProtobufFields(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, len(tt.wantFields), len(fields))

			for _, fieldNum := range tt.wantFields {
				_, ok := fields[fieldNum]
				require.True(t, ok, "field %d should exist", fieldNum)
			}
		})
	}
}

func TestSwapFields7And8(t *testing.T) {
	tests := []struct {
		name        string
		input       map[int][]byte
		wantErr     bool
		checkField7 []byte
		checkField8 []byte
	}{
		{
			name: "both fields present - should swap",
			input: map[int][]byte{
				1: []byte("field1"),
				7: []byte("was-field-7"),
				8: []byte("was-field-8"),
			},
			wantErr:     false,
			checkField7: []byte("was-field-8"),
			checkField8: []byte("was-field-7"),
		},
		{
			name: "only field 7 present",
			input: map[int][]byte{
				1: []byte("field1"),
				7: []byte("extension"),
			},
			wantErr:     false,
			checkField7: nil,                 // Field 7 should be gone
			checkField8: []byte("extension"), // Should be at field 8 now
		},
		{
			name: "only field 8 present",
			input: map[int][]byte{
				1: []byte("field1"),
				8: []byte("ibc2"),
			},
			wantErr:     false,
			checkField7: []byte("ibc2"), // Should be at field 7 now
			checkField8: nil,            // Field 8 should be gone
		},
		{
			name: "neither field present",
			input: map[int][]byte{
				1: []byte("field1"),
				2: []byte("field2"),
			},
			wantErr:     false,
			checkField7: nil,
			checkField8: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := createTestProtobuf(tt.input)

			result, err := SwapFields7And8(input)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Parse result and check fields 7 and 8
			fields, err := ParseProtobufFields(result)
			require.NoError(t, err)

			if tt.checkField7 != nil {
				value, err := GetFieldValue(result, 7)
				require.NoError(t, err)
				require.Equal(t, tt.checkField7, value)
			} else {
				_, ok := fields[7]
				require.False(t, ok, "field 7 should not exist")
			}

			if tt.checkField8 != nil {
				value, err := GetFieldValue(result, 8)
				require.NoError(t, err)
				require.Equal(t, tt.checkField8, value)
			} else {
				_, ok := fields[8]
				require.False(t, ok, "field 8 should not exist")
			}
		})
	}
}

func TestClearField8(t *testing.T) {
	tests := []struct {
		name    string
		input   map[int][]byte
		wantErr bool
	}{
		{
			name: "field 8 present - should be removed",
			input: map[int][]byte{
				1: []byte("field1"),
				7: []byte("extension"),
				8: []byte("ibc2"),
			},
			wantErr: false,
		},
		{
			name: "field 8 not present - no change",
			input: map[int][]byte{
				1: []byte("field1"),
				7: []byte("extension"),
			},
			wantErr: false,
		},
		{
			name:    "empty protobuf",
			input:   map[int][]byte{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := createTestProtobuf(tt.input)

			result, err := ClearField8(input)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Verify field 8 doesn't exist
			fields, err := ParseProtobufFields(result)
			require.NoError(t, err)

			_, ok := fields[8]
			require.False(t, ok, "field 8 should not exist after clearing")

			// Verify other fields still exist
			for fieldNum := range tt.input {
				if fieldNum != 8 {
					_, ok := fields[fieldNum]
					require.True(t, ok, "field %d should still exist", fieldNum)
				}
			}
		})
	}
}

func TestGetFieldValue(t *testing.T) {
	tests := []struct {
		name      string
		input     map[int][]byte
		fieldNum  int
		wantValue []byte
		wantErr   bool
	}{
		{
			name: "field exists",
			input: map[int][]byte{
				7: []byte("extension-data"),
			},
			fieldNum:  7,
			wantValue: []byte("extension-data"),
			wantErr:   false,
		},
		{
			name: "field doesn't exist",
			input: map[int][]byte{
				7: []byte("extension"),
			},
			fieldNum:  8,
			wantValue: nil,
			wantErr:   false,
		},
		{
			name: "empty data",
			input: map[int][]byte{
				7: []byte(""),
			},
			fieldNum:  7,
			wantValue: []byte(""),
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := createTestProtobuf(tt.input)

			value, err := GetFieldValue(input, tt.fieldNum)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.wantValue, value)
		})
	}
}

func TestHasField(t *testing.T) {
	tests := []struct {
		name     string
		input    map[int][]byte
		fieldNum int
		want     bool
	}{
		{
			name: "field exists",
			input: map[int][]byte{
				7: []byte("data"),
			},
			fieldNum: 7,
			want:     true,
		},
		{
			name: "field doesn't exist",
			input: map[int][]byte{
				7: []byte("data"),
			},
			fieldNum: 8,
			want:     false,
		},
		{
			name:     "empty protobuf",
			input:    map[int][]byte{},
			fieldNum: 1,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := createTestProtobuf(tt.input)

			result := HasField(input, tt.fieldNum)
			require.Equal(t, tt.want, result)
		})
	}
}

func TestIsField8Empty(t *testing.T) {
	tests := []struct {
		name  string
		input map[int][]byte
		want  bool
	}{
		{
			name: "field 8 has data",
			input: map[int][]byte{
				8: []byte("data"),
			},
			want: false,
		},
		{
			name: "field 8 is empty",
			input: map[int][]byte{
				8: []byte(""),
			},
			want: true,
		},
		{
			name: "field 8 doesn't exist",
			input: map[int][]byte{
				7: []byte("data"),
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := createTestProtobuf(tt.input)

			result := IsField8Empty(input)
			require.Equal(t, tt.want, result)
		})
	}
}

func TestEncodeFieldTag(t *testing.T) {
	tests := []struct {
		name        string
		fieldNumber int
		wireType    int
	}{
		{"field 1 bytes", 1, WireBytes},
		{"field 7 bytes", 7, WireBytes},
		{"field 8 bytes", 8, WireBytes},
		{"field 1 varint", 1, WireVarint},
		{"field 100 bytes", 100, WireBytes},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tag := EncodeFieldTag(tt.fieldNumber, tt.wireType)
			require.NotEmpty(t, tag)

			// Decode and verify
			decoded, n := binary.Uvarint(tag)
			require.Greater(t, n, 0)

			decodedFieldNum := int(decoded >> 3)
			decodedWireType := int(decoded & 0x7)

			require.Equal(t, tt.fieldNumber, decodedFieldNum)
			require.Equal(t, tt.wireType, decodedWireType)
		})
	}
}

// Integration test: simulate the full migration process
func TestFullMigrationFlow(t *testing.T) {
	// Create a "broken" contract (SchemaBroken)
	brokenContract := createTestProtobuf(map[int][]byte{
		1: []byte("code_id"),
		2: []byte("creator"),
		7: []byte("ibc_port_id"), // Wrong - should be extension
		8: []byte("extension"),   // Wrong - should be empty
	})

	// Step 1: Swap fields 7 and 8
	afterSwap, err := SwapFields7And8(brokenContract)
	require.NoError(t, err)

	// Verify fields are swapped
	field7AfterSwap, err := GetFieldValue(afterSwap, 7)
	require.NoError(t, err)
	require.Equal(t, []byte("extension"), field7AfterSwap)

	field8AfterSwap, err := GetFieldValue(afterSwap, 8)
	require.NoError(t, err)
	require.Equal(t, []byte("ibc_port_id"), field8AfterSwap)

	// Step 2: Clear field 8 (IBCv2 never used)
	afterClear, err := ClearField8(afterSwap)
	require.NoError(t, err)

	// Verify field 8 is gone
	require.True(t, IsField8Empty(afterClear))

	// Verify field 7 still has extension
	field7Final, err := GetFieldValue(afterClear, 7)
	require.NoError(t, err)
	require.Equal(t, []byte("extension"), field7Final)

	// Verify other fields unchanged
	field1, err := GetFieldValue(afterClear, 1)
	require.NoError(t, err)
	require.Equal(t, []byte("code_id"), field1)
}
