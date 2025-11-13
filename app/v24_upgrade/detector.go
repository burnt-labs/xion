package v24_upgrade

import (
	"fmt"
)

// DetectSchemaVersion determines which schema version a contract is using
// Uses comprehensive validation approach:
// 1. First, try to parse protobuf structure - if this fails, it's corrupted
// 2. Check field 7 and field 8 to determine the schema state
// 3. Distinguish between:
//    - SchemaLegacy: Missing field 7 or field 8 (or both) - needs fields added
//    - SchemaCanonical: Both field 7 and field 8 present, field 8 empty (correct state)
//    - SchemaBroken: Field 8 has data (needs swap and clear)
//    - SchemaCorrupted: Cannot parse (unfixable)
//
// NOTE: We make NO assumptions about which fields exist - all contracts are checked
func DetectSchemaVersion(data []byte) SchemaVersion {
	// First, try to parse the protobuf structure
	fields, err := ParseProtobufFields(data)
	if err != nil {
		// Cannot parse protobuf at all - this is corrupted (invalid wire types, truncated data, etc.)
		// These contracts cannot be fixed by simple field swapping
		return SchemaCorrupted
	}

	// Check if field 7 exists
	_, hasField7 := fields[7]

	// Check if field 8 exists and has data
	_, hasField8 := fields[8]
	var field8HasData bool
	if hasField8 {
		value, err := GetFieldValue(data, 8)
		field8HasData = (err == nil && len(value) > 0)
	}

	// Determine schema based on field 7 and field 8 state
	// We CANNOT assume field 7 exists - must check explicitly
	switch {
	case field8HasData:
		// Field 8 has data - this is SchemaBroken (extension moved to field 8 by v20/v21 bug)
		return SchemaBroken

	case hasField7 && hasField8:
		// Both field 7 and field 8 exist (field 8 is empty) - this is SchemaCanonical (correct state)
		return SchemaCanonical

	default:
		// Missing field 7 or field 8 (or both) - this is SchemaLegacy (needs fields added)
		return SchemaLegacy
	}
}

// NeedsMigration determines if a contract needs migration based on its schema
func NeedsMigration(schema SchemaVersion) bool {
	switch schema {
	case SchemaBroken:
		// Field 8 has data - needs field swap
		return true
	case SchemaLegacy:
		// Missing field 7 and/or field 8 - needs fields added
		return true
	case SchemaCorrupted:
		// Corrupted data - migration will fail but we'll try to report it
		return true
	case SchemaCanonical:
		// Already correct - no migration needed
		return false
	default:
		return false
	}
}

// GetMigrationAction returns a human-readable description of what action is needed
func GetMigrationAction(schema SchemaVersion) string {
	switch schema {
	case SchemaLegacy:
		return "Add missing fields (field 7 extension and/or field 8 ibc2_port_id)"
	case SchemaBroken:
		return "Swap fields 7 and 8, then null field 8"
	case SchemaCanonical:
		return "None - already correct"
	case SchemaCorrupted:
		return "Cannot fix - data corruption (invalid wire types, truncated data, etc.)"
	default:
		return "Unknown - manual inspection required"
	}
}

// AnalyzeContract performs detailed analysis of a contract's schema
type ContractAnalysis struct {
	Address        string
	Schema         SchemaVersion
	HasField7      bool
	HasField8      bool
	Field8HasData  bool
	NeedsMigration bool
	Action         string
	Error          error
}

// AnalyzeContractData performs detailed analysis of contract protobuf data
func AnalyzeContractData(address string, data []byte) ContractAnalysis {
	analysis := ContractAnalysis{
		Address: address,
	}

	// Parse fields
	fields, err := ParseProtobufFields(data)
	if err != nil {
		analysis.Error = fmt.Errorf("failed to parse protobuf: %w", err)
		analysis.Schema = SchemaUnknown
		return analysis
	}

	// Check field 7
	_, analysis.HasField7 = fields[7]

	// Check field 8
	_, hasField8 := fields[8]
	analysis.HasField8 = hasField8

	if hasField8 {
		value, err := GetFieldValue(data, 8)
		if err == nil && len(value) > 0 {
			analysis.Field8HasData = true
		}
	}

	// Detect schema
	analysis.Schema = DetectSchemaVersion(data)
	analysis.NeedsMigration = NeedsMigration(analysis.Schema)
	analysis.Action = GetMigrationAction(analysis.Schema)

	return analysis
}

// DetectCorruption checks if contract data is corrupted (returns error on query)
// This is different from detecting schema - this checks if the data is completely invalid
func DetectCorruption(data []byte) error {
	_, err := ParseProtobufFields(data)
	return err
}
