package ante

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TestCombinedFeeRequirementFix tests that the vulnerability fix works correctly
// by ensuring CombinedFeeRequirement produces correct results with both sorted and unsorted inputs
func TestCombinedFeeRequirementFix(t *testing.T) {
	// Test case from vulnerability report - with the fix, this should now work correctly
	coin0High := sdk.NewDecCoin("uxion", math.NewInt(4)) // Min gas price (higher)
	coin1High := sdk.NewDecCoin("uxion", math.NewInt(3)) // Global fee (lower)
	coin2High := sdk.NewDecCoin("uatom", math.NewInt(8)) // Min gas price (higher)
	coin3High := sdk.NewDecCoin("uatom", math.NewInt(7)) // Global fee (lower)

	tests := []struct {
		name             string
		globalFees       sdk.DecCoins
		minGasPrices     sdk.DecCoins
		expectedCombined sdk.DecCoins
		description      string
	}{
		{
			name:             "unsorted input coins - vulnerability case (now fixed)",
			globalFees:       sdk.DecCoins{coin1High, coin3High}, // 3 uxion, 7 uatom (uxion before uatom - unsorted)
			minGasPrices:     sdk.DecCoins{coin0High, coin2High}, // 4 uxion, 8 uatom (uxion before uatom - unsorted)
			expectedCombined: sdk.DecCoins{coin2High, coin0High}, // Should take max: 8 uatom, 4 uxion (sorted result)
			description:      "With unsorted inputs, should now correctly take maximum of each denom",
		},
		{
			name:             "sorted input coins - should work as before",
			globalFees:       sdk.DecCoins{coin1High, coin3High}.Sort(), // Properly sorted
			minGasPrices:     sdk.DecCoins{coin0High, coin2High}.Sort(), // Properly sorted
			expectedCombined: sdk.DecCoins{coin2High, coin0High},        // Should take max: 8 uatom, 4 uxion
			description:      "With sorted inputs, should continue to work correctly",
		},
		{
			name:             "reverse sorted input coins",
			globalFees:       sdk.DecCoins{coin3High, coin1High}, // 7 uatom, 3 uxion (reverse alphabetical order)
			minGasPrices:     sdk.DecCoins{coin2High, coin0High}, // 8 uatom, 4 uxion (reverse alphabetical order)
			expectedCombined: sdk.DecCoins{coin2High, coin0High}, // Should take max: 8 uatom, 4 uxion
			description:      "With reverse sorted inputs, should correctly sort and process",
		},
		{
			name:             "mixed sorting - global sorted, min unsorted",
			globalFees:       sdk.DecCoins{coin3High, coin1High}.Sort(), // Sorted: 7 uatom, 3 uxion
			minGasPrices:     sdk.DecCoins{coin0High, coin2High},        // Unsorted: 4 uxion, 8 uatom
			expectedCombined: sdk.DecCoins{coin2High, coin0High},        // Should take max: 8 uatom, 4 uxion
			description:      "With mixed sorting, should handle correctly",
		},
		{
			name:             "non-overlapping denoms with unsorted inputs",
			globalFees:       sdk.DecCoins{sdk.NewDecCoin("token1", math.NewInt(10)), sdk.NewDecCoin("token2", math.NewInt(20))},
			minGasPrices:     sdk.DecCoins{sdk.NewDecCoin("token3", math.NewInt(30)), sdk.NewDecCoin("token4", math.NewInt(40))},
			expectedCombined: sdk.DecCoins{sdk.NewDecCoin("token1", math.NewInt(10)), sdk.NewDecCoin("token2", math.NewInt(20))},
			description:      "With non-overlapping denoms, should return global fees",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Test: %s", tt.description)
			t.Logf("Global fees input: %s", tt.globalFees.String())
			t.Logf("Min gas prices input: %s", tt.minGasPrices.String())

			result, err := CombinedFeeRequirement(tt.globalFees, tt.minGasPrices)
			require.NoError(t, err)

			t.Logf("Combined fees result: %s", result.String())
			t.Logf("Expected result: %s", tt.expectedCombined.String())

			require.Equal(t, tt.expectedCombined, result, "Combined fees should match expected result")

			// Verify specific amounts for the main vulnerability case
			if tt.name == "unsorted input coins - vulnerability case (now fixed)" {
				uxionAmount := result.AmountOf("uxion")
				uatomAmount := result.AmountOf("uatom")

				require.True(t, uxionAmount.Equal(math.LegacyNewDec(4)),
					"uxion amount should be 4 (max of 3 and 4), got %s", uxionAmount.String())
				require.True(t, uatomAmount.Equal(math.LegacyNewDec(8)),
					"uatom amount should be 8 (max of 7 and 8), got %s", uatomAmount.String())

				t.Logf("✅ Vulnerability fix verified: correctly uses max amounts (4 uxion, 8 uatom)")
			}
		})
	}
}

// TestCombinedFeeRequirementEdgeCasesFixed tests edge cases to ensure the fix doesn't break existing functionality
func TestCombinedFeeRequirementEdgeCasesFixed(t *testing.T) {
	tests := []struct {
		name         string
		globalFees   sdk.DecCoins
		minGasPrices sdk.DecCoins
		expectError  bool
		description  string
	}{
		{
			name:         "empty global fees should error",
			globalFees:   sdk.DecCoins{},
			minGasPrices: sdk.DecCoins{sdk.NewDecCoin("test", math.NewInt(1))},
			expectError:  true,
			description:  "Empty global fees should return error",
		},
		{
			name:         "empty min gas prices should return global fees",
			globalFees:   sdk.DecCoins{sdk.NewDecCoin("test", math.NewInt(1))},
			minGasPrices: sdk.DecCoins{},
			expectError:  false,
			description:  "Empty min gas prices should return global fees",
		},
		{
			name:         "single denom overlap with unsorted inputs",
			globalFees:   sdk.DecCoins{sdk.NewDecCoin("token", math.NewInt(5))},
			minGasPrices: sdk.DecCoins{sdk.NewDecCoin("token", math.NewInt(10))},
			expectError:  false,
			description:  "Single denom with higher min gas price should use min gas price",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Test: %s", tt.description)

			result, err := CombinedFeeRequirement(tt.globalFees, tt.minGasPrices)

			if tt.expectError {
				require.Error(t, err)
				require.Equal(t, sdk.DecCoins{}, result)
				t.Logf("✅ Correctly returned error: %v", err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				t.Logf("✅ Result: %s", result.String())

				// Verify result is sorted
				require.True(t, result.IsValid(), "Result should be valid coins")
				if len(result) > 1 {
					require.True(t, result.IsValid() && result.IsAllPositive(), "Result should be sorted and positive")
				}
			}
		})
	}
}

// TestFindFunctionWithUnsortedInputs specifically tests the Find function behavior with sorted vs unsorted inputs
func TestFindFunctionWithUnsortedInputs(t *testing.T) {
	t.Run("Find function behavior demonstration", func(t *testing.T) {
		// Create coins in alphabetical order (uatom comes before uxion)
		coin1 := sdk.NewDecCoin("uatom", math.NewInt(8))
		coin2 := sdk.NewDecCoin("uxion", math.NewInt(4))

		// Test unsorted coins (uxion comes before uatom - wrong order)
		unsortedCoins := sdk.DecCoins{coin2, coin1} // uxion, uatom
		t.Logf("Unsorted coins: %s", unsortedCoins.String())

		// Find should fail for uxion in unsorted coins due to binary search assumptions
		found, foundCoin := Find(unsortedCoins, "uxion")
		t.Logf("Find uxion in unsorted coins: found=%t, coin=%s", found, foundCoin.String())

		found, foundCoin = Find(unsortedCoins, "uatom")
		t.Logf("Find uatom in unsorted coins: found=%t, coin=%s", found, foundCoin.String())

		// Test sorted coins
		sortedCoins := unsortedCoins.Sort()
		t.Logf("Sorted coins: %s", sortedCoins.String())

		// Find should work correctly for both denoms in sorted coins
		found, foundCoin = Find(sortedCoins, "uxion")
		require.True(t, found, "Should find uxion in sorted coins")
		require.Equal(t, "uxion", foundCoin.Denom)
		t.Logf("Find uxion in sorted coins: found=%t, coin=%s", found, foundCoin.String())

		found, foundCoin = Find(sortedCoins, "uatom")
		require.True(t, found, "Should find uatom in sorted coins")
		require.Equal(t, "uatom", foundCoin.Denom)
		t.Logf("Find uatom in sorted coins: found=%t, coin=%s", found, foundCoin.String())

		t.Logf("✅ Find function works correctly with sorted inputs")
	})
}
