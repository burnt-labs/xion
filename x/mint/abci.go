package mint

import (
	"context"
	"cosmossdk.io/math"
	"time"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/mint/keeper"
	"github.com/burnt-labs/xion/x/mint/types"
)

// BeginBlocker mints new tokens for the previous block.
func BeginBlocker(goCtx context.Context, k keeper.Keeper, ic types.InflationCalculationFn) error {
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), telemetry.MetricKeyBeginBlocker)
	ctx := sdk.UnwrapSDKContext(goCtx)

	// fetch stored minter & params
	minter, err := k.Minter.Get(ctx)
	if err != nil {
		return err
	}

	params, err := k.Params.Get(ctx)
	if err != nil {
		return err
	}

	// fetch collected fees
	collectedFeeCoin := k.CountCollectedFees(ctx, params.MintDenom)

	// recalculate inflation rate
	totalBondedSupply, err := k.BondedTokenSupply(ctx)
	if err != nil {
		return err
	}

	bondedRatio, err := k.BondedRatio(ctx)
	if err != nil {
		return err
	}

	minter.Inflation = ic(ctx, minter, params, bondedRatio)
	minter.AnnualProvisions = minter.NextAnnualProvisions(params, totalBondedSupply)
	if err = k.Minter.Set(ctx, minter); err != nil {
		return err
	}

	// mint coins, update supply
	neededCoin := minter.BlockProvision(params)
	mintedCoin := sdk.NewCoin(params.MintDenom, math.ZeroInt())
	burnedCoin := sdk.NewCoin(params.MintDenom, math.ZeroInt())

	if collectedFeeCoin.IsLT(neededCoin) {
		// if the fee collector has not collected enough fees to meet the
		// staking incentive goals, mint enough to meet.
		mintedCoin = neededCoin.Sub(collectedFeeCoin)
		mintedCoins := sdk.NewCoins(mintedCoin)

		err := k.MintCoins(ctx, mintedCoins)
		if err != nil {
			panic(err)
		}

		// send the minted coins to the fee collector account
		err = k.AddCollectedFees(ctx, mintedCoins)
		if err != nil {
			panic(err)
		}

		if mintedCoin.Amount.IsInt64() {
			defer telemetry.ModuleSetGauge(types.ModuleName, float32(mintedCoin.Amount.Int64()), "minted_tokens")
		}

	} else {
		// if the fee collector has collected more fees than are needed to meet the
		// staking incentive goals, burn the rest.
		burnedCoin = collectedFeeCoin.Sub(neededCoin)
		burnedCoins := sdk.NewCoins(burnedCoin)
		err := k.BurnFees(ctx, burnedCoins)
		if err != nil {
			panic(err)
		}
	}

	mintEvent := types.MintIncentiveTokens{
		BondedRatio:      bondedRatio,
		Inflation:        minter.Inflation,
		AnnualProvisions: minter.AnnualProvisions,
		NeededAmount:     neededCoin.Amount.Uint64(),
		CollectedAmount:  collectedFeeCoin.Amount.Uint64(),
		MintedAmount:     mintedCoin.Amount.Uint64(),
		BurnedAmount:     burnedCoin.Amount.Uint64(),
	}
	if err := ctx.EventManager().EmitTypedEvent(&mintEvent); err != nil {
		k.Logger(ctx).Error("error emitting event",
			"error", err,
			"event", mintEvent)
	}

	return nil
}
