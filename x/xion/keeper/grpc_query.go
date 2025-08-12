package keeper

import (
	"bytes"
	"context"
	"encoding/json"
	"net/url"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"

	sdktypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/xion/types"
)

var _ types.QueryServer = Keeper{}

func (k Keeper) WebAuthNVerifyRegister(_ context.Context, request *types.QueryWebAuthNVerifyRegisterRequest) (*types.QueryWebAuthNVerifyRegisterResponse, error) {
	rp, err := url.Parse(request.Rp)
	if err != nil {
		return nil, err
	}

	data, err := protocol.ParseCredentialCreationResponseBody(bytes.NewReader(request.Data))
	if err != nil {
		return nil, err
	}

	credential, err := types.VerifyRegistration(rp, request.Addr, request.Challenge, data)
	if err != nil {
		return nil, err
	}

	credentialBz, err := json.Marshal(&credential)
	if err != nil {
		return nil, err
	}

	return &types.QueryWebAuthNVerifyRegisterResponse{Credential: credentialBz}, nil
}

func (k Keeper) WebAuthNVerifyAuthenticate(_ context.Context, request *types.QueryWebAuthNVerifyAuthenticateRequest) (*types.QueryWebAuthNVerifyAuthenticateResponse, error) {
	rp, err := url.Parse(request.Rp)
	if err != nil {
		return nil, err
	}

	data, err := protocol.ParseCredentialRequestResponseBody(bytes.NewReader(request.Data))
	if err != nil {
		return nil, err
	}

	var credential webauthn.Credential
	err = json.Unmarshal(request.Credential, &credential)
	if err != nil {
		return nil, err
	}

	_, err = types.VerifyAuthentication(rp, request.Addr, request.Challenge, &credential, data)
	if err != nil {
		return nil, err
	}

	return &types.QueryWebAuthNVerifyAuthenticateResponse{}, nil
}

// PlatformPercentage implements types.QueryServer.
func (k Keeper) PlatformPercentage(ctx context.Context, _ *types.QueryPlatformPercentageRequest) (*types.QueryPlatformPercentageResponse, error) {
	sdkCtx := sdktypes.UnwrapSDKContext(ctx)
	percentage := k.GetPlatformPercentage(sdkCtx).Uint64()
	return &types.QueryPlatformPercentageResponse{PlatformPercentage: percentage}, nil
}

// PlatformMinimum implements types.QueryServer.
func (k Keeper) PlatformMinimum(ctx context.Context, _ *types.QueryPlatformMinimumRequest) (*types.QueryPlatformMinimumResponse, error) {
	sdkCtx := sdktypes.UnwrapSDKContext(ctx)
	coins, err := k.GetPlatformMinimums(sdkCtx)
	if err != nil {
		return nil, err
	}

	return &types.QueryPlatformMinimumResponse{Minimums: coins}, nil
}
