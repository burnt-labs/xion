package keeper

import (
	"context"
	"cosmossdk.io/math"
	"fmt"

	sdkerrors "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrortypes "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/hashicorp/go-metrics"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"cosmossdk.io/errors"
	"github.com/burnt-labs/xion/x/xion/types"
)

type msgServer struct {
	k Keeper
}

var _ types.MsgServer = msgServer{}

// NewMsgServerImpl returns an implementation of the module MsgServer interface.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{k: keeper}
}

func (ms msgServer) UpdateParams(ctx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if ms.k.authority != msg.Authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", ms.k.authority, msg.Authority)
	}

	return nil, ms.k.Params.Set(ctx, msg.Params)
}

func (ms msgServer) Send(goCtx context.Context, msg *types.MsgSend) (*types.MsgSendResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := ms.k.bankKeeper.IsSendEnabledCoins(ctx, msg.Amount...); err != nil {
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

	if ms.k.bankKeeper.BlockedAddr(to) {
		return nil, sdkerrors.Wrapf(sdkerrortypes.ErrUnauthorized, "%s is not allowed to receive funds", msg.ToAddress)
	}

	percentage, err := ms.k.GetPlatformPercentage(ctx)
	if err != nil {
		return nil, err
	}
	throughCoins := msg.Amount

	if !percentage.IsZero() {
		platformCoins := msg.Amount.MulInt(percentage).QuoInt(math.NewInt(10000))
		throughCoins = throughCoins.Sub(platformCoins...)

		if err := ms.k.bankKeeper.SendCoinsFromAccountToModule(ctx, from, authtypes.FeeCollectorName, platformCoins); err != nil {
			return nil, err
		}
	}

	err = ms.k.bankKeeper.SendCoins(ctx, from, to, throughCoins)
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

func (ms msgServer) MultiSend(goCtx context.Context, msg *types.MsgMultiSend) (*types.MsgMultiSendResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// NOTE: totalIn == totalOut should already have been checked
	for _, in := range msg.Inputs {
		if err := ms.k.bankKeeper.IsSendEnabledCoins(ctx, in.Coins...); err != nil {
			return nil, err
		}
	}

	percentage, err := ms.k.GetPlatformPercentage(ctx)
	if err != nil {
		return nil, err
	}
	var outputs []banktypes.Output
	totalPlatformCoins := sdk.NewCoins()

	for _, out := range msg.Outputs {
		accAddr := sdk.MustAccAddressFromBech32(out.Address)

		if ms.k.bankKeeper.BlockedAddr(accAddr) {
			return nil, sdkerrors.Wrapf(sdkerrortypes.ErrUnauthorized, "%s is not allowed to receive funds", out.Address)
		}

		// if there is a platform fee set, reduce it from each output
		if !percentage.IsZero() {
			platformCoins := out.Coins.MulInt(percentage).QuoInt(math.NewInt(10000))
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
		feeCollectorAcc := ms.k.accountKeeper.GetModuleAccount(ctx, authtypes.FeeCollectorName).GetAddress()
		outputs = append(outputs, banktypes.NewOutput(feeCollectorAcc, totalPlatformCoins))
	}

	err = ms.k.bankKeeper.InputOutputCoins(ctx, msg.Inputs[0], outputs)
	if err != nil {
		return nil, err
	}

	return &types.MsgMultiSendResponse{}, nil
}
