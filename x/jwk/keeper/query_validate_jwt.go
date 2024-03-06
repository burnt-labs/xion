package keeper

import (
	"context"
	"time"

	"github.com/burnt-labs/xion/x/jwk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) ValidateJWT(goCtx context.Context, req *types.QueryValidateJWTRequest) (*types.QueryValidateJWTResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	audience, exists := k.GetAudience(ctx, req.Aud)
	if !exists {
		return nil, status.Error(codes.NotFound, "not found")
	}

	key, err := jwk.ParseKey([]byte(audience.Key))
	if err != nil {
		return nil, err
	}

	if _, err := jwt.Parse([]byte(req.SigBytes),
		jwt.WithKey(key.Algorithm(), key),
		jwt.WithAudience(req.Aud),
		jwt.WithSubject(req.Sub),
		jwt.WithClock(jwt.ClockFunc(func() time.Time {
			// adjust the time from the block-height due to lagging reported time
			return ctx.BlockTime().Add(time.Duration(k.GetParams(ctx).TimeOffset))
		})),
		jwt.WithValidate(true),
	); err != nil {
		return nil, err
	}

	return &types.QueryValidateJWTResponse{}, nil
}
