package keeper

import (
	"context"
	"fmt"

	"github.com/burnt-labs/xion/x/jwk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (k msgServer) CreateAudience(goCtx context.Context, msg *types.MsgCreateAudience) (*types.MsgCreateAudienceResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Check if the value already exists
	_, isFound := k.GetAudience(
		ctx,
		msg.Aud,
	)
	if isFound {
		return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "index already set")
	}

	var audience = types.Audience{
		Admin: msg.Admin,
		Aud:   msg.Aud,
		Key:   msg.Key,
	}

	// extra gas consumed to dis-incentivize spamming
	ctx.GasMeter().ConsumeGas(k.GetParams(ctx).DeploymentGas, fmt.Sprintf("gas for jwt verifier %s", msg.Aud))

	k.SetAudience(
		ctx,
		audience,
	)
	return &types.MsgCreateAudienceResponse{}, nil
}

func (k msgServer) UpdateAudience(goCtx context.Context, msg *types.MsgUpdateAudience) (*types.MsgUpdateAudienceResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Check if the value exists
	valFound, isFound := k.GetAudience(
		ctx,
		msg.Aud,
	)
	if !isFound {
		return nil, sdkerrors.Wrap(sdkerrors.ErrKeyNotFound, "index not set")
	}

	// Checks if the msg signer is the same as the current owner
	if msg.Signer != valFound.Admin {
		return nil, sdkerrors.Wrap(sdkerrors.ErrUnauthorized, "incorrect owner")
	}

	var audience = types.Audience{
		Admin: msg.Admin,
		Aud:   msg.Aud,
		Key:   msg.Key,
	}

	k.SetAudience(ctx, audience)

	return &types.MsgUpdateAudienceResponse{}, nil
}

func (k msgServer) DeleteAudience(goCtx context.Context, msg *types.MsgDeleteAudience) (*types.MsgDeleteAudienceResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Check if the value exists
	valFound, isFound := k.GetAudience(
		ctx,
		msg.Aud,
	)
	if !isFound {
		return nil, sdkerrors.Wrap(sdkerrors.ErrKeyNotFound, "index not set")
	}

	// Checks if the msg admin is the same as the current owner
	if msg.Admin != valFound.Admin {
		return nil, sdkerrors.Wrap(sdkerrors.ErrUnauthorized, "incorrect owner")
	}

	k.RemoveAudience(
		ctx,
		msg.Aud,
	)

	return &types.MsgDeleteAudienceResponse{}, nil
}
