package v24_upgrade

import (
	"time"
)

// SchemaVersion represents different ContractInfo protobuf schemas
type SchemaVersion int

const (
	// SchemaUnknown indicates we couldn't determine the schema
	SchemaUnknown SchemaVersion = iota

	// SchemaLegacy: Pre-v20 contracts (< wasmd v0.61.0)
	// - extension at position 7 (correct)
	// - No ibc2_port_id field (position 8 empty)
	// Action: None needed - already safe
	SchemaLegacy

	// SchemaBroken: v20 & v21 contracts (wasmd v0.61.0 - v0.61.4)
	// - extension at position 8 (WRONG - was moved here by bug)
	// - ibc_port_id at position 7 (WRONG - should be extension)
	// Action: Swap fields 7 and 8
	SchemaBroken

	// SchemaCanonical: Target state (wasmd v0.61.6+)
	// - extension at position 7 (correct)
	// - ibc2_port_id at position 8 (correct, but always null since IBCv2 never used)
	// Action: None needed - already correct
	SchemaCanonical
)

// String returns the name of the schema version
func (s SchemaVersion) String() string {
	switch s {
	case SchemaLegacy:
		return "SchemaLegacy"
	case SchemaBroken:
		return "SchemaBroken"
	case SchemaCanonical:
		return "SchemaCanonical"
	default:
		return "SchemaUnknown"
	}
}

// ContractMigrationResult represents the outcome of migrating a single contract
type ContractMigrationResult struct {
	Address        string
	OriginalSchema SchemaVersion
	Success        bool
	Error          error
	Migrated       bool   // True if contract data was actually changed
	SkipReason     string // Reason if skipped (already correct, etc.)
}

// MigrationStats tracks statistics during migration
type MigrationStats struct {
	TotalContracts     uint64
	ProcessedContracts uint64
	MigratedContracts  uint64 // Contracts that actually needed changes
	SkippedContracts   uint64 // Contracts already correct
	FailedContracts    uint64

	// Schema distribution
	LegacyCount    uint64 // Pre-v20 contracts
	BrokenCount    uint64 // v20/v21 contracts needing migration
	CanonicalCount uint64 // Already correct contracts
	UnknownCount   uint64 // Could not determine schema

	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration

	// Performance metrics
	ContractsPerSecond float64

	// Validation
	ValidationSampleSize int
	ValidationSuccesses  int
	ValidationFailures   int
}

// MigrationReport contains the full migration report
type MigrationReport struct {
	Stats           MigrationStats
	FailedAddresses []string // Addresses that failed migration
	NetworkType     NetworkType
	Mode            MigrationMode

	// Phase timings
	DiscoveryDuration  time.Duration
	BackupDuration     time.Duration
	MigrationDuration  time.Duration
	ValidationDuration time.Duration
	CleanupDuration    time.Duration
}

// ProtobufField represents a parsed protobuf field
type ProtobufField struct {
	Number   int
	WireType int
	Data     []byte
}

// ValidationResult represents a contract validation result
type ValidationResult struct {
	Address     string
	Valid       bool
	Error       error
	Field7Type  int  // Should be extension (wire type 2)
	Field8Empty bool // Should always be true (IBCv2 never used)
}
