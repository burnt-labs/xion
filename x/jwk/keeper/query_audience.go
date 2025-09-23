package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"cosmossdk.io/store/prefix"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/burnt-labs/xion/x/jwk/types"
)

func (k Keeper) AudienceAll(goCtx context.Context, req *types.QueryAudienceAllRequest) (*types.QueryAudienceAllResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.Pagination != nil && req.Pagination.Limit > 100 {
		return nil, status.Error(codes.ResourceExhausted, "requests audience page size >100, too large")
	}

	var audiences []types.Audience
	ctx := sdk.UnwrapSDKContext(goCtx)

	store := ctx.KVStore(k.storeKey)
	audienceStore := prefix.NewStore(store, types.KeyPrefix(types.AudienceKeyPrefix))

	pageRes, err := query.Paginate(audienceStore, req.Pagination, func(_ []byte, value []byte) error {
		var audience types.Audience
		if err := k.cdc.Unmarshal(value, &audience); err != nil {
			return err
		}

		audiences = append(audiences, audience)
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAudienceAllResponse{Audience: audiences, Pagination: pageRes}, nil
}

func (k Keeper) Audience(goCtx context.Context, req *types.QueryAudienceRequest) (*types.QueryAudienceResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	val, found := k.GetAudience(
		ctx,
		req.Aud,
	)
	if !found {
		return nil, status.Error(codes.NotFound, "not found")
	}

	return &types.QueryAudienceResponse{Audience: val}, nil
}

func (k Keeper) AudienceClaim(goCtx context.Context, req *types.QueryAudienceClaimRequest) (*types.QueryAudienceClaimResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	val, found := k.GetAudienceClaim(
		ctx,
		req.Hash,
	)
	if !found {
		return nil, status.Error(codes.NotFound, "not found")
	}

	return &types.QueryAudienceClaimResponse{Claim: &val}, nil
}
