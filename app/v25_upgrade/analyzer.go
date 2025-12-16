package v25_upgrade

import (
	"encoding/binary"
	"fmt"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/gogo/protobuf/proto"
)

// AnalyzeContract performs deep analysis of a contract
// This is used to understand corruption and determine fix strategies
func AnalyzeContract(address string, data []byte) ContractAnalysis {
	analysis := ContractAnalysis{
		Address:     address,
		RawDataSize: len(data),
		RawDataHex:  HexDump(data, 200), // First 200 bytes in hex
	}

	// Step 1: Try to unmarshal
	var contractInfo wasmtypes.ContractInfo
	unmarshalErr := proto.Unmarshal(data, &contractInfo)

	if unmarshalErr != nil {
		// Unmarshal failed - corrupted
		analysis.State = StateUnmarshalFails
		analysis.UnmarshalError = unmarshalErr
		analysis.Fixable = true // Potentially fixable with pattern analysis

		// Analyze corruption patterns
		pattern, fixStrategy := analyzeCorruptionPatterns(data, unmarshalErr)
		analysis.CorruptionPattern = pattern
		analysis.FixStrategy = fixStrategy

		// Try to parse protobuf fields to get more info
		fields, parseErr := ParseProtobufFields(data)
		if parseErr != nil {
			// Can't even parse fields - severe corruption
			analysis.Fixable = false
			analysis.FixStrategy = "Severe corruption - cannot parse protobuf fields"
		} else {
			// Can parse fields - analyze them
			analyzeFields(&analysis, fields, data)
		}

		return analysis
	}

	// Step 2: Unmarshal succeeded - extract contract info
	analysis.CodeID = contractInfo.CodeID
	analysis.Creator = contractInfo.Creator
	analysis.Admin = contractInfo.Admin
	analysis.Label = contractInfo.Label

	// Step 3: Check schema consistency
	fields, parseErr := ParseProtobufFields(data)
	if parseErr != nil {
		// Can unmarshal but can't parse - unusual but OK
		analysis.State = StateHealthy
		analysis.Fixable = false
		return analysis
	}

	analyzeFields(&analysis, fields, data)

	// Determine state based on field analysis
	analysis.State = determineStateFromFields(&analysis)

	// Set fix strategy if needed
	switch analysis.State {
	case StateSchemaInconsistent:
		analysis.Fixable = true
		analysis.FixStrategy = determineSchemaFixStrategy(&analysis)
	case StateHealthy:
		analysis.Fixable = false
		analysis.FixStrategy = "No fix needed - already healthy"
	}

	return analysis
}

// analyzeFields extracts field information
func analyzeFields(analysis *ContractAnalysis, fields map[int]*ProtobufField, data []byte) {
	// Check field 7
	if field7, ok := fields[7]; ok {
		analysis.HasField7 = true
		analysis.Field7WireType = field7.WireType
	}

	// Check field 8
	if field8, ok := fields[8]; ok {
		analysis.HasField8 = true
		analysis.Field8WireType = field8.WireType

		// Check if field 8 has data
		value, err := GetFieldValue(data, 8)
		if err == nil && len(value) > 0 {
			analysis.Field8HasData = true
		}
	}
}

// determineStateFromFields determines contract state based on field analysis
func determineStateFromFields(analysis *ContractAnalysis) ContractState {
	// If unmarshal succeeded and schema is canonical → Healthy
	// Otherwise → SchemaInconsistent

	// Canonical schema requires:
	// - Field 7 present (extension) with WireBytes type
	// - Field 8 present (ibc2_port_id)
	// Note: Field 8 can be empty OR have data - both are valid!

	if !analysis.HasField7 {
		return StateSchemaInconsistent
	}

	if !analysis.HasField8 {
		return StateSchemaInconsistent
	}

	if analysis.Field7WireType != WireBytes {
		return StateSchemaInconsistent
	}

	// If we got here and unmarshal succeeded (which it did if we're calling this function
	// with unmarshal success), then the contract is healthy
	return StateHealthy
}

// determineSchemaFixStrategy determines the fix strategy for schema inconsistency
func determineSchemaFixStrategy(analysis *ContractAnalysis) string {
	// Note: We should only get here if unmarshal FAILED or fields are missing
	// If unmarshal succeeded but we're here, it means fields are missing or wrong wire type

	if !analysis.HasField7 && !analysis.HasField8 {
		return "Add empty field 7 (extension) and field 8 (ibc2_port_id)"
	}

	if !analysis.HasField7 {
		return "Add empty field 7 (extension)"
	}

	if !analysis.HasField8 {
		return "Add empty field 8 (ibc2_port_id)"
	}

	if analysis.Field7WireType != WireBytes {
		return "Fix field 7 wire type (should be WireBytes)"
	}

	return "Unknown schema fix required"
}

// analyzeCorruptionPatterns analyzes corruption patterns when unmarshal fails
func analyzeCorruptionPatterns(data []byte, unmarshalErr error) (CorruptionPattern, string) {
	// First check the unmarshal error message for known patterns
	unmarshalErrMsg := unmarshalErr.Error()

	// Check for illegal/invalid wire type in unmarshal error
	if contains(unmarshalErrMsg, "illegal wireType") || contains(unmarshalErrMsg, "invalid wire type") {
		return PatternInvalidWireType, fmt.Sprintf("Illegal wire type detected - %s", unmarshalErrMsg)
	}

	if contains(unmarshalErrMsg, "unexpected EOF") || contains(unmarshalErrMsg, "truncated") {
		return PatternTruncatedField, "Data appears truncated or incomplete"
	}

	// Try to parse protobuf fields
	fields, parseErr := ParseProtobufFields(data)

	if parseErr != nil {
		// Can't parse fields at all - check for specific patterns in error
		errMsg := parseErr.Error()

		if contains(errMsg, "invalid wire type") || contains(errMsg, "illegal wireType") {
			return PatternInvalidWireType, "Remove or fix fields with invalid wire types (6, 7)"
		}

		if contains(errMsg, "unexpected EOF") || contains(errMsg, "truncated") {
			return PatternTruncatedField, "Attempt to reconstruct truncated fields or remove them"
		}

		if contains(errMsg, "failed to read length") {
			return PatternMalformedLength, "Fix malformed length delimiters"
		}

		return PatternUnknown, fmt.Sprintf("Severe corruption - cannot parse fields: %s", errMsg)
	}

	// Can parse fields but unmarshal fails
	// This might be due to field content issues

	// Check for invalid wire types in fields
	for _, field := range fields {
		if field.WireType == 6 || field.WireType == 7 {
			return PatternInvalidWireType, fmt.Sprintf("Remove field %d with invalid wire type %d", field.Number, field.WireType)
		}
	}

	// Check for missing required fields
	requiredFields := []int{1, 2, 5} // CodeID, Creator, Created are required
	for _, reqField := range requiredFields {
		if _, ok := fields[reqField]; !ok {
			return PatternMissingRequiredFields, fmt.Sprintf("Add missing required field %d", reqField)
		}
	}

	// Check for duplicate fields
	// In our parsing, duplicates would overwrite, but let's check data
	if len(fields) != countUniqueFields(data) {
		return PatternDuplicateFields, "Remove duplicate field numbers"
	}

	// Unknown pattern - unmarshal fails for unknown reason
	return PatternUnknown, fmt.Sprintf("Unknown corruption - unmarshal error: %s", unmarshalErr.Error())
}

// countUniqueFields counts unique field numbers in protobuf data
func countUniqueFields(data []byte) int {
	seen := make(map[int]bool)
	offset := 0

	for offset < len(data) {
		// Read tag
		tag, n := binary.Uvarint(data[offset:])
		if n <= 0 {
			break
		}
		offset += n

		fieldNumber := int(tag >> 3)
		wireType := int(tag & 0x7)

		seen[fieldNumber] = true

		// Skip field data
		_, bytesRead, err := readFieldData(data[offset:], wireType)
		if err != nil {
			break
		}
		offset += bytesRead
	}

	return len(seen)
}

// contains checks if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || findSubstring(s, substr)))
}

// findSubstring searches for substring in string
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// AnalyzeContractBatch analyzes multiple contracts and returns aggregated statistics
func AnalyzeContractBatch(contracts map[string][]byte) map[ContractState]int {
	distribution := make(map[ContractState]int)

	for address, data := range contracts {
		analysis := AnalyzeContract(address, data)
		distribution[analysis.State]++
	}

	return distribution
}

// AnalyzeCorruptionPatterns analyzes corruption patterns across multiple contracts
func AnalyzeCorruptionPatterns(contracts map[string][]byte) map[CorruptionPattern][]string {
	patterns := make(map[CorruptionPattern][]string)

	for address, data := range contracts {
		analysis := AnalyzeContract(address, data)
		if analysis.State == StateUnmarshalFails {
			patterns[analysis.CorruptionPattern] = append(patterns[analysis.CorruptionPattern], address)
		}
	}

	return patterns
}

// FormatAnalysis returns a human-readable analysis report
func FormatAnalysis(analysis ContractAnalysis) string {
	report := fmt.Sprintf("Contract Analysis: %s\n", FormatAddress(analysis.Address))
	report += "=================================\n\n"

	report += fmt.Sprintf("State: %s\n", analysis.State)
	report += fmt.Sprintf("Data size: %d bytes\n\n", analysis.RawDataSize)

	if analysis.UnmarshalError != nil {
		report += fmt.Sprintf("Unmarshal Error: %s\n", analysis.UnmarshalError)
		report += fmt.Sprintf("Corruption Pattern: %s\n", analysis.CorruptionPattern)
	} else {
		report += "✓ Unmarshal successful\n"
		report += fmt.Sprintf("CodeID: %d\n", analysis.CodeID)
		report += fmt.Sprintf("Creator: %s\n", FormatAddress(analysis.Creator))
		if analysis.Admin != "" {
			report += fmt.Sprintf("Admin: %s\n", FormatAddress(analysis.Admin))
		}
		if analysis.Label != "" {
			report += fmt.Sprintf("Label: %s\n", analysis.Label)
		}
	}

	report += "\nField Analysis:\n"
	report += fmt.Sprintf("  Field 7 (extension): %v", analysis.HasField7)
	if analysis.HasField7 {
		report += fmt.Sprintf(" (wire type %d)", analysis.Field7WireType)
	}
	report += "\n"

	report += fmt.Sprintf("  Field 8 (ibc2_port_id): %v", analysis.HasField8)
	if analysis.HasField8 {
		report += fmt.Sprintf(" (wire type %d, has data: %v)", analysis.Field8WireType, analysis.Field8HasData)
	}
	report += "\n"

	report += fmt.Sprintf("\nFixable: %v\n", analysis.Fixable)
	report += fmt.Sprintf("Fix Strategy: %s\n", analysis.FixStrategy)

	if analysis.FixAttempted {
		report += fmt.Sprintf("\nFix Attempted: %v\n", analysis.FixAttempted)
		report += fmt.Sprintf("Fix Succeeded: %v\n", analysis.FixSucceeded)
		if analysis.FixError != nil {
			report += fmt.Sprintf("Fix Error: %s\n", analysis.FixError)
		}
	}

	report += fmt.Sprintf("\nRaw Data (first 200 bytes hex):\n%s\n", analysis.RawDataHex)

	return report
}
