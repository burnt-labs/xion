package keeper

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/jwk/types"
)

func (k Keeper) VerifyJWS(goCtx context.Context, req *types.QueryVerifyJWSRequest) (*types.QueryVerifyJWSResponse, error) {
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
	verifyGas := types.JWSVerifyBaseGas + types.JWSVerifyPerByteGas*uint64(len(audience.Key))
	ctx.GasMeter().ConsumeGas(verifyGas, "jwk/VerifyJWS: JWS verification cost")

	// basic sanity check
	if len(req.SigBytes) == 0 {
		return nil, status.Error(codes.InvalidArgument, "empty jws")
	}

	// SECURITY: Do not remove or relax this check without a thorough security review.
	// Reject JWS JSON Serialization (both flattened and general forms).
	// Only compact JWS (header.payload.signature) is accepted.
	//
	// See ValidateJWT for full rationale. Same defence-in-depth applies here:
	//
	// 1. Explicit leading-byte check (primary): reject anything that looks like
	//    JSON after trimming whitespace. Uses TrimLeftFunc with unicode.IsSpace
	//    for parity with bytes.TrimSpace used internally by the jwx library.
	//
	// 2. jws.WithCompact() option (backstop): passed to jws.Verify() to force
	//    compact-only deserialization, rejecting JSON serialization at the
	//    library level.
	trimmed := strings.TrimLeftFunc(req.SigBytes, unicode.IsSpace)
	if len(trimmed) == 0 {
		return nil, status.Error(codes.InvalidArgument, "empty jws")
	}

	if trimmed[0] == '{' {
		return nil, status.Error(codes.InvalidArgument, "JWS JSON serialization is not supported; use compact JWS format")
	}

	// verify with panic safety (defensive: lib should not panic, but guard anyway)
	var payload []byte
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = status.Error(codes.Internal, "panic during jws verify")
			}
		}()

		payload, err = jws.Verify([]byte(req.SigBytes), jws.WithKey(key.Algorithm(), key), jws.WithCompact())
	}()

	if err != nil {
		return nil, err
	}

	return &types.QueryVerifyJWSResponse{
		Payload: payload,
	}, nil
}
