package v24_upgrade

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Migrator handles the contract migration process
type Migrator struct {
	logger      log.Logger
	storeKey    storetypes.StoreKey
	network     NetworkType
	mode        MigrationMode
	stats       *MigrationStats
	statsMutex  sync.Mutex
	failedAddrs []string
	dryRun      bool // If true, migration runs but doesn't save changes
}

// NewMigrator creates a new migrator instance
func NewMigrator(logger log.Logger, storeKey storetypes.StoreKey, network NetworkType, mode MigrationMode) *Migrator {
	return &Migrator{
		logger:      logger,
		storeKey:    storeKey,
		network:     network,
		mode:        mode,
		stats:       &MigrationStats{},
		failedAddrs: make([]string, 0),
		dryRun:      false, // Default to real migration
	}
}

// SetDryRun enables or disables dry-run mode
// In dry-run mode, migration logic executes but changes are not saved to the store
func (m *Migrator) SetDryRun(enabled bool) {
	m.dryRun = enabled
	if enabled {
		m.logger.Warn("⚠️  DRY-RUN MODE ENABLED - No changes will be saved to blockchain state")
	}
}

// IsDryRun returns whether dry-run mode is enabled
func (m *Migrator) IsDryRun() bool {
	return m.dryRun
}

// MigrateContract migrates a single contract based on the simplified strategy
// Returns the migrated data and whether any changes were made
func (m *Migrator) MigrateContract(address string, data []byte) ([]byte, bool, error) {
	// Detect schema
	schema := DetectSchemaVersion(data)

	// Update stats
	m.updateSchemaCount(schema)

	// Check if this is unfixable corruption
	if schema == SchemaCorrupted {
		// Log the corrupted contract address for investigation
		m.logger.Error("Found corrupted contract that cannot be fixed",
			"address", FormatAddress(address),
			"error", "unfixable data corruption (invalid wire types, truncated data, etc.)",
			"recommendation", "Manual investigation required - contract may need to be deleted or recreated",
		)
		// Cannot fix corrupted contracts - report as failed
		return nil, false, fmt.Errorf("unfixable data corruption (invalid wire types, truncated data, etc.)")
	}

	// Check if SchemaCanonical needs field 8 added (even if otherwise correct)
	if schema == SchemaCanonical {
		// Parse to check if field 8 exists
		fields, err := ParseProtobufFields(data)
		if err != nil {
			return nil, false, fmt.Errorf("failed to parse protobuf: %w", err)
		}

		_, hasField8 := fields[8]
		if hasField8 {
			// Field 8 already exists - check if it's empty
			value, err := GetFieldValue(data, 8)
			if err == nil && len(value) == 0 {
				// Already correct - no migration needed
				return data, false, nil
			}
		}

		// Field 8 missing or not empty - ensure it's empty
		migratedData, err := EnsureEmptyField8(data)
		if err != nil {
			return nil, false, fmt.Errorf("failed to ensure field 8: %w", err)
		}
		return migratedData, true, nil
	}

	// Check if migration is needed for other schemas
	if !NeedsMigration(schema) {
		return data, false, nil // No changes needed
	}

	var migratedData []byte
	var err error

	switch schema {
	case SchemaLegacy:
		// Legacy contract - might be missing field 7, field 8, or both
		// First ensure field 7 exists
		migratedData, err = AddEmptyField7(data)
		if err != nil {
			return nil, false, fmt.Errorf("failed to add field 7: %w", err)
		}

		// Then ensure field 8 exists
		migratedData, err = EnsureEmptyField8(migratedData)
		if err != nil {
			return nil, false, fmt.Errorf("failed to ensure field 8: %w", err)
		}

	case SchemaBroken:
		// v20/v21 contract - swap fields 7 and 8
		migratedData, err = SwapFields7And8(data)
		if err != nil {
			return nil, false, fmt.Errorf("failed to swap fields: %w", err)
		}

		// Ensure field 8 exists as empty string (replaces data with empty, or adds if missing)
		migratedData, err = EnsureEmptyField8(migratedData)
		if err != nil {
			return nil, false, fmt.Errorf("failed to ensure field 8: %w", err)
		}

	default:
		return nil, false, fmt.Errorf("unexpected schema type: %v", schema)
	}

	return migratedData, true, nil
}

// MigrateAllContracts performs the full migration of all contracts
func (m *Migrator) MigrateAllContracts(ctx sdk.Context) (*MigrationReport, error) {
	m.logger.Info("Starting v24 contract migration",
		"network", m.network,
		"mode", m.mode,
	)

	startTime := time.Now()
	m.stats.StartTime = startTime

	// Phase 1: Count total contracts
	discoveryStart := time.Now()
	totalContracts, err := m.countContracts(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count contracts: %w", err)
	}
	m.stats.TotalContracts = totalContracts
	discoveryDuration := time.Since(discoveryStart)

	m.logger.Info("Discovery complete",
		"total_contracts", totalContracts,
		"duration", discoveryDuration,
	)

	// Phase 2: Migration
	migrationStart := time.Now()
	err = m.migrateContractsParallel(ctx)
	if err != nil {
		return nil, fmt.Errorf("migration failed: %w", err)
	}
	migrationDuration := time.Since(migrationStart)

	// Calculate stats
	m.stats.EndTime = time.Now()
	m.stats.Duration = time.Since(startTime)
	if m.stats.Duration.Seconds() > 0 {
		m.stats.ContractsPerSecond = float64(m.stats.ProcessedContracts) / m.stats.Duration.Seconds()
	}

	m.logger.Info("Migration complete",
		"total", m.stats.TotalContracts,
		"processed", m.stats.ProcessedContracts,
		"migrated", m.stats.MigratedContracts,
		"skipped", m.stats.SkippedContracts,
		"failed", m.stats.FailedContracts,
		"duration", m.stats.Duration,
		"contracts_per_second", m.stats.ContractsPerSecond,
	)

	// Build report
	report := &MigrationReport{
		Stats:             *m.stats,
		FailedAddresses:   m.failedAddrs,
		NetworkType:       m.network,
		Mode:              m.mode,
		DryRun:            m.dryRun,
		DiscoveryDuration: discoveryDuration,
		MigrationDuration: migrationDuration,
	}

	return report, nil
}

// migrateContractsParallel migrates contracts using parallel workers
func (m *Migrator) migrateContractsParallel(ctx sdk.Context) error {
	store := ctx.KVStore(m.storeKey)

	workers := GetWorkerCount(m.network)
	batchSize := GetBatchSize(m.network)
	progressInterval := GetProgressInterval(m.network)

	m.logger.Info("Starting parallel migration",
		"workers", workers,
		"batch_size", batchSize,
	)

	// Create channels for work distribution
	contractChan := make(chan contractWork, workers*2)
	resultChan := make(chan ContractMigrationResult, workers*2)
	var wg sync.WaitGroup

	// Start worker goroutines
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go m.worker(ctx, contractChan, resultChan, &wg)
	}

	// Start result collector
	collectorDone := make(chan struct{})
	go m.collectResults(resultChan, collectorDone, progressInterval)

	// Iterate over all contracts and send to workers
	prefix := []byte{ContractKeyPrefix}
	iterator := storetypes.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		contractChan <- contractWork{
			address: string(iterator.Key()[len(prefix):]),
			data:    iterator.Value(),
		}
	}

	// Close contract channel to signal workers
	close(contractChan)

	// Wait for all workers to finish
	wg.Wait()

	// Close results channel and wait for collector
	close(resultChan)
	<-collectorDone

	return nil
}

// contractWork represents work to be done by a worker
type contractWork struct {
	address string
	data    []byte
}

// worker processes contracts from the work channel
func (m *Migrator) worker(ctx sdk.Context, workChan <-chan contractWork, resultChan chan<- ContractMigrationResult, wg *sync.WaitGroup) {
	defer wg.Done()

	// Create a new context with its own gas meter for this worker to avoid race conditions
	workerCtx := ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	store := workerCtx.KVStore(m.storeKey)
	prefix := []byte{ContractKeyPrefix}

	for work := range workChan {
		result := ContractMigrationResult{
			Address: work.address,
		}

		// Detect original schema
		result.OriginalSchema = DetectSchemaVersion(work.data)

		// Attempt migration
		migratedData, changed, err := m.MigrateContract(work.address, work.data)
		if err != nil {
			result.Success = false
			result.Error = err
			resultChan <- result
			continue
		}

		result.Success = true
		result.Migrated = changed

		if !changed {
			result.SkipReason = "Already correct schema"
		}

		// Write migrated data back to store if changed
		if changed {
			if m.dryRun {
				// Dry-run mode: Log what would be migrated but don't save
				m.logger.Debug("DRY-RUN: Would migrate contract",
					"address", FormatAddress(work.address),
					"original_schema", result.OriginalSchema.String(),
				)
			} else {
				// Real migration: Save the migrated data
				key := append(prefix, []byte(work.address)...)
				store.Set(key, migratedData)
			}
		}

		resultChan <- result
	}
}

// collectResults collects results from workers and updates stats
func (m *Migrator) collectResults(resultChan <-chan ContractMigrationResult, done chan struct{}, progressInterval int) {
	defer close(done)

	lastLog := time.Now()

	for result := range resultChan {
		atomic.AddUint64(&m.stats.ProcessedContracts, 1)

		if result.Success {
			if result.Migrated {
				atomic.AddUint64(&m.stats.MigratedContracts, 1)
			} else {
				atomic.AddUint64(&m.stats.SkippedContracts, 1)
			}
		} else {
			atomic.AddUint64(&m.stats.FailedContracts, 1)
			m.statsMutex.Lock()
			m.failedAddrs = append(m.failedAddrs, result.Address)
			m.statsMutex.Unlock()

			m.logger.Error("Failed to migrate contract",
				"address", FormatAddress(result.Address),
				"error", result.Error,
			)
		}

		// Log progress
		processed := atomic.LoadUint64(&m.stats.ProcessedContracts)
		if processed%uint64(progressInterval) == 0 || time.Since(lastLog) > 30*time.Second {
			m.logProgress()
			lastLog = time.Now()
		}
	}
}

// logProgress logs current migration progress
func (m *Migrator) logProgress() {
	processed := atomic.LoadUint64(&m.stats.ProcessedContracts)
	migrated := atomic.LoadUint64(&m.stats.MigratedContracts)
	skipped := atomic.LoadUint64(&m.stats.SkippedContracts)
	failed := atomic.LoadUint64(&m.stats.FailedContracts)

	elapsed := time.Since(m.stats.StartTime)
	var rate float64
	if elapsed.Seconds() > 0 {
		rate = float64(processed) / elapsed.Seconds()
	}

	m.logger.Info("Migration progress",
		"processed", processed,
		"total", m.stats.TotalContracts,
		"migrated", migrated,
		"skipped", skipped,
		"failed", failed,
		"elapsed", elapsed,
		"rate_per_sec", fmt.Sprintf("%.1f", rate),
	)
}

// countContracts counts the total number of contracts
func (m *Migrator) countContracts(ctx sdk.Context) (uint64, error) {
	store := ctx.KVStore(m.storeKey)
	prefix := []byte{ContractKeyPrefix}
	iterator := storetypes.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	var count uint64
	for ; iterator.Valid(); iterator.Next() {
		count++
	}

	return count, nil
}

// updateSchemaCount updates the schema distribution stats
func (m *Migrator) updateSchemaCount(schema SchemaVersion) {
	m.statsMutex.Lock()
	defer m.statsMutex.Unlock()

	switch schema {
	case SchemaLegacy:
		m.stats.LegacyCount++
	case SchemaBroken:
		m.stats.BrokenCount++
	case SchemaCanonical:
		m.stats.CanonicalCount++
	default:
		m.stats.UnknownCount++
	}
}
