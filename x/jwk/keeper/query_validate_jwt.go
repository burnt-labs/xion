package keeper

import (
	"context"
	"sort"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/jwk/types"
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

	token, err := jwt.Parse([]byte(req.SigBytes),
		jwt.WithKey(key.Algorithm(), key),
		jwt.WithAudience(req.Aud),
		jwt.WithSubject(req.Sub),
		jwt.WithClock(jwt.ClockFunc(func() time.Time {
			// adjust the time from the block-height due to lagging reported time
			timeOffset := sdkmath.NewUint(k.GetTimeOffset(ctx)).BigInt().Int64()
			return ctx.BlockTime().Add(time.Duration(timeOffset))
		})),
		jwt.WithValidate(true),
	)
	if err != nil {
		return nil, err
	}
	// returning maps in protobufs can get hairy, we return a list instead
	privateClaimsMap := token.PrivateClaims()
	privateClaims := make([]*types.PrivateClaim, len(privateClaimsMap))

	i := 0
	for k, v := range privateClaimsMap {
		privateClaims[i] = &types.PrivateClaim{
			Key:   k,
			Value: v.(string),
		}
		i++
	}
	// even though there should be no duplicates, sort this deterministically
	sort.SliceStable(privateClaims, func(i, j int) bool {
		return privateClaims[i].Key < privateClaims[j].Key
	})

	return &types.QueryValidateJWTResponse{
		PrivateClaims: privateClaims,
	}, nil
}
