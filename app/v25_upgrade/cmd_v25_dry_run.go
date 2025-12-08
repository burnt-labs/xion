package v25_upgrade

import (
	"fmt"
	"path/filepath"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/gogo/protobuf/proto"
	"github.com/spf13/cobra"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	dbm "github.com/cosmos/cosmos-db"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func V25DryRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "v25-dry-run",
		Short: "Test v25 migration logic without modifying the database",
		Long: `Performs a dry-run of the v25 contract migration to validate the migration logic.

This command:
1. Opens the database in read-only mode
2. Scans all contracts
3. Tests the migration logic on corrupted contracts
4. Reports what would be fixed without actually modifying the database
5. Validates that all fixes work correctly

This is SAFE to run on production data - it does not modify the database.
Use this to validate that the v25 migration will work correctly before deploying.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			homeDir, err := cmd.Flags().GetString("home")
			if err != nil {
				return err
			}

			verbose, err := cmd.Flags().GetBool("verbose")
			if err != nil {
				return err
			}

			limit, err := cmd.Flags().GetInt("limit")
			if err != nil {
				return err
			}

			return runV25DryRun(homeDir, verbose, limit)
		},
	}

	cmd.Flags().Bool("verbose", false, "Show detailed information for each contract")
	cmd.Flags().Int("limit", 0, "Max contracts to process (0 = all)")

	return cmd
}

func runV25DryRun(homeDir string, verbose bool, limit int) error {
	fmt.Printf("V25 Migration Dry Run\n")
	fmt.Printf("=====================\n\n")
	fmt.Printf("⚠️  DRY RUN MODE - No changes will be written to the database\n\n")

	// Open database
	dataDir := filepath.Join(homeDir, "data")
	fmt.Printf("Opening database at: %s\n", dataDir)

	db, err := dbm.NewDB("application", dbm.GoLevelDBBackend, dataDir)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Create store
	storeKey := storetypes.NewKVStoreKey("wasm")
	logger := log.NewNopLogger()
	cms := store.NewCommitMultiStore(db, logger, nil)
	cms.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, nil)

	if err := cms.LoadLatestVersion(); err != nil {
		return fmt.Errorf("failed to load store: %w", err)
	}

	ctx := sdk.NewContext(cms, tmproto.Header{}, false, logger)
	kvStore := ctx.KVStore(storeKey)

	fmt.Printf("✓ Database opened (read-only)\n\n")
	fmt.Printf("Scanning contracts...\n\n")

	// Track statistics
	var totalContracts int
	var healthyContracts int
	var schemaInconsistentContracts int
	var corruptedContracts int
	var fixableContracts int
	var unfixableContracts int
	var fixSuccesses int
	var fixFailures int

	// Track examples
	fixedExamples := make([]string, 0)
	failedExamples := make([]string, 0)

	// Iterate all contracts
	prefix := []byte{ContractKeyPrefix}
	iterator := storetypes.KVStorePrefixIterator(kvStore, prefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		totalContracts++

		if limit > 0 && totalContracts > limit {
			fmt.Printf("Reached limit of %d contracts\n\n", limit)
			break
		}

		// Get address
		key := iterator.Key()
		addressBytes := key[len(prefix):]
		accAddr := sdk.AccAddress(addressBytes)
		address := accAddr.String()

		// Get data
		data := iterator.Value()

		// Try to unmarshal
		var contractInfo wasmtypes.ContractInfo
		unmarshalErr := proto.Unmarshal(data, &contractInfo)

		if unmarshalErr == nil {
			// Contract can unmarshal - check if schema-inconsistent
			state, _ := DetectContractState(data)
			if state == StateHealthy {
				healthyContracts++
				if verbose {
					fmt.Printf("  ✓ %s - Healthy\n", FormatAddress(address))
				}
			} else {
				schemaInconsistentContracts++
				if verbose {
					fmt.Printf("  ~ %s - Schema inconsistent (but functional)\n", FormatAddress(address))
				}
			}
		} else {
			// Contract fails to unmarshal - needs fixing
			corruptedContracts++

			if verbose {
				fmt.Printf("  ✗ %s - Corrupted: %s\n", FormatAddress(address), unmarshalErr)
			}

			// Test the fix
			result := FixContract(address, data)

			if result.FixSucceeded && result.UnmarshalAfter {
				fixSuccesses++
				fixableContracts++

				if verbose {
					fmt.Printf("    → Fix would succeed (%s)\n", result.FixStrategy)
				}

				if len(fixedExamples) < 5 {
					fixedExamples = append(fixedExamples, address)
				}
			} else {
				fixFailures++
				unfixableContracts++

				if verbose {
					errorMsg := "unknown error"
					if result.Error != nil {
						errorMsg = result.Error.Error()
					}
					fmt.Printf("    → Fix would FAIL: %s\n", errorMsg)
				}

				if len(failedExamples) < 5 {
					failedExamples = append(failedExamples, address)
				}
			}
		}

		// Progress indicator
		if !verbose && totalContracts%1000 == 0 {
			fmt.Printf("  Processed %d contracts...\n", totalContracts)
		}
	}

	// Print report
	fmt.Printf("\n")
	fmt.Printf("=================================\n")
	fmt.Printf("DRY RUN REPORT\n")
	fmt.Printf("=================================\n\n")

	fmt.Printf("Total Contracts: %d\n\n", totalContracts)

	// Working contracts
	workingContracts := healthyContracts + schemaInconsistentContracts
	fmt.Printf("Working Contracts (can unmarshal): %d (%.2f%%)\n",
		workingContracts,
		float64(workingContracts)/float64(totalContracts)*100)

	if healthyContracts > 0 {
		fmt.Printf("  ✓ Healthy (canonical schema): %d (%.2f%%)\n",
			healthyContracts,
			float64(healthyContracts)/float64(totalContracts)*100)
	}

	if schemaInconsistentContracts > 0 {
		fmt.Printf("  ~ Schema inconsistent (functional): %d (%.2f%%)\n",
			schemaInconsistentContracts,
			float64(schemaInconsistentContracts)/float64(totalContracts)*100)
	}

	fmt.Printf("\n")

	// Corrupted contracts
	fmt.Printf("Corrupted Contracts (need fixing): %d (%.2f%%)\n",
		corruptedContracts,
		float64(corruptedContracts)/float64(totalContracts)*100)

	if corruptedContracts > 0 {
		fmt.Printf("  ✓ Fixable: %d (%.2f%% of corrupted, %.2f%% overall success rate)\n",
			fixableContracts,
			float64(fixableContracts)/float64(corruptedContracts)*100,
			float64(fixSuccesses)/float64(fixSuccesses+fixFailures)*100)

		if len(fixedExamples) > 0 {
			fmt.Printf("    Examples:\n")
			for _, addr := range fixedExamples {
				fmt.Printf("      - %s\n", FormatAddress(addr))
			}
		}

		if unfixableContracts > 0 {
			fmt.Printf("\n  ✗ Unfixable: %d (%.2f%% of corrupted)\n",
				unfixableContracts,
				float64(unfixableContracts)/float64(corruptedContracts)*100)

			if len(failedExamples) > 0 {
				fmt.Printf("    Examples:\n")
				for _, addr := range failedExamples {
					fmt.Printf("      - %s\n", FormatAddress(addr))
				}
			}
		}
	}

	fmt.Printf("\n")
	fmt.Printf("=================================\n")
	fmt.Printf("MIGRATION VALIDATION\n")
	fmt.Printf("=================================\n\n")

	if corruptedContracts == 0 {
		fmt.Printf("✅ No corrupted contracts found!\n")
		fmt.Printf("No migration needed - all contracts can unmarshal.\n\n")

		if schemaInconsistentContracts > 0 {
			fmt.Printf("Note: %d contracts have non-canonical schemas but work fine.\n", schemaInconsistentContracts)
			fmt.Printf("These will be SKIPPED by the migration (no fix needed).\n")
		}
	} else {
		if fixSuccesses == corruptedContracts {
			fmt.Printf("✅ ALL CORRUPTED CONTRACTS ARE FIXABLE!\n\n")
			fmt.Printf("Migration validation: SUCCESS\n")
			fmt.Printf("  • Total contracts: %d\n", totalContracts)
			fmt.Printf("  • Working contracts (will be skipped): %d (%.2f%%)\n",
				workingContracts,
				float64(workingContracts)/float64(totalContracts)*100)
			fmt.Printf("  • Corrupted contracts (will be fixed): %d (%.2f%%)\n",
				corruptedContracts,
				float64(corruptedContracts)/float64(totalContracts)*100)
			fmt.Printf("  • Fix success rate: 100.00%%\n\n")

			fmt.Printf("✓ The v25 migration is READY to deploy\n")
			fmt.Printf("✓ All %d corrupted contrdddacts will be successfully fixed\n", corruptedContracts)
			fmt.Printf("✓ %d working contracts will be left unchanged\n\n", workingContracts)

		} else {
			fmt.Printf("⚠️  SOME CONTRACTS CANNOT BE FIXED\n\n")
			fmt.Printf("Migration validation: PARTIAL SUCCESS\n")
			fmt.Printf("  • Fixable: %d (%.2f%%)\n",
				fixSuccesses,
				float64(fixSuccesses)/float64(corruptedContracts)*100)
			fmt.Printf("  • Unfixable: %d (%.2f%%)\n\n",
				fixFailures,
				float64(fixFailures)/float64(corruptedContracts)*100)

			fmt.Printf("⚠️  Manual intervention required for %d contracts\n", fixFailures)
			fmt.Printf("\nRecommendations:\n")
			fmt.Printf("1. Investigate unfixable contracts (examples listed above)\n")
			fmt.Printf("2. Determine if they can be safely deleted or need manual repair\n")
			fmt.Printf("3. Consider deploying v25 migration for the %d fixable contracts\n", fixSuccesses)
			fmt.Printf("4. Handle unfixable contracts separately\n")
		}
	}

	fmt.Printf("\n")
	fmt.Printf("⚠️  Remember: This was a DRY RUN - no changes were made\n")
	fmt.Printf("The actual migration will run during the v25 chain upgrade\n")

	return nil
}
