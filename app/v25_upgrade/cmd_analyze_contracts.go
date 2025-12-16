package v25_upgrade

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	dbm "github.com/cosmos/cosmos-db"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func AnalyzeContractsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analyze-contracts",
		Short: "Analyze all contracts and report contracts patterns",
		Long: `Scans all contracts in the database and analyzes contracts patterns.

This command:
1. Scans all contracts in the database
2. Attempts to unmarshal each one
3. Groups contracts by state (Healthy, UnmarshalFails, SchemaInconsistent, etc.)
4. Analyzes contracts patterns for failed contracts
5. Generates a comprehensive report

This is useful for understanding the scope of contracts before running a migration.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			homeDir, err := cmd.Flags().GetString("home")
			if err != nil {
				return err
			}

			limit, err := cmd.Flags().GetInt("limit")
			if err != nil {
				return err
			}

			showExamples, err := cmd.Flags().GetBool("show-examples")
			if err != nil {
				return err
			}

			testFixes, err := cmd.Flags().GetBool("test-fixes")
			if err != nil {
				return err
			}

			return runAnalyzeContracts(homeDir, limit, showExamples, testFixes)
		},
	}

	cmd.Flags().Int("limit", 0, "Max contracts to analyze (0 = all)")
	cmd.Flags().Bool("show-examples", true, "Show example addresses for each category")
	cmd.Flags().Bool("test-fixes", false, "Test fixes on corrupted contracts to verify repairability")

	return cmd
}

func runAnalyzeContracts(homeDir string, limit int, showExamples bool, testFixes bool) error {
	fmt.Printf("Contract Contracts Analysis\n")
	fmt.Printf("=============================\n\n")

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

	fmt.Printf("✓ Database opened\n")
	fmt.Printf("Using contract prefix: 0x%02x\n\n", ContractKeyPrefix)
	fmt.Printf("Scanning contracts...\n\n")

	// Track statistics
	var totalContracts int
	stateDistribution := make(map[ContractState]int)
	patternDistribution := make(map[CorruptionPattern]int)

	// Track fix test results
	var fixTestAttempts int
	var fixTestSuccesses int
	var fixTestFailures int
	fixFailureExamples := make([]string, 0)

	// Track examples for each category
	examplesByState := make(map[ContractState][]string)
	examplesByPattern := make(map[CorruptionPattern][]string)

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

		// Analyze contract
		analysis := AnalyzeContract(address, data)

		// Update statistics
		stateDistribution[analysis.State]++
		if analysis.State == StateUnmarshalFails {
			patternDistribution[analysis.CorruptionPattern]++

			// Save examples (limit to 5 per pattern)
			if len(examplesByPattern[analysis.CorruptionPattern]) < 5 {
				examplesByPattern[analysis.CorruptionPattern] = append(
					examplesByPattern[analysis.CorruptionPattern],
					address,
				)
			}

			// Test fix if requested
			if testFixes {
				fixTestAttempts++
				result := FixContract(address, data)
				if result.FixSucceeded && result.UnmarshalAfter {
					fixTestSuccesses++
				} else {
					fixTestFailures++
					if len(fixFailureExamples) < 5 {
						fixFailureExamples = append(fixFailureExamples, address)
					}
				}
			}
		}

		// Save examples for each state (limit to 5 per state)
		if len(examplesByState[analysis.State]) < 5 {
			examplesByState[analysis.State] = append(examplesByState[analysis.State], address)
		}

		// Progress indicator
		if totalContracts%1000 == 0 {
			fmt.Printf("  Analyzed %d contracts...\n", totalContracts)
		}
	}

	// Print report
	fmt.Printf("\n")
	fmt.Printf("=================================\n")
	fmt.Printf("ANALYSIS REPORT\n")
	fmt.Printf("=================================\n\n")

	fmt.Printf("Total Contracts: %d\n\n", totalContracts)

	// State Distribution
	fmt.Printf("Contract State Distribution:\n")
	fmt.Printf("-----------------------------\n")

	states := []ContractState{
		StateHealthy,
		StateSchemaInconsistent,
		StateUnmarshalFails,
		StateUnfixable,
	}

	for _, state := range states {
		count := stateDistribution[state]
		if count == 0 && state != StateHealthy {
			continue
		}
		percentage := float64(count) / float64(totalContracts) * 100
		fmt.Printf("  %s: %d (%.2f%%)\n", state, count, percentage)

		if showExamples && len(examplesByState[state]) > 0 {
			fmt.Printf("    Examples:\n")
			for _, addr := range examplesByState[state] {
				fmt.Printf("      - %s\n", FormatAddress(addr))
			}
		}
		fmt.Printf("\n")
	}

	// Contracts Patterns (if any)
	if stateDistribution[StateUnmarshalFails] > 0 {
		fmt.Printf("Contracts Pattern Distribution:\n")
		fmt.Printf("--------------------------------\n")

		patterns := []CorruptionPattern{
			PatternInvalidWireType,
			PatternTruncatedField,
			PatternMalformedLength,
			PatternFieldNumberCorruption,
			PatternMissingRequiredFields,
			PatternDuplicateFields,
			PatternUnknown,
		}

		for _, pattern := range patterns {
			count := patternDistribution[pattern]
			if count == 0 {
				continue
			}
			percentage := float64(count) / float64(stateDistribution[StateUnmarshalFails]) * 100
			fmt.Printf("  %s: %d (%.2f%%)\n", pattern, count, percentage)

			if showExamples && len(examplesByPattern[pattern]) > 0 {
				fmt.Printf("    Examples:\n")
				for _, addr := range examplesByPattern[pattern] {
					fmt.Printf("      - %s\n", FormatAddress(addr))
				}
			}
			fmt.Printf("\n")
		}
	}

	// Summary
	fmt.Printf("=================================\n")
	fmt.Printf("SUMMARY\n")
	fmt.Printf("=================================\n\n")

	healthyCount := stateDistribution[StateHealthy]
	schemaInconsistentCount := stateDistribution[StateSchemaInconsistent]
	unmarshalFailsCount := stateDistribution[StateUnmarshalFails]
	unfixableCount := stateDistribution[StateUnfixable]

	// Only contracts that fail to unmarshal need fixing
	// SchemaInconsistent contracts can already unmarshal and work fine
	workingContracts := healthyCount + schemaInconsistentCount
	brokenContracts := unmarshalFailsCount

	fmt.Printf("✓ Working (can unmarshal): %d (%.2f%%)\n",
		workingContracts,
		float64(workingContracts)/float64(totalContracts)*100)

	if healthyCount > 0 {
		fmt.Printf("   - Canonical schema: %d (%.2f%%)\n",
			healthyCount,
			float64(healthyCount)/float64(totalContracts)*100)
	}

	if schemaInconsistentCount > 0 {
		fmt.Printf("   - Non-canonical but functional: %d (%.2f%%)\n",
			schemaInconsistentCount,
			float64(schemaInconsistentCount)/float64(totalContracts)*100)
	}

	fmt.Printf("\n❌ Broken (cannot unmarshal): %d (%.2f%%)\n",
		brokenContracts,
		float64(brokenContracts)/float64(totalContracts)*100)

	if unfixableCount > 0 {
		fmt.Printf("❌ Unfixable: %d (%.2f%%)\n",
			unfixableCount,
			float64(unfixableCount)/float64(totalContracts)*100)
	}

	fmt.Printf("\n")

	// Recommendations
	fmt.Printf("\n=================================\n")
	fmt.Printf("RECOMMENDATIONS\n")
	fmt.Printf("=================================\n\n")

	if brokenContracts == 0 && unfixableCount == 0 {
		fmt.Printf("✅ All contracts can unmarshal successfully!\n")
		fmt.Printf("No migration needed - chain can read all contract metadata.\n")
		if schemaInconsistentCount > 0 {
			fmt.Printf("\nNote: %d contracts have non-canonical schemas but work fine.\n", schemaInconsistentCount)
			fmt.Printf("These can be left as-is or normalized in a future optional migration.\n")
		}
	} else {
		if brokenContracts > 0 {
			fmt.Printf("⚠️  MIGRATION REQUIRED\n")
			fmt.Printf("------------------\n")
			fmt.Printf("%d contracts CANNOT unmarshal and must be fixed.\n\n", brokenContracts)

			// Show breakdown by pattern
			invalidWireTypeCount := patternDistribution[PatternInvalidWireType]
			if invalidWireTypeCount > 0 {
				fmt.Printf("Contracts patterns:\n")
				fmt.Printf("  • Invalid wire type: %d contracts (field swap from v20/v21)\n", invalidWireTypeCount)
			}

			truncatedCount := patternDistribution[PatternTruncatedField]
			if truncatedCount > 0 {
				fmt.Printf("  • Truncated fields: %d contracts\n", truncatedCount)
			}

			malformedCount := patternDistribution[PatternMalformedLength]
			if malformedCount > 0 {
				fmt.Printf("  • Malformed length: %d contracts\n", malformedCount)
			}

			unknownCount := patternDistribution[PatternUnknown]
			if unknownCount > 0 {
				fmt.Printf("  • Unknown pattern: %d contracts\n", unknownCount)
			}

			// Show fix test results if tested
			if testFixes && fixTestAttempts > 0 {
				fmt.Printf("\nFix Validation:\n")
				fmt.Printf("  ✓ Successfully fixable: %d (%.1f%%)\n",
					fixTestSuccesses,
					float64(fixTestSuccesses)/float64(fixTestAttempts)*100)
				if fixTestFailures > 0 {
					fmt.Printf("  ✗ Cannot fix: %d (%.1f%%)\n",
						fixTestFailures,
						float64(fixTestFailures)/float64(fixTestAttempts)*100)

					if len(fixFailureExamples) > 0 {
						fmt.Printf("\n  Examples of unfixable contracts:\n")
						for _, addr := range fixFailureExamples {
							fmt.Printf("    - %s\n", FormatAddress(addr))
						}
					}
				}
			}

			fmt.Printf("\n→ Run 'xiond migrate-v25' to fix the %d broken contracts\n", brokenContracts)
		}

		if schemaInconsistentCount > 0 {
			fmt.Printf("\nOptional schema normalization:\n")
			fmt.Printf("  %d contracts have non-canonical schemas but unmarshal successfully.\n", schemaInconsistentCount)
			fmt.Printf("  These contracts are FUNCTIONAL and do not need fixing.\n")
			fmt.Printf("  They can be normalized later if desired for consistency.\n")
		}
	}

	if unfixableCount > 0 {
		fmt.Printf("\n❌ %d contracts are unfixable - manual intervention required\n", unfixableCount)
		fmt.Printf("These contracts may need to be deleted or recreated\n")
	}

	return nil
}
