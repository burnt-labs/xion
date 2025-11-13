package v24_upgrade

import (
	"fmt"
)

// DetectSchemaVersion determines which schema version a contract is using
// Based on the simplified detection strategy: check if field 8 has data
//
// Detection logic:
// - If field 8 is empty/absent -> SchemaLegacy or SchemaCanonical (both safe)
// - If field 8 has data -> SchemaBroken (needs migration)
//
// Since IBCv2 was never used on XION, any data in field 8 indicates the
// extension field was incorrectly placed there during v20/v21.
func DetectSchemaVersion(data []byte) SchemaVersion {
	// Parse protobuf fields
	fields, err := ParseProtobufFields(data)
	if err != nil {
		return SchemaUnknown
	}

	// Check if field 8 exists and has data
	_, hasField8 := fields[8]

	if !hasField8 {
		// Field 8 doesn't exist - this is SchemaLegacy or SchemaCanonical
		// Both are safe (extension at position 7, no ibc2_port_id)
		// We'll treat them as SchemaLegacy for stats purposes
		return SchemaLegacy
	}

	// Field 8 exists - check if it has actual data
	value, err := GetFieldValue(data, 8)
	if err != nil || len(value) == 0 {
		// Field 8 is empty - safe schema
		return SchemaCanonical
	}

	// Field 8 has data - this is SchemaBroken
	// The extension was incorrectly moved to position 8
	return SchemaBroken
}

// NeedsM igration determines if a contract needs migration based on its schema
func NeedsMigration(schema SchemaVersion) bool {
	return schema == SchemaBroken
}

// GetMigrationAction returns a human-readable description of what action is needed
func GetMigrationAction(schema SchemaVersion) string {
	switch schema {
	case SchemaLegacy:
		return "None - already safe (pre-v20)"
	case SchemaBroken:
		return "Swap fields 7 and 8, then null field 8"
	case SchemaCanonical:
		return "None - already correct"
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
