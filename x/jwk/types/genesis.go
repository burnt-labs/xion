package types

import (
	"fmt"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// DefaultIndex is the default global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		AudienceList: []Audience{},
		// this line is used by starport scaffolding # genesis/types/default
		Params: DefaultParams(),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// Check for duplicated index in audience
	audienceIndexMap := make(map[string]struct{})

	for _, elem := range gs.AudienceList {
		_, err := sdk.AccAddressFromBech32(elem.Admin)
		if err != nil {
			return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid admin address (%s)", err)
		}

		index := string(AudienceKey(elem.Aud))
		if _, ok := audienceIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for audience")
		}
		audienceIndexMap[index] = struct{}{}

		// Validate JWK key format if key is present
		if elem.Key != "" {
			// Enforce size limit before parsing to avoid expensive operations on huge inputs.
			if len(elem.Key) > MaxJWKKeySize {
				return errorsmod.Wrapf(ErrInvalidJWK, "key size %d exceeds maximum %d bytes for audience %s", len(elem.Key), MaxJWKKeySize, elem.Aud)
			}

			parsedKey, err := jwk.ParseKey([]byte(elem.Key))
			if err != nil {
				return errorsmod.Wrapf(ErrInvalidJWK, "invalid JWK in genesis for audience %s: %s", elem.Aud, err)
			}

			// Reject symmetric (HMAC) and "none" signature algorithms — these are
			// disallowed for the same reasons as in MsgCreateAudience/MsgUpdateAudience.
			var sigAlg jwa.SignatureAlgorithm
			if err := sigAlg.Accept(parsedKey.Algorithm().String()); err != nil {
				return errorsmod.Wrapf(ErrInvalidJWK, "invalid algorithm in genesis JWK for audience %s: %s", elem.Aud, err)
			}

			switch sigAlg {
			case jwa.HS256, jwa.HS384, jwa.HS512, jwa.NoSignature:
				return errorsmod.Wrapf(ErrInvalidJWK, "invalid algorithm %s in genesis JWK for audience %s", sigAlg.String(), elem.Aud)
			}
		}
	}
	// this line is used by starport scaffolding # genesis/types/validate

	return gs.Params.Validate()
}
