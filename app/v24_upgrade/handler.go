package v24_upgrade

import (
	"fmt"
	"time"

	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// PerformMigration is the main entry point for the v24 upgrade migration
// It should be called from the upgrade handler in app/upgrades.go
//
// This function:
// 1. Detects the network type (mainnet/testnet) based on chain ID
// 2. Creates a migrator with appropriate configuration
// 3. Executes the migration
// 4. Validates the results
// 5. Generates a comprehensive report
//
// The migration mode can be configured (see constants.go)
func PerformMigration(ctx sdk.Context, storeKey storetypes.StoreKey) error {
	logger := ctx.Logger()

	logger.Info("=================================================================")
	logger.Info("          Starting V24 Upgrade: Contract Migration")
	logger.Info("=================================================================")

	// Determine network type from chain ID
	chainID := ctx.ChainID()
	network := detectNetwork(chainID)
	logger.Info("Detected network", "chain_id", chainID, "network", network)

	// Set migration mode (can be configured)
	// For production: use ModeAutoMigrate
	// For testing: use ModeLogAndContinue or ModeFailOnCorruption
	mode := ModeAutoMigrate

	logger.Info("Migration configuration",
		"mode", mode,
		"workers", GetWorkerCount(network),
		"batch_size", GetBatchSize(network),
	)

	// Phase 1: Create migrator and execute migration
	logger.Info("--- PHASE 1: MIGRATION ---")
	migrationStart := time.Now()

	migrator := NewMigrator(logger, storeKey, network, mode)
	report, err := migrator.MigrateAllContracts(ctx)
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	report.MigrationDuration = time.Since(migrationStart)

	// Phase 2: Validation
	logger.Info("--- PHASE 2: VALIDATION ---")
	validationStart := time.Now()

	validator := NewValidator(logger, storeKey, network)

	// Validate schema distribution
	if err := validator.ValidateSchemaDistribution(report.Stats); err != nil {
		logger.Warn("Schema distribution validation warning", "error", err)
	}

	// Perform statistical validation
	validationResults, err := validator.ValidateMigration(ctx, report.Stats.TotalContracts)
	if err != nil {
		logger.Error("Validation failed", "error", err)
		// Continue anyway - we'll report the validation failure
	}

	report.ValidationDuration = time.Since(validationStart)

	// Phase 3: Generate report
	logger.Info("--- PHASE 3: REPORTING ---")
	reportGen := NewReportGenerator(logger)
	reportGen.GenerateReport(report, validationResults)

	// Determine if we should fail the upgrade based on results
	if report.Stats.FailedContracts > 0 {
		if mode == ModeFailOnCorruption {
			return fmt.Errorf("upgrade halted: %d contracts failed migration", report.Stats.FailedContracts)
		}
		logger.Warn("Migration completed with failures - manual remediation may be required",
			"failed_count", report.Stats.FailedContracts)
	}

	logger.Info("=================================================================")
	logger.Info("          V24 Upgrade Complete")
	logger.Info("=================================================================")

	return nil
}

// detectNetwork determines if we're on mainnet or testnet based on chain ID
func detectNetwork(chainID string) NetworkType {
	// XION mainnet chain ID is "xion-1"
	// XION testnets use "xion-testnet-*" pattern
	if chainID == "xion-1" {
		return Mainnet
	}
	return Testnet
}

// DryRunAnalysis performs a dry-run analysis without making changes
// Useful for testing and planning before actual upgrade
func DryRunAnalysis(ctx sdk.Context, storeKey storetypes.StoreKey) (*MigrationStats, error) {
	logger := ctx.Logger()
	network := detectNetwork(ctx.ChainID())

	logger.Info("Performing dry-run analysis (no changes will be made)")

	stats := &MigrationStats{
		StartTime: time.Now(),
	}

	// Count contracts and analyze schemas
	store := ctx.KVStore(storeKey)
	prefix := []byte{ContractKeyPrefix}
	iterator := storetypes.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		stats.TotalContracts++

		address := string(iterator.Key()[len(prefix):])
		data := iterator.Value()

		// Detect schema
		schema := DetectSchemaVersion(data)

		switch schema {
		case SchemaLegacy:
			stats.LegacyCount++
		case SchemaBroken:
			stats.BrokenCount++
		case SchemaCanonical:
			stats.CanonicalCount++
		default:
			stats.UnknownCount++
			logger.Warn("Unknown schema detected", "address", address)
		}

		// Log progress every 100k contracts
		if stats.TotalContracts%100000 == 0 {
			logger.Info("Dry-run progress", "analyzed", stats.TotalContracts)
		}
	}

	stats.EndTime = time.Now()
	stats.Duration = stats.EndTime.Sub(stats.StartTime)

	// Log summary
	logger.Info("Dry-run complete",
		"total", stats.TotalContracts,
		"legacy", stats.LegacyCount,
		"broken", stats.BrokenCount,
		"canonical", stats.CanonicalCount,
		"unknown", stats.UnknownCount,
		"duration", stats.Duration,
	)

	logger.Info("Estimated migration",
		"contracts_needing_migration", stats.BrokenCount,
		"contracts_already_safe", stats.LegacyCount+stats.CanonicalCount,
	)

	// Estimate time needed
	targetRate := float64(GetWorkerCount(network) * 30) // Rough estimate: 30 contracts/sec per worker
	estimatedSeconds := float64(stats.TotalContracts) / targetRate
	estimatedDuration := time.Duration(estimatedSeconds) * time.Second

	logger.Info("Time estimate",
		"target_rate", fmt.Sprintf("%.0f contracts/sec", targetRate),
		"estimated_duration", estimatedDuration,
	)

	return stats, nil
}

// AnalyzeSingleContract analyzes a single contract (useful for debugging)
func AnalyzeSingleContract(ctx sdk.Context, storeKey storetypes.StoreKey, address string) (*ContractAnalysis, error) {
	store := ctx.KVStore(storeKey)
	prefix := []byte{ContractKeyPrefix}
	key := append(prefix, []byte(address)...)

	data := store.Get(key)
	if data == nil {
		return nil, fmt.Errorf("contract not found: %s", address)
	}

	analysis := AnalyzeContractData(address, data)
	return &analysis, analysis.Error
}
