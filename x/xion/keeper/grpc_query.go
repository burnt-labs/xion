package keeper

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/url"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/protocol/webauthncbor"
	"github.com/go-webauthn/webauthn/webauthn"

	errorsmod "cosmossdk.io/errors"

	sdktypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/xion/types"
)

var _ types.QueryServer = Keeper{}

func (k Keeper) WebAuthNVerifyRegister(ctx context.Context, request *types.QueryWebAuthNVerifyRegisterRequest) (*types.QueryWebAuthNVerifyRegisterResponse, error) {
	rp, err := url.Parse(request.Rp)
	if err != nil {
		return nil, err
	}

	if err := validateCredentialCreation(bytes.NewReader(request.Data)); err != nil {
		return nil, errorsmod.Wrap(types.ErrNoValidWebAuth, err.Error())
	}

	data, err := protocol.ParseCredentialCreationResponseBody(bytes.NewReader(request.Data))
	if err != nil {
		return nil, errorsmod.Wrap(types.ErrNoValidWebAuth, err.Error())
	}

	sdkCtx := sdktypes.UnwrapSDKContext(ctx) // NOTE: verify this is the same for X nodes
	credential, err := types.VerifyRegistration(sdkCtx, rp, request.Addr, request.Challenge, data)
	if err != nil {
		return nil, errorsmod.Wrap(types.ErrNoValidWebAuth, err.Error())
	}

	credentialBz, err := json.Marshal(&credential)
	if err != nil {
		return nil, err
	}

	return &types.QueryWebAuthNVerifyRegisterResponse{Credential: credentialBz}, nil
}

func (k Keeper) WebAuthNVerifyAuthenticate(ctx context.Context, request *types.QueryWebAuthNVerifyAuthenticateRequest) (response *types.QueryWebAuthNVerifyAuthenticateResponse, err error) {
	sdkCtx := sdktypes.UnwrapSDKContext(ctx)
	// Recover from panics to prevent DoS attacks with malformed WebAuthn data
	defer func() {
		if r := recover(); r != nil {
			response = nil
			err = errorsmod.Wrap(types.ErrNoValidWebAuth, fmt.Sprintf("panic during WebAuthn verification: %v", r))
		}
	}()

	rp, err := url.Parse(request.Rp)
	if err != nil {
		return nil, err
	}

	if err := validateCredentialRequest(bytes.NewReader(request.Data)); err != nil {
		return nil, errorsmod.Wrap(types.ErrNoValidWebAuth, err.Error())
	}

	data, err := protocol.ParseCredentialRequestResponseBody(bytes.NewReader(request.Data))
	if err != nil {
		return nil, errorsmod.Wrap(types.ErrNoValidWebAuth, err.Error())
	}

	var credential webauthn.Credential
	err = json.Unmarshal(request.Credential, &credential)
	if err != nil {
		return nil, err
	}

	_, err = types.VerifyAuthentication(sdkCtx, rp, request.Addr, request.Challenge, &credential, data)
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

func validateCredentialCreation(body io.Reader) error {
	var ccr protocol.CredentialCreationResponse

	if err := json.NewDecoder(body).Decode(&ccr); err != nil {
		return err
	}

	p := &protocol.ParsedAttestationResponse{}

	if err := json.Unmarshal(ccr.AttestationResponse.ClientDataJSON, &p.CollectedClientData); err != nil {
		return err
	}

	if err := webauthncbor.Unmarshal(ccr.AttestationResponse.AttestationObject, &p.AttestationObject); err != nil {
		return err
	}
	return validateAttestation(p.AttestationObject.RawAuthData)
}

func validateCredentialRequest(body io.Reader) error {
	var car protocol.CredentialAssertionResponse
	if err := json.NewDecoder(body).Decode(&car); err != nil {
		return err
	}

	return validateAttestation(car.AssertionResponse.AuthenticatorData)
}

func validateAttestation(rawAuthData []byte) error {
	var a protocol.AuthenticatorData

	minAuthDataLength := 37
	if minAuthDataLength > len(rawAuthData) {
		return fmt.Errorf("expected data greater than %d bytes. Got %d bytes", minAuthDataLength, len(rawAuthData))
	}

	a.RPIDHash = rawAuthData[:32]
	a.Flags = protocol.AuthenticatorFlags(rawAuthData[32])
	a.Counter = binary.BigEndian.Uint32(rawAuthData[33:37])

	remaining := len(rawAuthData) - minAuthDataLength

	if a.Flags.HasExtensions() {
		if remaining != 0 && len(rawAuthData)-remaining > len(rawAuthData) {
			return fmt.Errorf("raw auth data seems to be malformed")
		}
	}
	return nil
}
