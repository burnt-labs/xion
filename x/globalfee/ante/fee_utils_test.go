package ante

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Note that in a real Gaia deployment all zero coins can be removed from minGasPrice.
// This sanitizing happens when the minGasPrice is set into the context.
// (see baseapp.SetMinGasPrices in gaia/cmd/root.go line 221)
func TestCombinedFeeRequirement(t *testing.T) {
	zeroCoin1 := sdk.NewDecCoin("photon", math.ZeroInt())
	zeroCoin2 := sdk.NewDecCoin("stake", math.ZeroInt())
	zeroCoin3 := sdk.NewDecCoin("quark", math.ZeroInt())
	coin1 := sdk.NewDecCoin("photon", math.NewInt(1))
	coin2 := sdk.NewDecCoin("stake", math.NewInt(2))
	coin1High := sdk.NewDecCoin("photon", math.NewInt(10))
	coin2High := sdk.NewDecCoin("stake", math.NewInt(20))
	coinNewDenom1 := sdk.NewDecCoin("Newphoton", math.NewInt(1))
	coinNewDenom2 := sdk.NewDecCoin("Newstake", math.NewInt(1))
	// coins must be valid !!! and sorted!!!
	coinsEmpty := sdk.DecCoins{}
	coinsNonEmpty := sdk.DecCoins{coin1, coin2}.Sort()
	coinsNonEmptyHigh := sdk.DecCoins{coin1High, coin2High}.Sort()
	coinsNonEmptyOneHigh := sdk.DecCoins{coin1High, coin2}.Sort()
	coinsNewDenom := sdk.DecCoins{coinNewDenom1, coinNewDenom2}.Sort()
	coinsNewOldDenom := sdk.DecCoins{coin1, coinNewDenom1}.Sort()
	coinsNewOldDenomHigh := sdk.DecCoins{coin1High, coinNewDenom1}.Sort()
	coinsCointainZero := sdk.DecCoins{coin1, zeroCoin2}.Sort()
	coinsCointainZeroNewDenom := sdk.DecCoins{coin1, zeroCoin3}.Sort()
	coinsAllZero := sdk.DecCoins{zeroCoin1, zeroCoin2}.Sort()
	tests := map[string]struct {
		cGlobal  sdk.DecCoins
		c        sdk.DecCoins
		combined sdk.DecCoins
	}{
		"global fee invalid, return combined fee empty and non-nil error": {
			cGlobal:  coinsEmpty,
			c:        coinsEmpty,
			combined: coinsEmpty,
		},
		"global fee nonempty, min fee empty, combined fee = global fee": {
			cGlobal:  coinsNonEmpty,
			c:        coinsNonEmpty,
			combined: coinsNonEmpty,
		},
		"global fee and min fee have overlapping denom, min fees amounts are all higher": {
			cGlobal:  coinsNonEmpty,
			c:        coinsNonEmptyHigh,
			combined: coinsNonEmptyHigh,
		},
		"global fee and min fee have overlapping denom, one of min fees amounts is higher": {
			cGlobal:  coinsNonEmpty,
			c:        coinsNonEmptyOneHigh,
			combined: coinsNonEmptyOneHigh,
		},
		"global fee and min fee have no overlapping denom, combined fee = global fee": {
			cGlobal:  coinsNonEmpty,
			c:        coinsNewDenom,
			combined: coinsNonEmpty,
		},
		"global fees and min fees have partial overlapping denom, min fee amount <= global fee amount, combined fees = global fees": {
			cGlobal:  coinsNonEmpty,
			c:        coinsNewOldDenom,
			combined: coinsNonEmpty,
		},
		"global fees and min fees have partial overlapping denom, one min fee amount > global fee amount, combined fee = overlapping highest": {
			cGlobal:  coinsNonEmpty,
			c:        coinsNewOldDenomHigh,
			combined: sdk.DecCoins{coin1High, coin2},
		},
		"global fees have zero fees, min fees have overlapping non-zero fees, combined fees = overlapping highest": {
			cGlobal:  coinsCointainZero,
			c:        coinsNonEmpty,
			combined: sdk.DecCoins{coin1, coin2},
		},
		"global fees have zero fees, min fees have overlapping zero fees": {
			cGlobal:  coinsCointainZero,
			c:        coinsCointainZero,
			combined: coinsCointainZero,
		},
		"global fees have zero fees, min fees have non-overlapping zero fees": {
			cGlobal:  coinsCointainZero,
			c:        coinsCointainZeroNewDenom,
			combined: coinsCointainZero,
		},
		"global fees are all zero fees, min fees have overlapping zero fees": {
			cGlobal:  coinsAllZero,
			c:        coinsAllZero,
			combined: coinsAllZero,
		},
		"global fees are all zero fees, min fees have overlapping non-zero fees, combined fee = overlapping highest": {
			cGlobal:  coinsAllZero,
			c:        coinsCointainZeroNewDenom,
			combined: sdk.DecCoins{coin1, zeroCoin2},
		},
		"global fees are all zero fees, fees have one overlapping non-zero fee": {
			cGlobal:  coinsAllZero,
			c:        coinsCointainZero,
			combined: coinsCointainZero,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			allFees, err := CombinedFeeRequirement(test.cGlobal, test.c)
			if len(test.cGlobal) == 0 {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, test.combined, allFees)
		})
	}
}

func TestMaxCoins(t *testing.T) {
	// Test with empty coins
	coins1 := sdk.DecCoins{}
	coins2 := sdk.DecCoins{}
	result := MaxCoins(coins1, coins2)
	require.Equal(t, sdk.DecCoins{}, result)

	// Test with one empty, one non-empty
	coins1 = sdk.DecCoins{}
	coins2 = sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(1, 3))}
	result = MaxCoins(coins1, coins2)
	require.Equal(t, coins2, result)

	// Test with both non-empty, different denoms
	coins1 = sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(2, 3))}
	coins2 = sdk.DecCoins{sdk.NewDecCoinFromDec("stake", math.LegacyNewDecWithPrec(1, 3))}
	result = MaxCoins(coins1, coins2)
	expected := sdk.DecCoins{
		sdk.NewDecCoinFromDec("stake", math.LegacyNewDecWithPrec(1, 3)),
		sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(2, 3)),
	}
	require.Equal(t, expected, result)

	// Test with same denom, different amounts
	coins1 = sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(2, 3))}
	coins2 = sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(1, 3))}
	result = MaxCoins(coins1, coins2)
	expected = sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(2, 3))}
	require.Equal(t, expected, result)

	// Test with multiple denoms - debug to understand AmountOf issue
	coin1Uatom := sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(2, 3))
	coin1Stake := sdk.NewDecCoinFromDec("stake", math.LegacyNewDecWithPrec(3, 3))
	coin2Uatom := sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(1, 3))
	coin2Stake := sdk.NewDecCoinFromDec("stake", math.LegacyNewDecWithPrec(4, 3))

	t.Logf("coin1Uatom: %s", coin1Uatom)
	t.Logf("coin1Stake: %s", coin1Stake)
	t.Logf("coin2Uatom: %s", coin2Uatom)
	t.Logf("coin2Stake: %s", coin2Stake)

	coins1 = sdk.DecCoins{coin1Uatom, coin1Stake}
	coins2 = sdk.DecCoins{coin2Uatom, coin2Stake}

	t.Logf("coins1 before sort: %s", coins1)
	t.Logf("coins2 before sort: %s", coins2)

	// Debug: check individual amounts
	t.Logf("coins1 uxion amount: %s", coins1.AmountOf("uxion"))
	t.Logf("coins1 stake amount: %s", coins1.AmountOf("stake"))
	t.Logf("coins2 uxion amount: %s", coins2.AmountOf("uxion"))
	t.Logf("coins2 stake amount: %s", coins2.AmountOf("stake"))

	result = MaxCoins(coins1, coins2)
	t.Logf("result: %s", result)

	// Expected: max(uxion: 0.002, 0.001) = 0.002, max(stake: 0.003, 0.004) = 0.004
	expected = sdk.DecCoins{
		sdk.NewDecCoinFromDec("stake", math.LegacyNewDecWithPrec(4, 3)),
		sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(2, 3)),
	}

	require.Equal(t, expected, result)
}

func TestIsAllGT(t *testing.T) {
	// Test with empty coins
	coins1 := sdk.DecCoins{}
	coins2 := sdk.DecCoins{}
	result := IsAllGT(coins1, coins2)
	require.False(t, result)

	// Test with one empty, one non-empty
	coins1 = sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(1, 3))}
	coins2 = sdk.DecCoins{}
	result = IsAllGT(coins1, coins2)
	require.True(t, result)

	// Test with same coins
	coins1 = sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(1, 3))}
	coins2 = sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(1, 3))}
	result = IsAllGT(coins1, coins2)
	require.False(t, result)

	// Test with greater coins
	coins1 = sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(2, 3))}
	coins2 = sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(1, 3))}
	result = IsAllGT(coins1, coins2)
	require.True(t, result)

	// Test with less coins
	coins1 = sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(1, 3))}
	coins2 = sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(2, 3))}
	result = IsAllGT(coins1, coins2)
	require.False(t, result)

	// Test with different denoms - coins1 has extra denom
	coins1 = sdk.DecCoins{
		sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(2, 3)),
		sdk.NewDecCoinFromDec("stake", math.LegacyNewDecWithPrec(1, 3)),
	}
	coins2 = sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(1, 3))}
	result = IsAllGT(coins1, coins2)
	require.False(t, result) // Different denoms means not all GT

	// Test with mixed results
	coins1 = sdk.DecCoins{
		sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(2, 3)),
		sdk.NewDecCoinFromDec("stake", math.LegacyNewDecWithPrec(1, 3)),
	}
	coins2 = sdk.DecCoins{
		sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(1, 3)),
		sdk.NewDecCoinFromDec("stake", math.LegacyNewDecWithPrec(2, 3)),
	}
	result = IsAllGT(coins1, coins2)
	require.False(t, result)
}

func TestDenomsSubsetOf(t *testing.T) {
	// Test with empty coins
	coins1 := sdk.DecCoins{}
	coins2 := sdk.DecCoins{}
	result := DenomsSubsetOf(coins1, coins2)
	require.True(t, result)

	// Test with empty subset
	coins1 = sdk.DecCoins{}
	coins2 = sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(1, 3))}
	result = DenomsSubsetOf(coins1, coins2)
	require.True(t, result)

	// Test with empty superset
	coins1 = sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(1, 3))}
	coins2 = sdk.DecCoins{}
	result = DenomsSubsetOf(coins1, coins2)
	require.False(t, result)

	// Test with same denoms
	coins1 = sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(1, 3))}
	coins2 = sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(2, 3))}
	result = DenomsSubsetOf(coins1, coins2)
	require.True(t, result)

	// Test with subset
	coins1 = sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(1, 3))}
	coins2 = sdk.DecCoins{
		sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(2, 3)),
		sdk.NewDecCoinFromDec("stake", math.LegacyNewDecWithPrec(1, 3)),
	}.Sort()
	result = DenomsSubsetOf(coins1, coins2)
	require.True(t, result)

	// Test with non-subset
	coins1 = sdk.DecCoins{
		sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(1, 3)),
		sdk.NewDecCoinFromDec("other", math.LegacyNewDecWithPrec(1, 3)),
	}.Sort()
	coins2 = sdk.DecCoins{sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(2, 3))}
	result = DenomsSubsetOf(coins1, coins2)
	require.False(t, result)

	// Test with multiple denoms subset
	coins1 = sdk.DecCoins{
		sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(1, 3)),
		sdk.NewDecCoinFromDec("stake", math.LegacyNewDecWithPrec(1, 3)),
	}.Sort()
	coins2 = sdk.DecCoins{
		sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(2, 3)),
		sdk.NewDecCoinFromDec("stake", math.LegacyNewDecWithPrec(2, 3)),
		sdk.NewDecCoinFromDec("other", math.LegacyNewDecWithPrec(1, 3)),
	}.Sort()
	result = DenomsSubsetOf(coins1, coins2)
	require.True(t, result)
}
