package ante

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Test utility functions
func TestUtilityFunctions(t *testing.T) {
	// Test MaxCoins function
	coins1 := sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(1, 3))}
	coins2 := sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(2, 3))}
	result := MaxCoins(coins1, coins2)
	expected := sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(2, 3))}
	require.Equal(t, expected, result)

	// Test IsAllGT function
	require.True(t, IsAllGT(coins2, coins1))
	require.False(t, IsAllGT(coins1, coins2))
	require.False(t, IsAllGT(coins1, coins1))

	// Test DenomsSubsetOf function
	require.True(t, DenomsSubsetOf(coins1, coins1))
	subset := sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(1, 3))}
	superset := sdk.DecCoins{
		sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(1, 3)),
		sdk.NewDecCoinFromDec("stake", math.LegacyNewDecWithPrec(1, 3)),
	}.Sort() // Ensure coins are sorted
	require.True(t, DenomsSubsetOf(subset, superset))
	require.False(t, DenomsSubsetOf(superset, subset))
}
