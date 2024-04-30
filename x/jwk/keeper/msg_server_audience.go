package keeper

import (
	"context"
	"encoding/base64"
	"fmt"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/burnt-labs/xion/x/jwk/types"
)

func (k msgServer) CreateAudienceClaim(goCtx context.Context, msg *types.MsgCreateAudienceClaim) (*types.MsgCreateAudienceClaimResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// check if the audience is already claimed
	_, isFound := k.GetAudienceClaim(
		ctx,
		msg.AudHash,
	)
	if isFound {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "audience already claimed")
	}

	// extra gas consumed to dis-incentivize spamming
	ctx.GasMeter().ConsumeGas(k.GetDeploymentGas(ctx), fmt.Sprintf("gas for audience in jwt verifier %b", msg.AudHash))

	k.SetAudienceClaim(ctx, msg.AudHash, sdk.AccAddress(msg.Admin))

	return &types.MsgCreateAudienceClaimResponse{}, nil
}

func (k msgServer) DeleteAudienceClaim(goCtx context.Context, msg *types.MsgDeleteAudienceClaim) (*types.MsgDeleteAudienceClaimResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Check if the value exists
	valFound, isFound := k.GetAudienceClaim(
		ctx,
		msg.AudHash,
	)
	if !isFound {
		return nil, errorsmod.Wrap(sdkerrors.ErrKeyNotFound, "index not set")
	}

	// Checks if the msg admin is the same as the current owner
	if msg.Admin != valFound.Signer {
		return nil, errorsmod.Wrap(sdkerrors.ErrUnauthorized, "incorrect owner")
	}

	k.RemoveAudienceClaim(
		ctx,
		msg.AudHash,
	)

	return &types.MsgDeleteAudienceClaimResponse{}, nil
}

func (k msgServer) CreateAudience(goCtx context.Context, msg *types.MsgCreateAudience) (*types.MsgCreateAudienceResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Check if the value already exists
	_, isFound := k.GetAudience(
		ctx,
		msg.Aud,
	)
	if isFound {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "audience already created")
	}

	audHash, err := base64.URLEncoding.DecodeString(msg.Aud)
	if err != nil {
		return nil, errorsmod.Wrapf(err, "invalid base64: %s", msg.Aud)
	}

	claim, isFound := k.GetAudienceClaim(ctx, audHash)
	if !isFound {
		return nil, errorsmod.Wrapf(sdkerrors.ErrNotFound, "claim not found for aud %s", msg.Aud)
	}

	if claim.Signer != msg.Admin {
		return nil, errorsmod.Wrapf(sdkerrors.ErrorInvalidSigner, "expected %s, got %s", claim.Signer, msg.Admin)
	}

	audience := types.Audience{
		Admin: msg.Admin,
		Aud:   msg.Aud,
		Key:   msg.Key,
	}

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
		return nil, errorsmod.Wrap(sdkerrors.ErrKeyNotFound, "index not set")
	}

	// Checks if the msg signer is the same as the current owner
	if msg.Admin != valFound.Admin {
		return nil, errorsmod.Wrap(sdkerrors.ErrUnauthorized, "incorrect owner")
	}

	// updates based on new values provided, potentially admin, aud and key
	audience := types.Audience{
		Admin: msg.NewAdmin,
		Aud:   msg.Aud,
		Key:   msg.Key,
	}

	// if changing the aud, make sure a claim exists under this admin, and that it won't override
	if valFound.Aud != msg.Aud {
		// Check if the value already exists
		_, isFound := k.GetAudience(
			ctx,
			msg.Aud,
		)
		if isFound {
			return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "audience already created")
		}

		audHash, err := base64.URLEncoding.DecodeString(msg.Aud)
		if err != nil {
			return nil, errorsmod.Wrapf(err, "invalid base64: %s", msg.Aud)
		}

		claim, isFound := k.GetAudienceClaim(ctx, audHash)
		if !isFound {
			return nil, errorsmod.Wrapf(sdkerrors.ErrNotFound, "claim not found for aud %s", msg.Aud)
		}

		if claim.Signer != msg.Admin {
			return nil, errorsmod.Wrapf(sdkerrors.ErrorInvalidSigner, "expected %s, got %s", claim.Signer, msg.Admin)
		}

		k.RemoveAudience(ctx, valFound.Aud)
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
		return nil, errorsmod.Wrap(sdkerrors.ErrKeyNotFound, "index not set")
	}

	// Checks if the msg admin is the same as the current owner
	if msg.Admin != valFound.Admin {
		return nil, errorsmod.Wrap(sdkerrors.ErrUnauthorized, "incorrect owner")
	}

	k.RemoveAudience(
		ctx,
		msg.Aud,
	)

	return &types.MsgDeleteAudienceResponse{}, nil
}
