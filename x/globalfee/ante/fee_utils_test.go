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
