package keeper

import (
	"cosmossdk.io/math"
	"github.com/burnt-labs/xion/x/xion/types"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	mintkeeper "github.com/cosmos/cosmos-sdk/x/mint/keeper"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"time"
)

const (
	AttributeKeyCollectedAmount = "collected_amount"
	AttributeKeyMintedAmount    = "minted_amount"
	AttributeKeyBurnedAmount    = "burned_amount"
)

func StakedInflationMintFn(feeCollectorName string, ic minttypes.InflationCalculationFn, bankKeeper types.BankKeeper, accountKeeper types.AccountKeeper, stakingKeeper types.StakingKeeper) func(ctx sdk.Context, k *mintkeeper.Keeper) error {
	return func(ctx sdk.Context, k *mintkeeper.Keeper) error {
		defer telemetry.ModuleMeasureSince(minttypes.ModuleName, time.Now(), telemetry.MetricKeyBeginBlocker)

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
		collectedFeeCoin := bankKeeper.GetBalance(ctx, accountKeeper.GetModuleAccount(ctx, feeCollectorName).GetAddress(), params.MintDenom)

		// recalculate inflation rate
		bondedRatio, err := k.BondedRatio(ctx)
		if err != nil {
			return err
		}
		minter.Inflation = ic(ctx, minter, params, bondedRatio)

		bondedTokenSupply, err := stakingKeeper.TotalBondedTokens(ctx)
		if err != nil {
			return err
		}
		minter.AnnualProvisions = minter.NextAnnualProvisions(params, bondedTokenSupply)
		if err := k.Minter.Set(ctx, minter); err != nil {
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
				return err
			}

			// send the minted coins to the fee collector account
			err = k.AddCollectedFees(ctx, mintedCoins)
			if err != nil {
				return err
			}

			if mintedCoin.Amount.IsInt64() {
				defer telemetry.ModuleSetGauge(types.ModuleName, float32(mintedCoin.Amount.Int64()), "minted_tokens")
			}

		} else {
			// if the fee collector has collected more fees than are needed to meet the
			// staking incentive goals, burn the rest.
			burnedCoin = collectedFeeCoin.Sub(neededCoin)
			burnedCoins := sdk.NewCoins(burnedCoin)
			err := bankKeeper.BurnCoins(ctx, feeCollectorName, burnedCoins)
			if err != nil {
				return err
			}
		}

		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				minttypes.EventTypeMint,
				sdk.NewAttribute(minttypes.AttributeKeyBondedRatio, bondedRatio.String()),
				sdk.NewAttribute(minttypes.AttributeKeyInflation, minter.Inflation.String()),
				sdk.NewAttribute(minttypes.AttributeKeyAnnualProvisions, minter.AnnualProvisions.String()),
				sdk.NewAttribute(AttributeKeyMintedAmount, mintedCoin.Amount.String()),
				sdk.NewAttribute(AttributeKeyCollectedAmount, collectedFeeCoin.Amount.String()),
				sdk.NewAttribute(AttributeKeyBurnedAmount, burnedCoin.Amount.String()),
			),
		)

		return nil
	}
}
