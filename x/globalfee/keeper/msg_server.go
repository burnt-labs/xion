package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"

	"github.com/burnt-labs/xion/x/globalfee/types"
)

type msgServer struct {
	k Keeper
}

func NewMsgServerImpl(k Keeper) types.MsgServer {
	return msgServer{k: k}
}

func (ms msgServer) UpdateParams(ctx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if msg.Authority != ms.k.GetAuthority() {
		return nil, errorsmod.Wrapf(types.ErrInvalidAuthority, "expected %s, got %s", ms.k.GetAuthority(), msg.Authority)
	}
	if err := ms.k.SetParams(ctx, msg.Params); err != nil {
		return nil, err
	}
	return &types.MsgUpdateParamsResponse{}, nil
}
