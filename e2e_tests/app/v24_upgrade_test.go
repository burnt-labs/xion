package e2e_app

import (
	"testing"

	v24_upgrade "github.com/burnt-labs/xion/app/v24_upgrade"
	"github.com/stretchr/testify/require"
)

// NOTE: Full chain e2e tests for v24 upgrade (TestV24Upgrade_ContractMigration, etc.)
// have been moved to app/v24_upgrade/e2e_upgrade_test.go as integration-style tests
// that use app.Setup(t) instead of the full interchaintest framework.
//
// This allows for faster test execution and easier maintenance while still
// providing comprehensive end-to-end testing of the upgrade logic.

// TestV24Upgrade_SchemaDetection tests that the migration correctly detects different schema versions
func TestV24Upgrade_SchemaDetection(t *testing.T) {
	// This is a unit-style test within the e2e package to verify schema detection logic
	// These are the protobuf field scenarios we expect to see

	testCases := []struct {
		name           string
		hasField7      bool
		hasField8      bool
		field8HasData  bool
		expectedSchema v24_upgrade.SchemaVersion
	}{
		{
			name:           "Legacy - field 7 only",
			hasField7:      true,
			hasField8:      false,
			field8HasData:  false,
			expectedSchema: v24_upgrade.SchemaLegacy,
		},
		{
			name:           "Broken - field 8 has data",
			hasField7:      true,
			hasField8:      true,
			field8HasData:  true,
			expectedSchema: v24_upgrade.SchemaBroken,
		},
		{
			name:           "Canonical - field 8 empty",
			hasField7:      true,
			hasField8:      true,
			field8HasData:  false,
			expectedSchema: v24_upgrade.SchemaCanonical,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test protobuf data
			data := createTestProtobuf(tc.hasField7, tc.hasField8, tc.field8HasData)

			// Detect schema
			schema := v24_upgrade.DetectSchemaVersion(data)

			// Verify detection
			require.Equal(t, tc.expectedSchema, schema,
				"Schema detection mismatch for %s", tc.name)
		})
	}
}

func createTestProtobuf(hasField7, hasField8, field8HasData bool) []byte {
	result := make([]byte, 0)

	// Add field 7 if needed
	if hasField7 {
		tag := v24_upgrade.EncodeFieldTag(7, v24_upgrade.WireBytes)
		result = append(result, tag...)
		data := []byte("extension")
		result = append(result, byte(len(data)))
		result = append(result, data...)
	}

	// Add field 8 if needed
	if hasField8 {
		tag := v24_upgrade.EncodeFieldTag(8, v24_upgrade.WireBytes)
		result = append(result, tag...)
		if field8HasData {
			data := []byte("has_data")
			result = append(result, byte(len(data)))
			result = append(result, data...)
		} else {
			result = append(result, 0) // Empty field
		}
	}

	return result
}
