package main

import (
	"fmt"

	"github.com/spf13/cobra"

	dbm "github.com/cosmos/cosmos-db"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/cosmos/cosmos-sdk/server"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/app"
	v24_upgrade "github.com/burnt-labs/xion/app/v24_upgrade"
)

// MigrateV24Cmd returns a command to manually run the v24 migration
// This is a temporary command for testing/emergency migration outside of upgrades
func MigrateV24Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate-v24",
		Short: "Manually run the v24 contract migration",
		Long: `Manually run the v24 contract migration outside of an upgrade.

This command performs the following:
1. Opens the database
2. Detects schema for all contracts
3. Migrates contracts that need fixing
4. Validates all contracts after migration

WARNING: This should only be used for:
- Testing before actual upgrade
- Emergency recovery scenarios
- Development/debugging

The migration makes NO assumptions about contract state and checks every contract.`,
		Example: `  # Dry-run to see what would be migrated (no changes saved)
  xiond migrate-v24 --dry-run

  # Run the actual migration
  xiond migrate-v24

  # Run migration and skip validation
  xiond migrate-v24 --skip-validation`,
		RunE: func(cmd *cobra.Command, args []string) error {
			serverCtx := server.GetServerContextFromCmd(cmd)
			config := serverCtx.Config

			// Get flags
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			skipValidation, _ := cmd.Flags().GetBool("skip-validation")

			// Open database
			serverCtx.Logger.Info("Opening database", "root_dir", config.RootDir, "data_dir", config.RootDir+"/data")
			db, err := openDB(config.RootDir, server.GetAppDBBackend(serverCtx.Viper))
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer db.Close()

			// Create app
			logger := serverCtx.Logger
			logger.Info("Initializing app for migration...")

			wasmApp := app.NewWasmApp(
				logger,
				db,
				nil,                       // trace writer
				true,                      // load latest
				simtestutil.EmptyAppOptions{}, // app options
				nil,                       // wasm options
			)

			// Create context with empty header
			ctx := sdk.NewContext(
				wasmApp.CommitMultiStore(),
				tmproto.Header{},
				false, // checkTx
				logger,
			)

			// Detect network (testnet vs mainnet) based on chain ID
			network := v24_upgrade.Testnet
			if ctx.ChainID() == "xion-mainnet-1" {
				network = v24_upgrade.Mainnet
			}

			logger.Info("Starting v24 migration",
				"chain_id", ctx.ChainID(),
				"network", network,
				"dry_run", dryRun,
			)

			// Create migrator
			migrator := v24_upgrade.NewMigrator(
				logger,
				wasmApp.GetKey("wasm"),
				network,
				v24_upgrade.ModeAutoMigrate,
			)

			// Set dry-run mode if requested
			if dryRun {
				migrator.SetDryRun(true)
				logger.Warn("⚠️  DRY-RUN MODE - No changes will be saved")
			}

			// Run migration
			report, err := migrator.MigrateAllContracts(ctx)
			if err != nil {
				return fmt.Errorf("migration failed: %w", err)
			}

			// Commit changes if not dry-run
			if !dryRun {
				logger.Info("Committing migration changes to database...")
				wasmApp.CommitMultiStore().Commit()
				logger.Info("✅ Migration changes committed")
			} else {
				logger.Info("✅ Dry-run completed (no changes saved)")
			}

			// Run validation unless skipped
			var validationResults []v24_upgrade.ValidationResult
			if !skipValidation {
				logger.Info("Running post-migration validation...")
				validator := v24_upgrade.NewValidator(logger, wasmApp.GetKey("wasm"), network)

				var validationErr error
				validationResults, validationErr = validator.ValidateMigration(ctx, report.Stats.TotalContracts)
				if validationErr != nil {
					logger.Error("❌ Validation failed", "error", validationErr)
					if !dryRun {
						logger.Error("⚠️  Migration was committed but validation failed!")
						logger.Error("⚠️  You may need to investigate and fix manually")
					}
					return fmt.Errorf("validation failed: %w", validationErr)
				}

				logger.Info("✅ Validation passed - all contracts are valid")
			}

			// Generate report
			reportGen := v24_upgrade.NewReportGenerator(logger)
			reportGen.GenerateReport(report, validationResults)

			if dryRun {
				logger.Info("")
				logger.Info("====================================================")
				logger.Info("DRY-RUN COMPLETE - Run without --dry-run to migrate")
				logger.Info("====================================================")
			} else {
				logger.Info("")
				logger.Info("====================================================")
				logger.Info("MIGRATION COMPLETE")
				logger.Info("====================================================")
			}

			return nil
		},
	}

	// Add flags
	cmd.Flags().Bool("dry-run", false, "Run migration without saving changes (analyze only)")
	cmd.Flags().Bool("skip-validation", false, "Skip post-migration validation (not recommended)")

	return cmd
}

// openDB opens the database at the given root directory
func openDB(rootDir string, backendType dbm.BackendType) (dbm.DB, error) {
	if rootDir == "" {
		return nil, fmt.Errorf("root directory cannot be empty")
	}

	// Database is stored in the "data" subdirectory
	dataDir := rootDir + "/data"

	return dbm.NewDB("application", backendType, dataDir)
}
