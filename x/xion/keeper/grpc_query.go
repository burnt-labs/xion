package keeper

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"

	errorsmod "cosmossdk.io/errors"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"

	sdktypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/xion/types"
)

var _ types.QueryServer = Keeper{}

func (k Keeper) WebAuthNVerifyRegister(ctx context.Context, request *types.QueryWebAuthNVerifyRegisterRequest) (*types.QueryWebAuthNVerifyRegisterResponse, error) {
	rp, err := url.Parse(request.Rp)
	if err != nil {
		return nil, err
	}
	credentials := bytes.NewReader(request.Data)
	if err := validateAttestation(credentials); err != nil {
		return nil, errorsmod.Wrapf(types.ErrNoValidWebAuth, err.Error())
	}

	data, err := protocol.ParseCredentialCreationResponseBody(credentials)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrNoValidWebAuth, err.Error())
	}

	sdkCtx := sdktypes.UnwrapSDKContext(ctx) // NOTE: verify this is the same for X nodes
	credential, err := types.VerifyRegistration(sdkCtx, rp, request.Addr, request.Challenge, data)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrNoValidWebAuth, err.Error())
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

	credentials := bytes.NewReader(request.Data)
	if err := validateAttestation(credentials); err != nil {
		return nil, errorsmod.Wrapf(types.ErrNoValidWebAuth, err.Error())
	}

	data, err := protocol.ParseCredentialRequestResponseBody(credentials)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrNoValidWebAuth, err.Error())
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

func validateAttestation(body io.Reader) error {
	var ccr protocol.CredentialCreationResponse

	if err := json.NewDecoder(body).Decode(&ccr); err != nil {
		return err
	}

	p := &protocol.ParsedAttestationResponse{}

	if err := json.Unmarshal(ccr.AttestationResponse.ClientDataJSON, &p.CollectedClientData); err != nil {
		return err
	}
	rawAuthData := p.AttestationObject.RawAuthData
	a := p.AttestationObject.AuthData

	minAuthDataLength := 37
	if minAuthDataLength > len(rawAuthData) {
		return errors.New(fmt.Sprintf("Expected data greater than %d bytes. Got %d bytes", minAuthDataLength, len(rawAuthData)))
	}

	a.RPIDHash = rawAuthData[:32]
	a.Flags = protocol.AuthenticatorFlags(rawAuthData[32])
	a.Counter = binary.BigEndian.Uint32(rawAuthData[33:37])

	remaining := len(rawAuthData) - minAuthDataLength
	if a.Flags.HasExtensions() {
		if remaining != 0 {
			if len(rawAuthData)-remaining > len(rawAuthData) {
				return errors.New(fmt.Sprint("Raw Auth Data seems to be malformed"))
			}
			a.ExtData = rawAuthData[len(rawAuthData)-remaining:]
			remaining -= len(a.ExtData)
		}
	}
	return nil
}
