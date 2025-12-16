package v25_upgrade

import (
	"testing"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/require"
)

// TestDetectHealthyContract verifies detection of healthy contracts
func TestDetectHealthyContract(t *testing.T) {
	// Create a healthy contract with all required fields including 7 and 8
	data := []byte{}

	// Field 1: CodeID = 1
	data = append(data, EncodeFieldTag(1, WireVarint)...)
	data = append(data, 1)

	// Field 2: Creator
	creator := "xion1test"
	data = append(data, EncodeFieldTag(2, WireBytes)...)
	data = append(data, byte(len(creator)))
	data = append(data, []byte(creator)...)

	// Field 7: Empty extension
	data = append(data, EncodeFieldTag(7, WireBytes)...)
	data = append(data, 0)

	// Field 8: Empty ibc2_port_id
	data = append(data, EncodeFieldTag(8, WireBytes)...)
	data = append(data, 0)

	// Detect state
	state, err := DetectContractState(data)
	require.NoError(t, err)
	require.Equal(t, StateHealthy, state)
}

// TestDetectCorruptedContract verifies detection of unmarshalable contracts
func TestDetectCorruptedContract(t *testing.T) {
	// Create corrupted data with invalid wire type
	corruptedData := []byte{}

	// Field 1: CodeID = 1
	corruptedData = append(corruptedData, EncodeFieldTag(1, WireVarint)...)
	corruptedData = append(corruptedData, 1)

	// Field 7: Invalid wire type 7 (illegal)
	corruptedData = append(corruptedData, EncodeFieldTag(7, 7)...)
	corruptedData = append(corruptedData, 0xFF, 0xFF) // Some invalid data

	// Detect state
	state, _ := DetectContractState(corruptedData)
	require.Equal(t, StateUnmarshalFails, state)

	// Also verify with DetectCorruption function
	isCorrupted := DetectCorruption(corruptedData)
	require.True(t, isCorrupted, "should detect corruption")
}

// TestDetectSchemaInconsistent verifies detection of schema-inconsistent contracts
func TestDetectSchemaInconsistent(t *testing.T) {
	// Create a contract missing field 7 (but can still unmarshal)
	data := []byte{}

	// Field 1: CodeID = 1
	data = append(data, EncodeFieldTag(1, WireVarint)...)
	data = append(data, 1)

	// Field 2: Creator
	creator := "xion1test"
	data = append(data, EncodeFieldTag(2, WireBytes)...)
	data = append(data, byte(len(creator)))
	data = append(data, []byte(creator)...)

	// Skip fields 7 and 8 - makes it schema-inconsistent

	// Verify it can still unmarshal (protobuf is lenient)
	var contractInfo wasmtypes.ContractInfo
	err := proto.Unmarshal(data, &contractInfo)
	require.NoError(t, err, "should unmarshal despite missing fields")

	// Detect state
	state, err := DetectContractState(data)
	require.NoError(t, err)
	require.Equal(t, StateSchemaInconsistent, state, "should detect schema inconsistency")
}

// TestCanUnmarshal verifies the CanUnmarshal helper
func TestCanUnmarshal(t *testing.T) {
	// Healthy contract
	contractInfo := &wasmtypes.ContractInfo{
		CodeID:  1,
		Creator: "xion1test",
	}
	data, err := proto.Marshal(contractInfo)
	require.NoError(t, err)

	canUnmarshal, err := CanUnmarshal(data)
	require.NoError(t, err)
	require.True(t, canUnmarshal, "healthy contract should unmarshal")

	// Corrupted contract
	corruptedData := []byte{0xFF, 0xFF, 0xFF, 0xFF}
	canUnmarshal, err = CanUnmarshal(corruptedData)
	require.Error(t, err)
	require.False(t, canUnmarshal, "corrupted data should not unmarshal")
}

// TestUnmarshalContract verifies the UnmarshalContract helper
func TestUnmarshalContract(t *testing.T) {
	// Create contract
	originalInfo := &wasmtypes.ContractInfo{
		CodeID:  42,
		Creator: "xion1creator",
		Admin:   "xion1admin",
		Label:   "my-contract",
	}

	data, err := proto.Marshal(originalInfo)
	require.NoError(t, err)

	// Unmarshal it
	contractInfo, err := UnmarshalContract(data)
	require.NoError(t, err)
	require.NotNil(t, contractInfo)
	require.Equal(t, uint64(42), contractInfo.CodeID)
	require.Equal(t, "xion1creator", contractInfo.Creator)
	require.Equal(t, "xion1admin", contractInfo.Admin)
	require.Equal(t, "my-contract", contractInfo.Label)
}

// TestValidateUnmarshal verifies round-trip marshal/unmarshal validation
func TestValidateUnmarshal(t *testing.T) {
	// Create healthy contract
	contractInfo := &wasmtypes.ContractInfo{
		CodeID:  1,
		Creator: "xion1test",
		Admin:   "xion1admin",
		Label:   "test",
	}

	data, err := proto.Marshal(contractInfo)
	require.NoError(t, err)

	// Validate round-trip
	err = ValidateUnmarshal(data)
	require.NoError(t, err, "round-trip should succeed for healthy contract")

	// Test with corrupted data
	corruptedData := []byte{0xFF, 0xFF, 0xFF}
	err = ValidateUnmarshal(corruptedData)
	require.Error(t, err, "round-trip should fail for corrupted data")
}

// TestNeedsRepair verifies the NeedsRepair helper
func TestNeedsRepair(t *testing.T) {
	require.True(t, NeedsRepair(StateUnmarshalFails), "StateUnmarshalFails needs repair")
	require.True(t, NeedsRepair(StateSchemaInconsistent), "StateSchemaInconsistent needs repair")
	require.False(t, NeedsRepair(StateHealthy), "StateHealthy does not need repair")
	require.False(t, NeedsRepair(StateUnfixable), "StateUnfixable cannot be repaired")
}

// TestIsHealthy verifies the IsHealthy helper
func TestIsHealthy(t *testing.T) {
	require.True(t, IsHealthy(StateHealthy))
	require.False(t, IsHealthy(StateUnmarshalFails))
	require.False(t, IsHealthy(StateSchemaInconsistent))
	require.False(t, IsHealthy(StateUnfixable))
}

// TestIsFixable verifies the IsFixable helper
func TestIsFixable(t *testing.T) {
	require.True(t, IsFixable(StateUnmarshalFails), "StateUnmarshalFails should be fixable")
	require.True(t, IsFixable(StateSchemaInconsistent), "StateSchemaInconsistent should be fixable")
	require.False(t, IsFixable(StateHealthy), "StateHealthy doesn't need fixing")
	require.False(t, IsFixable(StateUnfixable), "StateUnfixable cannot be fixed")
}

// TestGetRepairAction verifies repair action descriptions
func TestGetRepairAction(t *testing.T) {
	action := GetRepairAction(StateHealthy)
	require.Contains(t, action, "already healthy")

	action = GetRepairAction(StateUnmarshalFails)
	require.Contains(t, action, "corruption")

	action = GetRepairAction(StateSchemaInconsistent)
	require.Contains(t, action, "Normalize schema")

	action = GetRepairAction(StateUnfixable)
	require.Contains(t, action, "Cannot fix")
}

// TestDetectWithField8Data verifies detection when field 8 has legitimate data
func TestDetectWithField8Data(t *testing.T) {
	// Create contract with field 8 containing ibc2_port_id data
	// This is LEGITIMATE and should be marked healthy
	data := []byte{}

	// Field 1: CodeID = 1
	data = append(data, EncodeFieldTag(1, WireVarint)...)
	data = append(data, 1)

	// Field 2: Creator
	creator := "xion1test"
	data = append(data, EncodeFieldTag(2, WireBytes)...)
	data = append(data, byte(len(creator)))
	data = append(data, []byte(creator)...)

	// Field 7: Empty extension (correct)
	data = append(data, EncodeFieldTag(7, WireBytes)...)
	data = append(data, 0) // Empty

	// Field 8: Contains legitimate ibc2_port_id data
	portID := "wasm.xion1contract"
	data = append(data, EncodeFieldTag(8, WireBytes)...)
	data = append(data, byte(len(portID)))
	data = append(data, []byte(portID)...)

	// Should unmarshal successfully
	var contractInfo wasmtypes.ContractInfo
	err := proto.Unmarshal(data, &contractInfo)
	require.NoError(t, err, "contract with field 8 data should unmarshal")

	// Should be detected as healthy
	state, err := DetectContractState(data)
	require.NoError(t, err)
	require.Equal(t, StateHealthy, state, "contract with field 8 data should be healthy")

	// Should NOT be marked as corrupted
	isCorrupted := DetectCorruption(data)
	require.False(t, isCorrupted, "contract with field 8 data is not corrupted")
}

// TestDetectMissingBothFields verifies detection when both fields 7 and 8 are missing
func TestDetectMissingBothFields(t *testing.T) {
	// Create contract missing both fields
	data := []byte{}

	// Field 1: CodeID = 1
	data = append(data, EncodeFieldTag(1, WireVarint)...)
	data = append(data, 1)

	// Field 2: Creator
	creator := "xion1test"
	data = append(data, EncodeFieldTag(2, WireBytes)...)
	data = append(data, byte(len(creator)))
	data = append(data, []byte(creator)...)

	// No fields 7 or 8

	// Should still unmarshal (protobuf is lenient with missing fields)
	var contractInfo wasmtypes.ContractInfo
	err := proto.Unmarshal(data, &contractInfo)
	require.NoError(t, err)

	// But should be detected as schema-inconsistent
	state, err := DetectContractState(data)
	require.NoError(t, err)
	require.Equal(t, StateSchemaInconsistent, state)
}

// TestDetectWrongWireType verifies detection of wrong wire type for field 7
func TestDetectWrongWireType(t *testing.T) {
	// Create contract with field 7 having wrong wire type
	data := []byte{}

	// Field 1: CodeID = 1
	data = append(data, EncodeFieldTag(1, WireVarint)...)
	data = append(data, 1)

	// Field 7: Using WireVarint instead of WireBytes (wrong!)
	data = append(data, EncodeFieldTag(7, WireVarint)...)
	data = append(data, 0)

	// Field 8: Correct
	data = append(data, EncodeFieldTag(8, WireBytes)...)
	data = append(data, 0)

	// Might still unmarshal depending on how lenient protobuf is
	var contractInfo wasmtypes.ContractInfo
	unmarshalErr := proto.Unmarshal(data, &contractInfo)

	// Detect state
	state, _ := DetectContractState(data)

	if unmarshalErr != nil {
		// Cannot unmarshal - corrupted
		require.Equal(t, StateUnmarshalFails, state)
	} else {
		// Can unmarshal but wrong wire type - schema inconsistent
		require.Equal(t, StateSchemaInconsistent, state)
	}
}
