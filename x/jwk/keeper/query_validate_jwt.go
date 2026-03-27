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

// Deprecated: Use DecodeJWT instead, which returns all claims (standard and private).
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

	// Validate key size to prevent DoS attacks from oversized keys
	// that might have been stored before validation was implemented
	if err := types.ValidateJWKKeySize(key); err != nil {
		return nil, status.Error(codes.FailedPrecondition, fmt.Sprintf("stored key validation failed: %s", err))
	}

	// Charge gas proportional to key size to prevent free DoS via
	// Stargate-whitelisted or CosmWasm-callable query endpoints.
	verifyGas := types.JWTVerifyBaseGas + types.JWTVerifyPerByteGas*uint64(len(audience.Key))
	ctx.GasMeter().ConsumeGas(verifyGas, "jwk/ValidateJWT: JWT verification cost")

	// basic sanity check
	if len(req.SigBytes) == 0 {
		return nil, status.Error(codes.InvalidArgument, "empty jwt")
	}

	// SECURITY: Do not remove or relax this check without a thorough security review.
	// Reject JWS JSON Serialization (both flattened and general forms).
	// Only compact JWTs (header.payload.signature) are accepted.
	//
	// Background: jwt.Parse() from lestrrat-go/jwx accepts both compact and
	// JWS JSON serialization formats.
	//
	// Defense in depth — two layers:
	//
	// 1. Explicit leading-byte check (primary): reject anything that looks like
	//    JSON after trimming whitespace. jwt.Parse() calls bytes.TrimSpace()
	//    internally, which uses unicode.IsSpace (includes \t \n \v \f \r,
	//    space, U+0085 NEL, U+00A0 NBSP). We use TrimLeftFunc with the same
	//    predicate to ensure exact parity.
	//
	// 2. jwt.Settings(jwt.WithCompactOnly(true)) (backstop): the global setting
	//    is safe here because ValidateJWT is the only jwt.Parse() call site in
	//    this binary. It uses atomic operations so concurrent calls are fine.
	//    This guards against future code paths that might call jwt.Parse()
	//    without the byte check above.
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
	// returning maps in protobufs can get hairy, we return a list instead
	privateClaimsMap := token.PrivateClaims()
	privateClaims := make([]*types.PrivateClaim, len(privateClaimsMap))

	i := 0
	for k, v := range privateClaimsMap {
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
				// Fallback to fmt if JSON marshaling fails
				valStr = fmt.Sprintf("%v", v)
			}
		}
		privateClaims[i] = &types.PrivateClaim{
			Key:   k,
			Value: valStr,
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
