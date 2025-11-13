package v24_upgrade

import (
	"fmt"
	"math/rand"
	"time"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Validator handles post-migration validation
type Validator struct {
	logger   log.Logger
	storeKey storetypes.StoreKey
	network  NetworkType
}

// NewValidator creates a new validator instance
func NewValidator(logger log.Logger, storeKey storetypes.StoreKey, network NetworkType) *Validator {
	return &Validator{
		logger:   logger,
		storeKey: storeKey,
		network:  network,
	}
}

// ValidateMigration performs full validation of ALL contracts after migration
func (v *Validator) ValidateMigration(ctx sdk.Context, totalContracts uint64) ([]ValidationResult, error) {
	v.logger.Info("Starting post-migration validation")

	startTime := time.Now()

	v.logger.Info("Validation parameters",
		"total_contracts", totalContracts,
		"validation_mode", "FULL (all contracts)",
	)

	// Get all contract addresses
	addresses, err := v.getAllContractAddresses(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get contract addresses: %w", err)
	}

	// Validate ALL contracts (no sampling)
	// Validate each contract
	results := make([]ValidationResult, 0, len(addresses))
	successCount := 0
	failureCount := 0

	for i, addr := range addresses {
		result := v.validateContract(ctx, addr)
		results = append(results, result)

		if result.Valid {
			successCount++
		} else {
			failureCount++
			v.logger.Error("Validation failed for contract",
				"address", FormatAddress(addr),
				"error", result.Error,
				"field7_type", result.Field7Type,
				"field8_empty", result.Field8Empty,
			)
		}

		// Log progress every 1000 contracts
		if (i+1)%1000 == 0 {
			v.logger.Info("Validation progress",
				"validated", i+1,
				"total", len(addresses),
				"success", successCount,
				"failures", failureCount,
			)
		}
	}

	duration := time.Since(startTime)

	// Calculate rate
	var contractsPerSecond float64
	if duration.Seconds() > 0 {
		contractsPerSecond = float64(len(addresses)) / duration.Seconds()
	}

	v.logger.Info("Validation complete",
		"total_validated", len(addresses),
		"successes", successCount,
		"failures", failureCount,
		"success_rate", fmt.Sprintf("%.2f%%", float64(successCount)/float64(len(addresses))*100),
		"duration", duration,
		"contracts_per_second", fmt.Sprintf("%.0f", contractsPerSecond),
	)

	if failureCount > 0 {
		return results, fmt.Errorf("validation failed: %d/%d contracts failed validation", failureCount, len(addresses))
	}

	return results, nil
}

// validateContract validates a single contract
func (v *Validator) validateContract(ctx sdk.Context, address string) ValidationResult {
	result := ValidationResult{
		Address: address,
		Valid:   false,
	}

	// Read contract data
	store := ctx.KVStore(v.storeKey)
	prefix := []byte{ContractKeyPrefix}
	key := append(prefix, []byte(address)...)
	data := store.Get(key)

	if data == nil {
		result.Error = fmt.Errorf("contract not found")
		return result
	}

	// Parse protobuf fields
	fields, err := ParseProtobufFields(data)
	if err != nil {
		result.Error = fmt.Errorf("failed to parse protobuf: %w", err)
		return result
	}

	// Detect schema version to determine expected structure
	schema := DetectSchemaVersion(data)

	switch schema {
	case SchemaLegacy:
		// Pre-v20 legacy contract - should have been migrated to add field 8
		// After migration, this should not exist (all SchemaLegacy become SchemaCanonical)
		result.Error = fmt.Errorf("contract still has SchemaLegacy after migration (field 8 missing)")
		result.Field8Empty = true // Field 8 is missing
		return result

	case SchemaBroken:
		// Should NEVER see this after migration - all broken contracts should be fixed
		result.Error = fmt.Errorf("contract still has SchemaBroken after migration (field 8 has data)")
		result.Field8Empty = false
		return result

	case SchemaCanonical:
		// Post-migration canonical state: Field 7 present (extension), field 8 present and empty
		// This is the ONLY valid state after migration
		field7, hasField7 := fields[7]
		if !hasField7 {
			result.Error = fmt.Errorf("canonical schema requires field 7 (extension)")
			return result
		}

		result.Field7Type = field7.WireType
		if field7.WireType != WireBytes {
			result.Error = fmt.Errorf("field 7 has wrong wire type: expected %d (bytes), got %d", WireBytes, field7.WireType)
			return result
		}

		// Validate field 8 exists and is empty (all contracts must have field 8 after v24)
		_, hasField8 := fields[8]
		if !hasField8 {
			result.Error = fmt.Errorf("field 8 (ibc2_port_id) is missing - all contracts must have field 8 after v24 migration")
			result.Field8Empty = true // Technically empty but missing
			return result
		}

		// Validate field 8 is empty (IBCv2 never used)
		result.Field8Empty = IsField8Empty(data)
		if !result.Field8Empty {
			result.Error = fmt.Errorf("field 8 is not empty (IBCv2 should never be used)")
			return result
		}

		// Canonical contract is valid
		result.Valid = true
		return result

	case SchemaCorrupted:
		// Data corruption detected
		result.Error = fmt.Errorf("contract data is corrupted (invalid wire types, truncated data, etc.)")
		return result

	default:
		result.Error = fmt.Errorf("unknown schema version")
		return result
	}
}

// getAllContractAddresses retrieves all contract addresses
func (v *Validator) getAllContractAddresses(ctx sdk.Context) ([]string, error) {
	store := ctx.KVStore(v.storeKey)
	prefix := []byte{ContractKeyPrefix}
	iterator := storetypes.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	addresses := make([]string, 0)
	for ; iterator.Valid(); iterator.Next() {
		addr := string(iterator.Key()[len(prefix):])
		addresses = append(addresses, addr)
	}

	return addresses, nil
}

// sampleAddresses randomly samples n addresses from the list
func (v *Validator) sampleAddresses(addresses []string, n int) []string {
	if n >= len(addresses) {
		return addresses
	}

	// Create a random number generator with current time as seed
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Shuffle and take first n
	shuffled := make([]string, len(addresses))
	copy(shuffled, addresses)
	rng.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	return shuffled[:n]
}

// ValidateContract is a public method to validate a specific contract
func (v *Validator) ValidateContract(ctx sdk.Context, address string) ValidationResult {
	return v.validateContract(ctx, address)
}

// ValidateSchemaDistribution validates that the schema distribution makes sense
func (v *Validator) ValidateSchemaDistribution(stats MigrationStats) error {
	total := stats.LegacyCount + stats.BrokenCount + stats.CanonicalCount + stats.UnknownCount

	if total != stats.TotalContracts {
		return fmt.Errorf("schema count mismatch: %d != %d", total, stats.TotalContracts)
	}

	// Check for unexpected unknown schemas
	if stats.UnknownCount > 0 {
		unknownPercent := float64(stats.UnknownCount) / float64(stats.TotalContracts) * 100
		v.logger.Warn("Found contracts with unknown schema",
			"count", stats.UnknownCount,
			"percentage", fmt.Sprintf("%.2f%%", unknownPercent),
		)
	}

	v.logger.Info("Schema distribution validated",
		"legacy", stats.LegacyCount,
		"broken", stats.BrokenCount,
		"canonical", stats.CanonicalCount,
		"unknown", stats.UnknownCount,
		"total", stats.TotalContracts,
	)

	return nil
}
