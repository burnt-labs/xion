package keeper

import (
	"bytes"
	"context"
	"encoding/json"
	"net/url"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"

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
