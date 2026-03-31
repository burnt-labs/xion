package keeper

import (
	"context"
	"fmt"
	"math"
	"math/big"

	"github.com/hashicorp/go-metrics"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/burnt-labs/xion/x/xion/types"
)

type msgServer struct {
	Keeper
}

var _ types.MsgServer = msgServer{}

// NewMsgServerImpl returns an implementation of the bank MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

// Send handles xion.v1.MsgSend with platform fees. Bank MsgSend intentionally has no platform fee.
func (k msgServer) Send(goCtx context.Context, msg *types.MsgSend) (*types.MsgSendResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := k.bankKeeper.IsSendEnabledCoins(ctx, msg.Amount...); err != nil {
		return nil, err
	}

	from, err := sdk.AccAddressFromBech32(msg.FromAddress)
	if err != nil {
		return nil, err
	}
	to, err := sdk.AccAddressFromBech32(msg.ToAddress)
	if err != nil {
		return nil, err
	}

	if k.bankKeeper.BlockedAddr(to) {
		return nil, errorsmod.Wrapf(sdkerrors.ErrUnauthorized, "%s is not allowed to receive funds", msg.ToAddress)
	}

	percentage := k.GetPlatformPercentage(ctx)
	minimums, err := k.GetPlatformMinimums(ctx)
	if err != nil {
		return nil, err
	}
	throughCoins := msg.Amount
	// Enforce per-denom minimums: for any denom that has a configured minimum,
	// the sent amount for that denom must be >= that minimum.
	if !meetsConfiguredMinimums(msg.Amount, minimums) {
		return nil, errorsmod.Wrapf(types.ErrMinimumNotMet, "received %v, needed at least %v", msg.Amount, minimums)
	}

	if !percentage.IsZero() {
		// Safe calculation to prevent overflow: use multiplication with bounds checking
		// For each coin, calculate: (amount * percentage) / 10000
		// But prevent overflow by checking if amount * percentage would overflow
		platformCoins := getPlatformCoins(msg.Amount, percentage)
		throughCoins = throughCoins.Sub(platformCoins...)

		if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, from, authtypes.FeeCollectorName, platformCoins); err != nil {
			return nil, err
		}
	}

	err = k.bankKeeper.SendCoins(ctx, from, to, throughCoins)
	if err != nil {
		return nil, err
	}

	defer func() {
		for _, a := range throughCoins {
			if a.Amount.IsInt64() {
				telemetry.SetGaugeWithLabels(
					[]string{"tx", "msg", "send"},
					float32(a.Amount.Int64()),
					[]metrics.Label{telemetry.NewLabel("denom", a.Denom)},
				)
			}
		}
	}()

	return &types.MsgSendResponse{}, nil
}

func (k msgServer) MultiSend(goCtx context.Context, msg *types.MsgMultiSend) (*types.MsgMultiSendResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// NOTE: totalIn == totalOut should already have been checked
	for _, in := range msg.Inputs {
		if err := k.bankKeeper.IsSendEnabledCoins(ctx, in.Coins...); err != nil {
			return nil, err
		}
	}

	percentage := k.GetPlatformPercentage(ctx)
	minimums, err := k.GetPlatformMinimums(ctx)
	if err != nil {
		return nil, err
	}
	var outputs []banktypes.Output
	totalPlatformCoins := sdk.NewCoins()

	// Enforce per-denom minimums on the input: for any denom that has a configured minimum,
	// the input amount for that denom must be >= that minimum.
	if !meetsConfiguredMinimums(msg.Inputs[0].Coins, minimums) {
		return nil, errorsmod.Wrapf(types.ErrMinimumNotMet, "received %v, needed at least %v", msg.Inputs[0].Coins, minimums)
	}

	for _, out := range msg.Outputs {
		accAddr, err := sdk.AccAddressFromBech32(out.Address)
		if err != nil {
			return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid output address %s: %s", out.Address, err)
		}

		if k.bankKeeper.BlockedAddr(accAddr) {
			return nil, errorsmod.Wrapf(sdkerrors.ErrUnauthorized, "%s is not allowed to receive funds", out.Address)
		}

		// if there is a platform fee set, reduce it from each output
		if !percentage.IsZero() {
			// Safe calculation to prevent overflow: use multiplication with bounds checking
			platformCoins := getPlatformCoins(out.Coins, percentage)
			throughCoins, wentNegative := out.Coins.SafeSub(platformCoins...)
			if wentNegative {
				return nil, fmt.Errorf("unable to subtract %v from %v", platformCoins, throughCoins)
			}

			outputs = append(outputs, banktypes.NewOutput(accAddr, throughCoins))
			totalPlatformCoins = totalPlatformCoins.Add(platformCoins...)
		} else {
			outputs = append(outputs, out)
		}
	}

	// if there is a platform fee set, create the final total output for module account
	if !totalPlatformCoins.IsZero() {
		feeCollectorAcc := k.accountKeeper.GetModuleAccount(ctx, authtypes.FeeCollectorName).GetAddress()
		outputs = append(outputs, banktypes.NewOutput(feeCollectorAcc, totalPlatformCoins))
	}

	err = k.bankKeeper.InputOutputCoins(ctx, msg.Inputs[0], outputs)
	if err != nil {
		return nil, err
	}

	return &types.MsgMultiSendResponse{}, nil
}

// meetsConfiguredMinimums returns true if every coin in amt has a configured minimum
// and meets it.  Denoms not present in mins are rejected: the minimums list acts as
// an explicit allowlist so that unconfigured denoms cannot bypass the minimum check.
// If no minimums are configured at all, this returns false to maintain backwards
// compatibility requiring platform minimums to be explicitly set.
func meetsConfiguredMinimums(amt sdk.Coins, mins sdk.Coins) bool {
	// Require that platform minimums be explicitly set (backwards compatibility)
	if len(mins) == 0 {
		return false
	}

	// Build a map for O(1) minimum lookups
	minMap := make(map[string]sdkmath.Int, len(mins))
	for _, m := range mins {
		minMap[m.Denom] = m.Amount
	}

	for _, c := range amt {
		min, ok := minMap[c.Denom]
		if !ok {
			// Denom has no configured minimum — reject it.
			// Unconfigured denoms are not permitted when minimums are in use,
			// preventing bypass by sending only non-listed denominations.
			return false
		}
		if !min.IsZero() && c.Amount.LT(min) {
			return false
		}
	}
	return true
}

func (k msgServer) SetPlatformPercentage(goCtx context.Context, msg *types.MsgSetPlatformPercentage) (*types.MsgSetPlatformPercentageResponse, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	if msg.PlatformPercentage > 10000 {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "platform percentage %d exceeds maximum 10000 (100%%)", msg.PlatformPercentage)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	k.OverwritePlatformPercentage(ctx, msg.PlatformPercentage)

	return &types.MsgSetPlatformPercentageResponse{}, nil
}

func (k msgServer) SetPlatformMinimum(goCtx context.Context, msg *types.MsgSetPlatformMinimum) (*types.MsgSetPlatformMinimumResponse, error) {
	if k.GetAuthority() != msg.Authority {
		return nil, errorsmod.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), msg.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	err := k.OverwritePlatformMinimum(ctx, msg.Minimums)

	return &types.MsgSetPlatformMinimumResponse{}, err
}

func getPlatformCoins(coins sdk.Coins, percentage sdkmath.Int) sdk.Coins {
	var platformCoins sdk.Coins

	// Handle zero percentage case
	if percentage.IsZero() {
		for _, coin := range coins {
			platformCoins = platformCoins.Add(sdk.NewCoin(coin.Denom, sdkmath.ZeroInt()))
		}
		return platformCoins
	}

	for _, coin := range coins {
		maxSafeAmount := sdkmath.NewIntFromUint64(math.MaxUint64).Quo(percentage)
		if coin.Amount.GT(maxSafeAmount) {
			// Use big integer arithmetic to prevent overflow
			bigAmount := coin.Amount.BigInt()
			bigPercentage := percentage.BigInt()
			bigDivisor := sdkmath.NewInt(10000).BigInt()

			// Ceiling division: add (divisor - 1) before dividing to round up.
			bigResult := new(big.Int).Mul(bigAmount, bigPercentage)
			bigResult.Add(bigResult, new(big.Int).Sub(bigDivisor, big.NewInt(1)))
			bigResult = bigResult.Quo(bigResult, bigDivisor)

			platformCoins = platformCoins.Add(sdk.NewCoin(coin.Denom, sdkmath.NewIntFromBigInt(bigResult)))
		} else {
			// Ceiling division: ensures any non-zero output with a non-zero platform
			// percentage always contributes at least 1 unit of fee, removing the
			// incentive to split outputs into many small amounts to exploit rounding.
			feeAmount := coin.Amount.Mul(percentage).Add(sdkmath.NewInt(9999)).Quo(sdkmath.NewInt(10000))
			platformCoins = platformCoins.Add(sdk.NewCoin(coin.Denom, feeAmount))
		}
	}
	return platformCoins
}
