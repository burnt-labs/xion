package v25_upgrade

import (
	"fmt"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/gogo/protobuf/proto"
)

// DetectContractState determines the state of a contract using unmarshal-based detection
//
// This is the CORE difference from v24: we actually try to unmarshal the contract
// as wasmtypes.ContractInfo, which is the only test that matters for chain operation.
//
// Detection flow:
// 1. Try proto.Unmarshal(data, &ContractInfo)
// 2. If unmarshal fails → StateUnmarshalFails (corruption)
// 3. If unmarshal succeeds → check schema consistency
// 4. If schema is canonical → StateHealthy
// 5. If schema needs normalization → StateSchemaInconsistent
func DetectContractState(data []byte) (ContractState, error) {
	// Step 1: Try to unmarshal as ContractInfo
	// This is the ONLY test that matters - can the chain actually use this contract?
	var contractInfo wasmtypes.ContractInfo
	unmarshalErr := proto.Unmarshal(data, &contractInfo)

	if unmarshalErr != nil {
		// Failed to unmarshal - this is corruption
		// We need to analyze and repair
		return StateUnmarshalFails, unmarshalErr
	}

	// Step 2: Unmarshal succeeded - contract is usable
	// Now check if schema is in canonical form

	// Try to parse protobuf fields for schema analysis
	fields, parseErr := ParseProtobufFields(data)
	if parseErr != nil {
		// Can unmarshal but can't parse fields manually?
		// This is unusual but means the contract is usable
		// Consider it healthy since unmarshal worked
		return StateHealthy, nil
	}

	// Step 3: Check field 7 and field 8 layout
	field7, hasField7 := fields[7]
	_, hasField8 := fields[8]

	// Canonical schema requires:
	// - Field 7 present (extension) with WireBytes type
	// - Field 8 present (ibc2_port_id)
	// Note: Field 8 can be empty OR have data - both are valid!
	//
	// The v20/v21 bug was that it SWAPPED fields 7 and 8, putting:
	// - extension at position 8 (wrong)
	// - ibc_port_id at position 7 (wrong)
	//
	// So we need to detect if field 7 contains the WRONG type of data
	// (i.e., if it looks like ibc_port_id instead of extension)

	if !hasField7 {
		// Missing field 7 - schema inconsistent
		return StateSchemaInconsistent, nil
	}

	if !hasField8 {
		// Missing field 8 - schema inconsistent
		return StateSchemaInconsistent, nil
	}

	// Check field 7 wire type (should be WireBytes for extension)
	if field7.WireType != WireBytes {
		// Wrong wire type for field 7 - schema inconsistent
		return StateSchemaInconsistent, nil
	}

	// All checks passed - canonical schema
	return StateHealthy, nil
}

// DetectCorruption is a quick check if contract data is corrupted
// Returns true if corrupted (unmarshal fails)
func DetectCorruption(data []byte) bool {
	var contractInfo wasmtypes.ContractInfo
	err := proto.Unmarshal(data, &contractInfo)
	return err != nil
}

// CanUnmarshal checks if data can be unmarshaled as ContractInfo
func CanUnmarshal(data []byte) (bool, error) {
	var contractInfo wasmtypes.ContractInfo
	err := proto.Unmarshal(data, &contractInfo)
	if err != nil {
		return false, err
	}
	return true, nil
}

// UnmarshalContract attempts to unmarshal contract data
// Returns the ContractInfo if successful, error otherwise
func UnmarshalContract(data []byte) (*wasmtypes.ContractInfo, error) {
	var contractInfo wasmtypes.ContractInfo
	err := proto.Unmarshal(data, &contractInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal contract: %w", err)
	}
	return &contractInfo, nil
}

// ValidateUnmarshal validates that data can be unmarshaled and re-marshaled correctly
// This ensures round-trip compatibility
func ValidateUnmarshal(data []byte) error {
	// Unmarshal
	contractInfo, err := UnmarshalContract(data)
	if err != nil {
		return fmt.Errorf("unmarshal failed: %w", err)
	}

	// Marshal back
	remarshaled, err := proto.Marshal(contractInfo)
	if err != nil {
		return fmt.Errorf("re-marshal failed: %w", err)
	}

	// Verify we can unmarshal the remarshaled data
	var contractInfo2 wasmtypes.ContractInfo
	err = proto.Unmarshal(remarshaled, &contractInfo2)
	if err != nil {
		return fmt.Errorf("re-unmarshal failed: %w", err)
	}

	return nil
}

// NeedsRepair determines if a contract needs repair based on its state
func NeedsRepair(state ContractState) bool {
	switch state {
	case StateUnmarshalFails:
		// Definitely needs repair
		return true
	case StateSchemaInconsistent:
		// Needs schema normalization
		return true
	case StateHealthy:
		// Already good
		return false
	case StateUnfixable:
		// Cannot be repaired
		return false
	default:
		return false
	}
}

// IsHealthy checks if a contract is in healthy state
func IsHealthy(state ContractState) bool {
	return state == StateHealthy
}

// IsFixable determines if a contract state can potentially be fixed
func IsFixable(state ContractState) bool {
	switch state {
	case StateUnmarshalFails:
		// Might be fixable with pattern analysis
		return true
	case StateSchemaInconsistent:
		// Definitely fixable with schema normalization
		return true
	case StateUnfixable:
		// Cannot fix
		return false
	case StateHealthy:
		// Already healthy, no fix needed
		return false
	default:
		return false
	}
}

// GetRepairAction returns a human-readable description of what repair is needed
func GetRepairAction(state ContractState) string {
	switch state {
	case StateHealthy:
		return "None - already healthy"
	case StateUnmarshalFails:
		return "Analyze corruption patterns and attempt targeted repair"
	case StateSchemaInconsistent:
		return "Normalize schema (add missing fields or fix field positions)"
	case StateUnfixable:
		return "Cannot fix - manual intervention required"
	default:
		return "Unknown - manual inspection required"
	}
}
