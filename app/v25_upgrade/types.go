package v25_upgrade

import (
	"time"
)

// ContractState represents the state of a contract after detection
type ContractState int

const (
	// StateUnknown indicates we couldn't determine the state
	StateUnknown ContractState = iota

	// StateHealthy: Contract unmarshals successfully and has canonical schema
	// - proto.Unmarshal() succeeds
	// - Field 7 (extension) present with WireBytes type
	// - Field 8 (ibc2_port_id) present
	// Note: Field 8 can be empty OR have data - both are valid!
	// Action: None needed
	StateHealthy

	// StateUnmarshalFails: Contract fails to unmarshal as ContractInfo
	// - proto.Unmarshal() fails with error
	// - Indicates data corruption at protobuf level
	// - Requires deep analysis and repair
	// Action: Analyze patterns and attempt targeted fixes
	StateUnmarshalFails

	// StateSchemaInconsistent: Unmarshal succeeds but schema is non-canonical
	// - proto.Unmarshal() succeeds
	// - But field 7/8 are missing or have wrong wire types
	// - Needs field addition or type correction
	// Action: Add missing fields with correct wire types
	StateSchemaInconsistent

	// StateUnfixable: Corruption is too severe to fix automatically
	// - Cannot determine valid repair strategy
	// - May need manual intervention or contract deletion
	// Action: Report to operators for manual handling
	StateUnfixable
)

// String returns the name of the contract state
func (s ContractState) String() string {
	switch s {
	case StateHealthy:
		return "StateHealthy"
	case StateUnmarshalFails:
		return "StateUnmarshalFails"
	case StateSchemaInconsistent:
		return "StateSchemaInconsistent"
	case StateUnfixable:
		return "StateUnfixable"
	default:
		return "StateUnknown"
	}
}

// CorruptionPattern represents a specific type of data corruption
type CorruptionPattern int

const (
	PatternUnknown               CorruptionPattern = iota
	PatternInvalidWireType                         // Wire types 6, 7 (invalid in proto3)
	PatternTruncatedField                          // Field with incomplete data
	PatternMalformedLength                         // Invalid length delimiter
	PatternFieldNumberCorruption                   // Field number is invalid
	PatternMissingRequiredFields                   // Required fields are missing
	PatternDuplicateFields                         // Same field number appears multiple times
)

// String returns the name of the corruption pattern
func (p CorruptionPattern) String() string {
	switch p {
	case PatternInvalidWireType:
		return "InvalidWireType"
	case PatternTruncatedField:
		return "TruncatedField"
	case PatternMalformedLength:
		return "MalformedLength"
	case PatternFieldNumberCorruption:
		return "FieldNumberCorruption"
	case PatternMissingRequiredFields:
		return "MissingRequiredFields"
	case PatternDuplicateFields:
		return "DuplicateFields"
	default:
		return "Unknown"
	}
}

// ContractAnalysis represents detailed analysis of a contract
type ContractAnalysis struct {
	Address           string
	State             ContractState
	UnmarshalError    error
	CorruptionPattern CorruptionPattern
	RawDataHex        string // Hex dump of raw data (for debugging)
	RawDataSize       int

	// Protobuf field analysis (if parseable)
	HasField7      bool
	HasField8      bool
	Field8HasData  bool
	Field7WireType int
	Field8WireType int

	// ContractInfo fields (if unmarshal succeeds)
	CodeID  uint64
	Creator string
	Admin   string
	Label   string

	// Fix metadata
	Fixable      bool
	FixStrategy  string
	FixAttempted bool
	FixSucceeded bool
	FixError     error
}

// ContractFixResult represents the result of attempting to fix a contract
type ContractFixResult struct {
	Address        string
	OriginalState  ContractState
	FixAttempted   bool
	FixSucceeded   bool
	FinalState     ContractState
	FixStrategy    string
	Error          error
	OriginalData   []byte // Original raw data
	FixedData      []byte // Fixed data (if fix succeeded)
	UnmarshalAfter bool   // True if fixed data can unmarshal
}

// MigrationStats tracks statistics during v25 migration
type MigrationStats struct {
	TotalContracts     uint64
	ProcessedContracts uint64

	// State distribution
	HealthyCount            uint64 // Already good
	UnmarshalFailsCount     uint64 // Needs repair
	SchemaInconsistentCount uint64 // Needs normalization
	UnfixableCount          uint64 // Cannot fix

	// Fix results
	FixAttemptedCount uint64 // Contracts we tried to fix
	FixSuccessCount   uint64 // Successfully fixed
	FixFailureCount   uint64 // Failed to fix

	// Performance metrics
	StartTime          time.Time
	EndTime            time.Time
	Duration           time.Duration
	ContractsPerSecond float64

	// Corruption patterns found
	CorruptionPatterns map[CorruptionPattern]uint64
}

// MigrationReport contains the full v25 migration report
type MigrationReport struct {
	Stats              MigrationStats
	HealthyContracts   []string // Contracts that were already healthy
	FixedContracts     []string // Contracts that were successfully fixed
	UnfixableContracts []string // Contracts that couldn't be fixed
	FailedContracts    []string // Contracts where fix was attempted but failed

	NetworkType NetworkType
	Mode        MigrationMode
	DryRun      bool // True if this was a dry-run

	// Phase timings
	DiscoveryDuration  time.Duration
	AnalysisDuration   time.Duration
	RepairDuration     time.Duration
	ValidationDuration time.Duration
}

// NetworkType indicates mainnet vs testnet
type NetworkType int

const (
	Mainnet NetworkType = iota
	Testnet
)

// String returns the network type name
func (n NetworkType) String() string {
	switch n {
	case Mainnet:
		return "mainnet"
	case Testnet:
		return "testnet"
	default:
		return "unknown"
	}
}

// MigrationMode controls migration behavior
type MigrationMode int

const (
	// ModeAutoFix: Automatically fix all fixable contracts
	ModeAutoFix MigrationMode = iota

	// ModeAnalyzeOnly: Analyze and report but don't fix
	ModeAnalyzeOnly

	// ModeFailOnUnfixable: Halt upgrade if any unfixable contracts found
	ModeFailOnUnfixable
)

// String returns the migration mode name
func (m MigrationMode) String() string {
	switch m {
	case ModeAutoFix:
		return "AutoFix"
	case ModeAnalyzeOnly:
		return "AnalyzeOnly"
	case ModeFailOnUnfixable:
		return "FailOnUnfixable"
	default:
		return "Unknown"
	}
}

// ProtobufField represents a parsed protobuf field
type ProtobufField struct {
	Number   int
	WireType int
	Data     []byte
	Offset   int // Offset in the original data
	Length   int // Total length including tag and data
}

// Wire types as defined in protobuf spec
const (
	WireVarint = 0
	Wire64Bit  = 1
	WireBytes  = 2
	// WireStartGroup = 3 (deprecated)
	// WireEndGroup = 4 (deprecated)
	Wire32Bit = 5
	// 6 and 7 are invalid in proto3
)

// Contract storage constants
const (
	// ContractKeyPrefix is the prefix for ContractInfo storage in the wasm module
	// 0x02 = ContractInfo (metadata: CodeID, Creator, Admin, Label, etc.)
	// 0x03 = ContractStorePrefix (actual contract state/KV store - NOT what we want)
	ContractKeyPrefix = 0x02
)
