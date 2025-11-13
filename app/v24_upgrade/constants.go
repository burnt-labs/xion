package v24_upgrade

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Migration configuration constants
const (
	// Worker pool configuration
	MainnetWorkers = 40    // Number of parallel workers for mainnet (6M contracts)
	TestnetWorkers = 10    // Number of parallel workers for testnet (500K contracts)
	BatchSize      = 10000 // Contracts per batch for mainnet
	TestnetBatch   = 5000  // Contracts per batch for testnet

	// Performance targets
	MainnetTargetRate = 1200 // Contracts per second for mainnet
	TestnetTargetRate = 250  // Contracts per second for testnet

	// Validation sampling
	MainnetSampleRate = 0.001 // 0.1% of contracts for mainnet validation
	TestnetSampleRate = 0.01  // 1% of contracts for testnet validation

	// Timeouts and limits
	MaxMigrationTime   = 3 * time.Hour // Maximum time allowed for migration
	ProgressLogEvery   = 250000        // Log progress every N contracts (mainnet)
	TestnetProgressLog = 50000         // Log progress every N contracts (testnet)
	ValidationTimeout  = 10 * time.Minute

	// Protobuf field positions (critical for migration)
	FieldExtension  = 7 // extension field - must be at position 7
	FieldIBC2PortID = 8 // ibc2_port_id field - must be at position 8 (and always null)

	// ContractInfo store prefix
	ContractKeyPrefix = 0x02 // Wasm module uses 0x02 prefix for contract info
)

// Migration mode determines behavior when corrupted contracts are found
type MigrationMode int

const (
	// ModeFailOnCorruption stops upgrade if any corrupted contracts are found
	ModeFailOnCorruption MigrationMode = iota

	// ModeAutoMigrate attempts to fix all corrupted contracts
	ModeAutoMigrate

	// ModeLogAndContinue logs corrupted contracts but continues (manual fix needed)
	ModeLogAndContinue
)

// Network type for configuration
type NetworkType string

const (
	Mainnet NetworkType = "mainnet"
	Testnet NetworkType = "testnet"
)

// GetWorkerCount returns the appropriate number of workers for the network
func GetWorkerCount(network NetworkType) int {
	if network == Mainnet {
		return MainnetWorkers
	}
	return TestnetWorkers
}

// GetBatchSize returns the appropriate batch size for the network
func GetBatchSize(network NetworkType) int {
	if network == Mainnet {
		return BatchSize
	}
	return TestnetBatch
}

// GetProgressInterval returns the progress logging interval for the network
func GetProgressInterval(network NetworkType) int {
	if network == Mainnet {
		return ProgressLogEvery
	}
	return TestnetProgressLog
}

// GetSampleRate returns the validation sample rate for the network
func GetSampleRate(network NetworkType) float64 {
	if network == Mainnet {
		return MainnetSampleRate
	}
	return TestnetSampleRate
}

// FormatAddress converts raw address bytes to human-readable Bech32 format
// Handles both raw bytes and string-encoded bytes
func FormatAddress(addr string) string {
	// Convert string to bytes (addresses are stored as raw bytes converted to string)
	addrBytes := []byte(addr)

	// Convert to SDK AccAddress and then to Bech32
	accAddr := sdk.AccAddress(addrBytes)
	return accAddr.String()
}
