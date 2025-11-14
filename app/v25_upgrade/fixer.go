package v25_upgrade

import (
	"encoding/binary"
	"fmt"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/gogo/protobuf/proto"
)

// FixContract attempts to fix a corrupted contract
// Returns the fixed data and whether the fix succeeded
func FixContract(address string, data []byte) ContractFixResult {
	result := ContractFixResult{
		Address:      address,
		OriginalData: data,
	}

	// Step 1: Analyze the contract
	analysis := AnalyzeContract(address, data)
	result.OriginalState = analysis.State
	result.FixStrategy = analysis.FixStrategy

	// Step 2: Check if fix is needed
	if analysis.State == StateHealthy {
		result.FixAttempted = false
		result.FixSucceeded = true // Already healthy = success
		result.FinalState = StateHealthy
		result.FixedData = data
		result.UnmarshalAfter = true
		return result
	}

	// StateSchemaInconsistent contracts can already unmarshal successfully
	// They don't NEED fixing - they're functional as-is
	// Only fix them if explicitly requested (future optional normalization)
	if analysis.State == StateSchemaInconsistent {
		result.FixAttempted = false
		result.FixSucceeded = true // Can unmarshal = functional
		result.FinalState = StateSchemaInconsistent
		result.FixedData = data
		result.UnmarshalAfter = true
		result.FixStrategy = "No fix needed - contract can unmarshal successfully (non-canonical schema is OK)"
		return result
	}

	if !analysis.Fixable {
		result.FixAttempted = false
		result.FixSucceeded = false
		result.FinalState = StateUnfixable
		result.Error = fmt.Errorf("contract is not fixable: %s", analysis.FixStrategy)
		return result
	}

	// Step 3: Attempt fix - only for StateUnmarshalFails
	result.FixAttempted = true

	var fixedData []byte
	var err error

	switch analysis.State {
	case StateUnmarshalFails:
		fixedData, err = fixCorruption(data, &analysis)

	default:
		err = fmt.Errorf("unknown state: %s", analysis.State)
	}

	if err != nil {
		result.FixSucceeded = false
		result.FinalState = StateUnfixable
		result.Error = fmt.Errorf("fix failed: %w", err)
		return result
	}

	// Step 4: Validate fix - attempt unmarshal
	var contractInfo wasmtypes.ContractInfo
	unmarshalErr := proto.Unmarshal(fixedData, &contractInfo)

	if unmarshalErr != nil {
		result.FixSucceeded = false
		result.FinalState = StateUnfixable
		result.Error = fmt.Errorf("fix applied but unmarshal still fails: %w", unmarshalErr)
		return result
	}

	// Step 5: Verify schema is now canonical
	finalState, _ := DetectContractState(fixedData)

	result.FixSucceeded = true
	result.FinalState = finalState
	result.FixedData = fixedData
	result.UnmarshalAfter = true

	// Warn if not fully healthy after fix
	if finalState != StateHealthy {
		result.Error = fmt.Errorf("contract fixed but not fully healthy: %s", finalState)
	}

	return result
}

// fixCorruption attempts to fix protobuf corruption
func fixCorruption(data []byte, analysis *ContractAnalysis) ([]byte, error) {
	switch analysis.CorruptionPattern {
	case PatternInvalidWireType:
		return fixInvalidWireType(data)

	case PatternTruncatedField:
		return fixTruncatedField(data)

	case PatternMalformedLength:
		return fixMalformedLength(data)

	case PatternMissingRequiredFields:
		return nil, fmt.Errorf("missing required fields - cannot auto-fix")

	case PatternDuplicateFields:
		return fixDuplicateFields(data)

	default:
		return nil, fmt.Errorf("unknown corruption pattern: %s", analysis.CorruptionPattern)
	}
}

// swapFields7And8 swaps field 7 and field 8 in protobuf data
func swapFields7And8(data []byte) ([]byte, error) {
	fields, err := ParseProtobufFields(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse protobuf: %w", err)
	}

	// Get fields 7 and 8
	field7, has7 := fields[7]
	field8, has8 := fields[8]

	if !has7 && !has8 {
		// Neither field exists - nothing to swap
		return data, nil
	}

	// Rebuild the protobuf with swapped fields
	result := make([]byte, 0, len(data))

	// Write all fields in order, swapping 7 and 8
	for fieldNum := 1; fieldNum <= 10; fieldNum++ {
		if fieldNum == 7 {
			// Write field 8's data with field number 7
			if has8 {
				result = append(result, EncodeFieldTag(7, field8.WireType)...)
				result = append(result, field8.Data...)
			}
		} else if fieldNum == 8 {
			// Write field 7's data with field number 8
			if has7 {
				result = append(result, EncodeFieldTag(8, field7.WireType)...)
				result = append(result, field7.Data...)
			}
		} else if field, ok := fields[fieldNum]; ok {
			// Write original field
			result = append(result, EncodeFieldTag(fieldNum, field.WireType)...)
			result = append(result, field.Data...)
		}
	}

	return result, nil
}

// addEmptyField7 adds an empty extension field at position 7
func addEmptyField7(data []byte) ([]byte, error) {
	fields, err := ParseProtobufFields(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse protobuf: %w", err)
	}

	// If field 7 already exists, nothing to do
	if _, has7 := fields[7]; has7 {
		return data, nil
	}

	// Create empty extension field (wire type 2 = length-delimited, length 0)
	emptyExtension := []byte{0} // Varint 0 = length 0

	// Rebuild the protobuf with field 7 inserted in correct position
	result := make([]byte, 0, len(data)+10)

	// Write all fields in order, inserting field 7
	for fieldNum := 1; fieldNum <= 10; fieldNum++ {
		if fieldNum == 7 {
			// Insert empty field 7
			result = append(result, EncodeFieldTag(7, WireBytes)...)
			result = append(result, emptyExtension...)
		}

		if field, ok := fields[fieldNum]; ok {
			// Write original field
			result = append(result, EncodeFieldTag(fieldNum, field.WireType)...)
			result = append(result, field.Data...)
		}
	}

	return result, nil
}

// ensureEmptyField8 ensures field 8 exists as an empty string
func ensureEmptyField8(data []byte) ([]byte, error) {
	fields, err := ParseProtobufFields(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse protobuf: %w", err)
	}

	// Check if field 8 already exists and is empty
	if field8, has8 := fields[8]; has8 {
		value, err := GetFieldValue(data, 8)
		if err == nil && len(value) == 0 && field8.WireType == WireBytes {
			// Field 8 already exists and is empty with correct wire type
			return data, nil
		}
		// Field 8 exists but has data or wrong wire type - we'll replace it
	}

	// Create empty string field (wire type 2 = length-delimited, length 0)
	emptyString := []byte{0} // Varint 0 = length 0

	// Rebuild the protobuf with field 8 as empty string
	result := make([]byte, 0, len(data)+10)

	// Write all fields in order, setting/adding field 8 as empty
	for fieldNum := 1; fieldNum <= 10; fieldNum++ {
		if fieldNum == 8 {
			// Set field 8 to empty string (wire type 2 for string)
			result = append(result, EncodeFieldTag(8, WireBytes)...)
			result = append(result, emptyString...)
		} else if field, ok := fields[fieldNum]; ok {
			// Write original field
			result = append(result, EncodeFieldTag(fieldNum, field.WireType)...)
			result = append(result, field.Data...)
		}
	}

	return result, nil
}

// fixInvalidWireType attempts to fix invalid wire types
// The "illegal wireType 7" error typically indicates fields 7 and 8 are swapped
// from the v20/v21 schema bug where:
// - Field 7 should be extension (google.protobuf.Any) but contains ibc_port_id data
// - Field 8 should be ibc2_port_id (string) but is missing
func fixInvalidWireType(data []byte) ([]byte, error) {
	// Strategy: Swap fields 7 and 8, then ensure proper schema

	// Step 1: Try to parse fields (this might fail due to wire type 7)
	fields, parseErr := ParseProtobufFields(data)

	if parseErr != nil {
		// Can't parse fields - try aggressive reconstruction
		return reconstructFromCorruption(data)
	}

	// Step 2: Check if we have the classic swap pattern
	// (field 7 exists with data that looks like ibc_port_id)
	field7, has7 := fields[7]
	_, has8 := fields[8]

	if has7 && !has8 {
		// Classic swap: field 7 has the ibc_port_id data
		// Solution:
		// 1. Move field 7 data to field 8 (as string)
		// 2. Add empty field 7 (as google.protobuf.Any)
		return swapAndNormalizeFields(fields, data)
	}

	// Step 3: If we have both fields but wrong wire types, fix them
	if has7 && has8 {
		// Check wire types
		if field7.WireType != WireBytes {
			// Field 7 should be WireBytes (length-delimited for Any type)
			return swapAndNormalizeFields(fields, data)
		}
	}

	// Step 4: Try the generic swap approach
	swapped, err := swapFields7And8(data)
	if err != nil {
		return nil, fmt.Errorf("failed to swap fields: %w", err)
	}

	// Step 5: Ensure both fields exist with correct types
	normalized, err := normalizeFields7And8(swapped)
	if err != nil {
		return nil, fmt.Errorf("failed to normalize after swap: %w", err)
	}

	return normalized, nil
}

// fixTruncatedField attempts to fix truncated fields
func fixTruncatedField(data []byte) ([]byte, error) {
	// Try to parse what we can and rebuild without truncated parts
	result := make([]byte, 0, len(data))
	offset := 0

	for offset < len(data) {
		// Read tag
		tag, n := binary.Uvarint(data[offset:])
		if n <= 0 {
			// Can't read tag - stop here
			break
		}

		_ = int(tag >> 3) // fieldNumber (not used here)
		wireType := int(tag & 0x7)

		// Try to read field data
		fieldData, bytesRead, err := readFieldData(data[offset+n:], wireType)
		if err != nil {
			// Truncated field - stop here
			break
		}

		// Valid field - copy it
		result = append(result, data[offset:offset+n]...)
		result = append(result, fieldData...)
		offset += n + bytesRead
	}

	return result, nil
}

// fixMalformedLength attempts to fix malformed length delimiters
func fixMalformedLength(data []byte) ([]byte, error) {
	// Similar to truncated field handling
	return fixTruncatedField(data)
}

// fixDuplicateFields removes duplicate field numbers (keeps first occurrence)
func fixDuplicateFields(data []byte) ([]byte, error) {
	seen := make(map[int]bool)
	result := make([]byte, 0, len(data))
	offset := 0

	for offset < len(data) {
		startOffset := offset

		// Read tag
		tag, n := binary.Uvarint(data[offset:])
		if n <= 0 {
			break
		}

		fieldNumber := int(tag >> 3)
		wireType := int(tag & 0x7)
		offset += n

		// Read field data
		fieldData, bytesRead, err := readFieldData(data[offset:], wireType)
		if err != nil {
			break
		}

		// Only include if we haven't seen this field before
		if !seen[fieldNumber] {
			result = append(result, data[startOffset:offset]...)
			result = append(result, fieldData...)
			seen[fieldNumber] = true
		}

		offset += bytesRead
	}

	return result, nil
}

// swapAndNormalizeFields swaps field 7 and 8, then normalizes them
func swapAndNormalizeFields(fields map[int]*ProtobufField, originalData []byte) ([]byte, error) {
	// Rebuild protobuf with fields 7 and 8 swapped
	result := make([]byte, 0, len(originalData)+10)

	// For a length-delimited field, Data already includes the length prefix
	// An empty field has Data = [0] (varint 0 for length 0)
	field7Data := []byte{0} // Empty extension with length prefix
	field8Data := []byte{0} // Empty string with length prefix

	// Extract field 7 data if it exists (move it to field 8)
	if field7, has7 := fields[7]; has7 {
		field8Data = field7.Data // Move field 7 data to field 8
	}

	// Write all fields in order
	for fieldNum := 1; fieldNum <= 10; fieldNum++ {
		if fieldNum == 7 {
			// Field 7: Write empty extension (google.protobuf.Any)
			result = append(result, EncodeFieldTag(7, WireBytes)...)
			result = append(result, field7Data...)
		} else if fieldNum == 8 {
			// Field 8: Write the data from old field 7 (ibc_port_id)
			result = append(result, EncodeFieldTag(8, WireBytes)...)
			result = append(result, field8Data...)
		} else if field, ok := fields[fieldNum]; ok {
			// Write other fields as-is
			result = append(result, EncodeFieldTag(fieldNum, field.WireType)...)
			result = append(result, field.Data...)
		}
	}

	return result, nil
}

// normalizeFields7And8 ensures both field 7 and 8 exist with correct types
func normalizeFields7And8(data []byte) ([]byte, error) {
	fields, err := ParseProtobufFields(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse fields: %w", err)
	}

	// Check what we need to add
	_, has7 := fields[7]
	_, has8 := fields[8]

	result := data

	// Add field 7 if missing
	if !has7 {
		result, err = addEmptyField7(result)
		if err != nil {
			return nil, fmt.Errorf("failed to add field 7: %w", err)
		}
	}

	// Add field 8 if missing
	if !has8 {
		result, err = ensureEmptyField8(result)
		if err != nil {
			return nil, fmt.Errorf("failed to add field 8: %w", err)
		}
	}

	return result, nil
}

// reconstructFromCorruption attempts to reconstruct contract from severely corrupted data
// This is called when even field parsing fails
func reconstructFromCorruption(data []byte) ([]byte, error) {
	// Try to extract whatever fields we can by being very permissive
	result := make([]byte, 0, len(data))
	offset := 0
	recoveredFields := make(map[int][]byte)

	for offset < len(data) {
		// Try to read a tag
		tag, n := binary.Uvarint(data[offset:])
		if n <= 0 {
			// Can't read tag - skip this byte and try next
			offset++
			continue
		}

		fieldNumber := int(tag >> 3)
		wireType := int(tag & 0x7)

		// Skip invalid wire types (6, 7)
		if wireType == 6 || wireType == 7 {
			offset += n
			// Try to skip some data (guess: skip next few bytes)
			// This is heuristic - we don't know the actual length
			offset += 10 // Skip 10 bytes as a guess
			if offset > len(data) {
				offset = len(data)
			}
			continue
		}

		// Try to read field data
		fieldData, bytesRead, err := readFieldData(data[offset+n:], wireType)
		if err != nil {
			// Failed to read - skip and continue
			offset += n + 1
			continue
		}

		// Successfully read a field
		recoveredFields[fieldNumber] = fieldData
		offset += n + bytesRead
	}

	// Rebuild from recovered fields
	for fieldNum := 1; fieldNum <= 10; fieldNum++ {
		if fieldNum == 7 {
			// Always add empty field 7
			result = append(result, EncodeFieldTag(7, WireBytes)...)
			result = append(result, 0) // Empty
		} else if fieldNum == 8 {
			// Add empty field 8
			result = append(result, EncodeFieldTag(8, WireBytes)...)
			result = append(result, 0) // Empty
		} else if fieldData, ok := recoveredFields[fieldNum]; ok {
			// Determine wire type based on field number
			wireType := WireBytes // Default to bytes for most fields
			if fieldNum == 1 {    // CodeID is varint
				wireType = WireVarint
			}
			result = append(result, EncodeFieldTag(fieldNum, wireType)...)
			result = append(result, fieldData...)
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("could not recover any fields from corrupted data")
	}

	return result, nil
}
