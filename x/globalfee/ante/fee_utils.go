package ante

import (
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// CombinedFeeRequirement returns the global fee and min_gas_price combined and sorted.
// Both globalFees and minGasPrices must be valid, but CombinedFeeRequirement
// does not validate them, so it may return 0denom.
// if globalfee is empty, CombinedFeeRequirement return sdk.Coins{}
func CombinedFeeRequirement(globalFees, minGasPrices sdk.DecCoins) (sdk.DecCoins, error) {
	// global fees should never be empty
	// since it has a default value using the staking module's bond denom
	if len(globalFees) == 0 {
		return sdk.DecCoins{}, errorsmod.Wrapf(sdkerrors.ErrNotFound, "global fee cannot be empty")
	}

	// empty min_gas_price
	if len(minGasPrices) == 0 {
		return globalFees, nil
	}

	// if min_gas_price denom is in globalfee, and the amount is higher than globalfee, add min_gas_price to allFees
	var allFees sdk.DecCoins
	for _, fee := range globalFees {
		// min_gas_price denom in global fee
		ok, c := Find(minGasPrices, fee.Denom)
		if ok && c.Amount.GT(fee.Amount) {
			allFees = append(allFees, c)
		} else {
			allFees = append(allFees, fee)
		}
	}

	return allFees.Sort(), nil
}

// Find replaces the functionality of Coins.Find from SDK v0.46.x
func Find(coins sdk.DecCoins, denom string) (bool, sdk.DecCoin) {
	switch len(coins) {
	case 0:
		return false, sdk.DecCoin{}

	case 1:
		coin := coins[0]
		if coin.Denom == denom {
			return true, coin
		}
		return false, sdk.DecCoin{}

	default:
		midIdx := len(coins) / 2 // 2:1, 3:1, 4:2
		coin := coins[midIdx]
		switch {
		case denom < coin.Denom:
			return Find(coins[:midIdx], denom)
		case denom == coin.Denom:
			return true, coin
		default:
			return Find(coins[midIdx+1:], denom)
		}
	}
}

// Returns the largest coins given 2 sets of coins
func MaxCoins(a, b sdk.DecCoins) sdk.DecCoins {
	if IsAllGT(a, b) {
		return a
	}
	return b
}

func IsAllGT(a, b sdk.DecCoins) bool {
	if len(a) == 0 {
		return false
	}

	if len(b) == 0 {
		return true
	}

	if !DenomsSubsetOf(b, a) {
		return false
	}

	for _, coinB := range b {
		amountA, amountB := a.AmountOf(coinB.Denom), coinB.Amount
		if !amountA.GT(amountB) {
			return false
		}
	}

	return true
}

func DenomsSubsetOf(a, b sdk.DecCoins) bool {
	// more denoms in B than in a
	if len(a) > len(b) {
		return false
	}

	for _, coin := range a {
		if b.AmountOf(coin.Denom).IsZero() {
			return false
		}
	}

	return true
}
