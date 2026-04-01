package keeper

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/jwk/types"
)

func (k Keeper) DecodeJWT(goCtx context.Context, req *types.QueryDecodeJWTRequest) (*types.QueryDecodeJWTResponse, error) {
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

	// Validate key size to prevent DoS attacks from oversized keys
	// that might have been stored before validation was implemented
	if err := types.ValidateJWKKeySize(key); err != nil {
		return nil, status.Error(codes.FailedPrecondition, fmt.Sprintf("stored key validation failed: %s", err))
	}

	// basic sanity check
	if len(req.SigBytes) == 0 {
		return nil, status.Error(codes.InvalidArgument, "empty jwt")
	}

	// SECURITY: Do not remove or relax this check without a thorough security review.
	// Reject JWS JSON Serialization (both flattened and general forms).
	// Only compact JWTs (header.payload.signature) are accepted.
	//
	// See ValidateJWT for full rationale. Same defence-in-depth applies here.
	jwt.Settings(jwt.WithCompactOnly(true))

	trimmed := strings.TrimLeftFunc(req.SigBytes, unicode.IsSpace)
	if len(trimmed) == 0 {
		return nil, status.Error(codes.InvalidArgument, "empty jwt")
	}

	if trimmed[0] == '{' {
		return nil, status.Error(codes.InvalidArgument, "JWS JSON serialization is not supported; use compact JWT format")
	}

	// parse + validate with panic safety (defensive: lib should not panic, but guard anyway)
	var (
		token jwt.Token
	)
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = status.Error(codes.Internal, "panic during jwt parse")
			}
		}()

		clock := jwt.ClockFunc(func() time.Time {
			timeOffset := sdkmath.NewUint(k.GetTimeOffset(ctx)).BigInt().Int64()
			return ctx.BlockTime().Add(time.Duration(timeOffset))
		})

		token, err = jwt.Parse(
			[]byte(req.SigBytes),
			jwt.WithKey(key.Algorithm(), key),
			jwt.WithAudience(req.Aud),
			jwt.WithSubject(req.Sub),
			jwt.WithClock(clock),
			jwt.WithValidate(true),
		)
	}()

	if err != nil {
		return nil, err
	}

	// Collect all claims (standard + private) into a flat list
	claims := make([]*types.JWTClaim, 0)

	// Standard claims
	if v := token.Issuer(); v != "" {
		claims = append(claims, &types.JWTClaim{Key: "iss", Value: v})
	}
	if v := token.Subject(); v != "" {
		claims = append(claims, &types.JWTClaim{Key: "sub", Value: v})
	}
	if v := token.Audience(); len(v) > 0 {
		b, _ := json.Marshal(v)
		claims = append(claims, &types.JWTClaim{Key: "aud", Value: string(b)})
	}
	if v := token.Expiration(); !v.IsZero() {
		claims = append(claims, &types.JWTClaim{Key: "exp", Value: fmt.Sprintf("%d", v.Unix())})
	}
	if v := token.NotBefore(); !v.IsZero() {
		claims = append(claims, &types.JWTClaim{Key: "nbf", Value: fmt.Sprintf("%d", v.Unix())})
	}
	if v := token.IssuedAt(); !v.IsZero() {
		claims = append(claims, &types.JWTClaim{Key: "iat", Value: fmt.Sprintf("%d", v.Unix())})
	}
	if v := token.JwtID(); v != "" {
		claims = append(claims, &types.JWTClaim{Key: "jti", Value: v})
	}

	// Private claims
	for claimKey, v := range token.PrivateClaims() {
		var valStr string
		switch c := v.(type) {
		case string:
			valStr = c
		case fmt.Stringer:
			valStr = c.String()
		case []byte:
			valStr = string(c)
		case float64, float32, int, int32, int64, uint, uint32, uint64, bool:
			valStr = fmt.Sprintf("%v", c)
		default:
			if b, mErr := json.Marshal(v); mErr == nil {
				valStr = string(b)
			} else {
				valStr = fmt.Sprintf("%v", v)
			}
		}
		claims = append(claims, &types.JWTClaim{Key: claimKey, Value: valStr})
	}

	// Sort deterministically by key
	sort.SliceStable(claims, func(i, j int) bool {
		return claims[i].Key < claims[j].Key
	})

	return &types.QueryDecodeJWTResponse{
		Claims: claims,
	}, nil
}
