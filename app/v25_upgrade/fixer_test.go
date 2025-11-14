package v25_upgrade

import (
	"testing"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/require"
)

// TestFixHealthyContract verifies that healthy contracts are not modified
func TestFixHealthyContract(t *testing.T) {
	// Create a healthy contract
	contractInfo := &wasmtypes.ContractInfo{
		CodeID:  1,
		Creator: "xion1test",
		Admin:   "xion1admin",
		Label:   "test-contract",
	}

	data, err := proto.Marshal(contractInfo)
	require.NoError(t, err)

	// Attempt to fix it
	result := FixContract("xion1test", data)

	// Should not be modified (it's schema-inconsistent but can unmarshal, so we skip it)
	require.False(t, result.FixAttempted, "schema-inconsistent contract should not be fixed")
	require.True(t, result.FixSucceeded, "should be marked as success (can unmarshal)")
	require.Equal(t, StateSchemaInconsistent, result.OriginalState)
	require.Equal(t, StateSchemaInconsistent, result.FinalState)
	require.True(t, result.UnmarshalAfter)
	require.Equal(t, data, result.FixedData, "data should not change")
}

// TestFixInvalidWireType verifies field swapping for InvalidWireType corruption
func TestFixInvalidWireType(t *testing.T) {
	// Create a contract with swapped fields 7 and 8
	// Field 7 should be extension (google.protobuf.Any) but contains ibc_port_id
	// Field 8 should be ibc2_port_id (string) but is missing

	// Build corrupted protobuf manually
	corruptedData := []byte{}

	// Field 1: CodeID = 1 (varint)
	corruptedData = append(corruptedData, EncodeFieldTag(1, WireVarint)...)
	corruptedData = append(corruptedData, 1) // CodeID = 1

	// Field 2: Creator (string)
	creator := "xion1test"
	corruptedData = append(corruptedData, EncodeFieldTag(2, WireBytes)...)
	corruptedData = append(corruptedData, byte(len(creator)))
	corruptedData = append(corruptedData, []byte(creator)...)

	// Field 3: Admin (string)
	admin := "xion1admin"
	corruptedData = append(corruptedData, EncodeFieldTag(3, WireBytes)...)
	corruptedData = append(corruptedData, byte(len(admin)))
	corruptedData = append(corruptedData, []byte(admin)...)

	// Field 4: Label (string)
	label := "test-contract"
	corruptedData = append(corruptedData, EncodeFieldTag(4, WireBytes)...)
	corruptedData = append(corruptedData, byte(len(label)))
	corruptedData = append(corruptedData, []byte(label)...)

	// Field 7: Contains ibc_port_id data (WRONG - this should be field 8)
	portID := "wasm.xion1test"
	corruptedData = append(corruptedData, EncodeFieldTag(7, WireBytes)...)
	corruptedData = append(corruptedData, byte(len(portID)))
	corruptedData = append(corruptedData, []byte(portID)...)

	// Field 8 is missing (WRONG - this should contain the port ID or be empty)

	// Verify this data cannot unmarshal due to wire type issue
	var testInfo wasmtypes.ContractInfo
	unmarshalErr := proto.Unmarshal(corruptedData, &testInfo)
	require.Error(t, unmarshalErr, "corrupted data should fail to unmarshal")

	// Attempt to fix it
	result := FixContract("xion1test", corruptedData)

	// Should be fixed
	require.True(t, result.FixAttempted, "corrupted contract should be fixed")
	require.True(t, result.FixSucceeded, "fix should succeed")
	require.Equal(t, StateUnmarshalFails, result.OriginalState)
	require.True(t, result.UnmarshalAfter, "fixed data should unmarshal")

	// Verify fixed data can unmarshal
	var fixedInfo wasmtypes.ContractInfo
	err := proto.Unmarshal(result.FixedData, &fixedInfo)
	require.NoError(t, err, "fixed data should unmarshal successfully")

	// Verify the data was preserved correctly
	require.Equal(t, uint64(1), fixedInfo.CodeID)
	require.Equal(t, creator, fixedInfo.Creator)
	require.Equal(t, admin, fixedInfo.Admin)
	require.Equal(t, label, fixedInfo.Label)
}

// TestFixEmptyFieldEncoding verifies that empty fields are encoded correctly
func TestFixEmptyFieldEncoding(t *testing.T) {
	// Create fields map with field 7 having data
	fields := map[int]*ProtobufField{
		1: {Number: 1, WireType: WireVarint, Data: []byte{1}},
		7: {Number: 7, WireType: WireBytes, Data: append([]byte{12}, []byte("wasm.xion123")...)},
	}

	// Swap and normalize
	fixed, err := swapAndNormalizeFields(fields, []byte{})
	require.NoError(t, err)

	// Parse fixed data
	fixedFields, err := ParseProtobufFields(fixed)
	require.NoError(t, err)

	// Verify field 7 is now empty with correct encoding
	_, has7 := fixedFields[7]
	require.True(t, has7, "field 7 should exist")
	require.Equal(t, WireBytes, fixedFields[7].WireType)
	require.Equal(t, []byte{0}, fixedFields[7].Data, "field 7 should be empty with length prefix [0]")

	// Verify field 8 now contains the original field 7 data
	_, has8 := fixedFields[8]
	require.True(t, has8, "field 8 should exist")
	require.Equal(t, WireBytes, fixedFields[8].WireType)
	require.Equal(t, append([]byte{12}, []byte("wasm.xion123")...), fixedFields[8].Data, "field 8 should contain swapped data")
}

// TestFixSchemaInconsistent verifies that schema-inconsistent contracts can unmarshal
func TestFixSchemaInconsistent(t *testing.T) {
	// Create a contract missing field 7 and 8 (but can still unmarshal)
	// This is schema-inconsistent but functional

	corruptedData := []byte{}

	// Field 1: CodeID = 1 (varint)
	corruptedData = append(corruptedData, EncodeFieldTag(1, WireVarint)...)
	corruptedData = append(corruptedData, 1)

	// Field 2: Creator (string)
	creator := "xion1test"
	corruptedData = append(corruptedData, EncodeFieldTag(2, WireBytes)...)
	corruptedData = append(corruptedData, byte(len(creator)))
	corruptedData = append(corruptedData, []byte(creator)...)

	// Skip fields 7 and 8 - this makes it schema-inconsistent

	// This should still unmarshal (protobuf is lenient with missing fields)
	var testInfo wasmtypes.ContractInfo
	unmarshalErr := proto.Unmarshal(corruptedData, &testInfo)
	require.NoError(t, unmarshalErr, "schema-inconsistent data should still unmarshal")

	// Attempt to fix it
	result := FixContract("xion1test", corruptedData)

	// Should NOT be fixed (it already works)
	require.False(t, result.FixAttempted, "schema-inconsistent contract should not be fixed")
	require.True(t, result.FixSucceeded, "should be marked as success (can unmarshal)")
	require.Equal(t, StateSchemaInconsistent, result.OriginalState)
	require.Equal(t, StateSchemaInconsistent, result.FinalState)
	require.True(t, result.UnmarshalAfter)
}

// TestFixDuplicateFields verifies duplicate field handling
func TestFixDuplicateFields(t *testing.T) {
	// Create data with duplicate field 1
	data := []byte{}

	// Field 1: CodeID = 1
	data = append(data, EncodeFieldTag(1, WireVarint)...)
	data = append(data, 1)

	// Field 1 again: CodeID = 2 (duplicate!)
	data = append(data, EncodeFieldTag(1, WireVarint)...)
	data = append(data, 2)

	// Field 2: Creator
	creator := "xion1test"
	data = append(data, EncodeFieldTag(2, WireBytes)...)
	data = append(data, byte(len(creator)))
	data = append(data, []byte(creator)...)

	// Fix duplicate fields
	fixed, err := fixDuplicateFields(data)
	require.NoError(t, err)

	// Parse fixed data
	fields, err := ParseProtobufFields(fixed)
	require.NoError(t, err)

	// Should only have one field 1 (first occurrence)
	field1, has1 := fields[1]
	require.True(t, has1)
	require.Equal(t, []byte{1}, field1.Data, "should keep first occurrence of duplicate field")
}

// TestSwapFields7And8 verifies basic field swapping
func TestSwapFields7And8(t *testing.T) {
	// Create data with both field 7 and field 8
	data := []byte{}

	// Field 7: Contains "field7data"
	field7Val := "field7data"
	data = append(data, EncodeFieldTag(7, WireBytes)...)
	data = append(data, byte(len(field7Val)))
	data = append(data, []byte(field7Val)...)

	// Field 8: Contains "field8data"
	field8Val := "field8data"
	data = append(data, EncodeFieldTag(8, WireBytes)...)
	data = append(data, byte(len(field8Val)))
	data = append(data, []byte(field8Val)...)

	// Swap them
	swapped, err := swapFields7And8(data)
	require.NoError(t, err)

	// Parse swapped data
	fields, err := ParseProtobufFields(swapped)
	require.NoError(t, err)

	// Verify fields are swapped
	_, has7 := fields[7]
	require.True(t, has7)
	field7Data, err := GetFieldValue(swapped, 7)
	require.NoError(t, err)
	require.Equal(t, []byte(field8Val), field7Data, "field 7 should now contain field 8 data")

	_, has8 := fields[8]
	require.True(t, has8)
	field8Data, err := GetFieldValue(swapped, 8)
	require.NoError(t, err)
	require.Equal(t, []byte(field7Val), field8Data, "field 8 should now contain field 7 data")
}

// TestAddEmptyField7 verifies adding empty field 7
func TestAddEmptyField7(t *testing.T) {
	// Create data without field 7
	data := []byte{}

	// Field 1: CodeID = 1
	data = append(data, EncodeFieldTag(1, WireVarint)...)
	data = append(data, 1)

	// Field 2: Creator
	creator := "xion1test"
	data = append(data, EncodeFieldTag(2, WireBytes)...)
	data = append(data, byte(len(creator)))
	data = append(data, []byte(creator)...)

	// Add empty field 7
	withField7, err := addEmptyField7(data)
	require.NoError(t, err)

	// Parse result
	fields, err := ParseProtobufFields(withField7)
	require.NoError(t, err)

	// Verify field 7 exists and is empty
	field7, has7 := fields[7]
	require.True(t, has7, "field 7 should exist")
	require.Equal(t, WireBytes, field7.WireType)
	require.Equal(t, []byte{0}, field7.Data, "field 7 should be empty with length prefix")
}

// TestEnsureEmptyField8 verifies adding/ensuring empty field 8
func TestEnsureEmptyField8(t *testing.T) {
	// Create data without field 8
	data := []byte{}

	// Field 1: CodeID = 1
	data = append(data, EncodeFieldTag(1, WireVarint)...)
	data = append(data, 1)

	// Ensure empty field 8
	withField8, err := ensureEmptyField8(data)
	require.NoError(t, err)

	// Parse result
	fields, err := ParseProtobufFields(withField8)
	require.NoError(t, err)

	// Verify field 8 exists and is empty
	field8, has8 := fields[8]
	require.True(t, has8, "field 8 should exist")
	require.Equal(t, WireBytes, field8.WireType)
	require.Equal(t, []byte{0}, field8.Data, "field 8 should be empty with length prefix")
}

// TestNormalizeFields7And8 verifies both fields are added if missing
func TestNormalizeFields7And8(t *testing.T) {
	// Create data without fields 7 and 8
	data := []byte{}

	// Field 1: CodeID = 1
	data = append(data, EncodeFieldTag(1, WireVarint)...)
	data = append(data, 1)

	// Normalize
	normalized, err := normalizeFields7And8(data)
	require.NoError(t, err)

	// Parse result
	fields, err := ParseProtobufFields(normalized)
	require.NoError(t, err)

	// Verify both fields exist
	_, has7 := fields[7]
	require.True(t, has7, "field 7 should exist")

	_, has8 := fields[8]
	require.True(t, has8, "field 8 should exist")
}

// TestFixContractRoundTrip verifies that fixed contracts can be marshaled and unmarshaled
func TestFixContractRoundTrip(t *testing.T) {
	// Create a corrupted contract
	corruptedData := []byte{}

	// Field 1: CodeID = 1
	corruptedData = append(corruptedData, EncodeFieldTag(1, WireVarint)...)
	corruptedData = append(corruptedData, 1)

	// Field 2: Creator
	creator := "xion1test"
	corruptedData = append(corruptedData, EncodeFieldTag(2, WireBytes)...)
	corruptedData = append(corruptedData, byte(len(creator)))
	corruptedData = append(corruptedData, []byte(creator)...)

	// Field 7: Contains port ID (wrong)
	portID := "wasm.xion1test"
	corruptedData = append(corruptedData, EncodeFieldTag(7, WireBytes)...)
	corruptedData = append(corruptedData, byte(len(portID)))
	corruptedData = append(corruptedData, []byte(portID)...)

	// Fix it
	result := FixContract("xion1test", corruptedData)
	require.True(t, result.FixSucceeded)

	// Unmarshal fixed data
	var contractInfo wasmtypes.ContractInfo
	err := proto.Unmarshal(result.FixedData, &contractInfo)
	require.NoError(t, err)

	// Marshal it again
	remarshaled, err := proto.Marshal(&contractInfo)
	require.NoError(t, err)

	// Unmarshal the remarshaled data
	var contractInfo2 wasmtypes.ContractInfo
	err = proto.Unmarshal(remarshaled, &contractInfo2)
	require.NoError(t, err)

	// Verify data integrity
	require.Equal(t, contractInfo.CodeID, contractInfo2.CodeID)
	require.Equal(t, contractInfo.Creator, contractInfo2.Creator)
}
