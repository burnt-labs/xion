package keeper

import (
	"context"

	"github.com/burnt-labs/xion/x/grantmanager/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type msgServer struct {
	keeper Keeper
}

func NewMsgServer(keeper Keeper) types.MsgServer {
	return &msgServer{
		keeper: keeper,
	}
}

func (ms *msgServer) IterateAuthzGrants(ctx context.Context) {}

func (ms *msgServer) RevokeAuthzGrants(ctx context.Context, msg *types.MsgRevokeAuthzGrants) (*types.MsgRevokeAuthzGrantsResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	granter, err := sdk.AccAddressFromBech32(msg.Granter)
	if err != nil {
		return nil, err
	}
	ms.keeper.RevokeAuthzGrants(ctx, granter, 100)
	return &types.MsgRevokeAuthzGrantsResponse{}, nil
}

func (ms *msgServer) RevokeFeegrantAllowances(context context.Context, msg *types.MsgRevokeFeegrantAllowances) (*types.MsgRevokeFeegrantAllowancesResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	return &types.MsgRevokeFeegrantAllowancesResponse{}, nil
}
