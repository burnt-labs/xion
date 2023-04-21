package keeper

import (
	"context"
	"fmt"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/burnt-labs/xion/x/xion/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
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

func (k msgServer) Send(goCtx context.Context, msg *types.MsgSend) (*types.MsgSendResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	percentage := k.GetParamSet(ctx).PlatformPercentage
	throughCoins := msg.Amount

	if percentage > 0 {
		sendAddress, err := sdk.AccAddressFromBech32(msg.FromAddress)
		if err != nil {
			return nil, err
		}

		platformCoins := msg.Amount.MulInt(sdk.NewIntFromUint64(uint64(percentage))).QuoInt(sdk.NewInt(10000))
		throughCoins, ok := throughCoins.SafeSub(platformCoins...)
		if !ok {
			return nil, fmt.Errorf("unable to subtract %v from %v", platformCoins, throughCoins)
		}

		if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, sendAddress, authtypes.FeeCollectorName, platformCoins); err != nil {
			return nil, err
		}
	}

	newMsg := banktypes.MsgSend{
		FromAddress: msg.FromAddress,
		ToAddress:   msg.ToAddress,
		Amount:      throughCoins,
	}

	if _, err := k.bankKeeper.Send(goCtx, &newMsg); err != nil {
		return nil, err
	}

	return &types.MsgSendResponse{}, nil
}

func (k msgServer) MultiSend(goCtx context.Context, msg *types.MsgMultiSend) (*types.MsgMultiSendResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if k.GetParamSet(ctx).PlatformPercentage > 0 {
		feeCollectorAcc := k.accountKeeper.GetModuleAccount(ctx, authtypes.FeeCollectorName).GetAddress()
		percentage := sdk.NewIntFromUint64(uint64(k.GetParamSet(ctx).PlatformPercentage))
		var newOutputs []banktypes.Output
		for _, output := range msg.Outputs {
			platformCoins := output.Coins.MulInt(percentage).QuoInt(sdk.NewInt(10000))
			throughCoins, ok := output.Coins.SafeSub(platformCoins...)
			if !ok {
				return nil, fmt.Errorf("unable to subtract %v from %v", platformCoins, throughCoins)
			}

			newOutputs = append(newOutputs,
				banktypes.NewOutput(feeCollectorAcc, platformCoins),
				banktypes.Output{
					Address: output.Address,
					Coins:   throughCoins,
				})
		}

		if _, err := k.bankKeeper.MultiSend(goCtx, banktypes.NewMsgMultiSend(msg.Inputs, newOutputs)); err != nil {
			return nil, err
		}

	} else {
		newMsg := banktypes.MsgMultiSend{
			Inputs:  msg.Inputs,
			Outputs: msg.Outputs,
		}
		if _, err := k.bankKeeper.MultiSend(goCtx, &newMsg); err != nil {
			return nil, err
		}
	}

	return &types.MsgMultiSendResponse{}, nil
}
