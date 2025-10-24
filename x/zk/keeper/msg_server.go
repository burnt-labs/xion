package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/zk/types"
)

type msgServer struct {
	k Keeper
}

var _ types.MsgServer = msgServer{}

// NewMsgServerImpl returns an implementation of the module MsgServer interface.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{k: keeper}
}

func (ms msgServer) AddVKey(goCtx context.Context, msg *types.MsgAddVKey) (*types.MsgAddVKeyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Validate basic message fields
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// Add the vkey (authority check and validation happens inside)
	id, err := ms.k.AddVKey(ctx, msg.Authority, msg.Name, msg.VkeyBytes, msg.Description)
	if err != nil {
		return nil, err
	}

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeAddVKey,
			sdk.NewAttribute(types.AttributeKeyVKeyID, fmt.Sprintf("%d", id)),
			sdk.NewAttribute(types.AttributeKeyVKeyName, msg.Name),
			sdk.NewAttribute(types.AttributeKeyAuthority, msg.Authority),
		),
	)

	return &types.MsgAddVKeyResponse{Id: id}, nil
}

func (ms msgServer) UpdateVKey(goCtx context.Context, msg *types.MsgUpdateVKey) (*types.MsgUpdateVKeyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	err := ms.k.UpdateVKey(ctx, msg.Authority, msg.Name, msg.VkeyBytes, msg.Description)
	if err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeUpdateVKey,
			sdk.NewAttribute(types.AttributeKeyVKeyName, msg.Name),
			sdk.NewAttribute(types.AttributeKeyAuthority, msg.Authority),
		),
	)

	return &types.MsgUpdateVKeyResponse{}, nil
}

// RemoveVKey handles the MsgRemoveVKey message
func (ms msgServer) RemoveVKey(goCtx context.Context, msg *types.MsgRemoveVKey) (*types.MsgRemoveVKeyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Remove the vkey (authority check happens inside)
	err := ms.k.RemoveVKey(ctx, msg.Authority, msg.Name)
	if err != nil {
		return nil, err
	}

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeRemoveVKey,
			sdk.NewAttribute(types.AttributeKeyVKeyName, msg.Name),
			sdk.NewAttribute(types.AttributeKeyAuthority, msg.Authority),
		),
	)

	return &types.MsgRemoveVKeyResponse{}, nil
}
