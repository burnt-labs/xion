package types

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/url"

	"cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkError "github.com/cosmos/cosmos-sdk/types/errors"
)

var (
	_ sdk.Msg = &MsgAddDkimPubKeys{}
	_ sdk.Msg = &MsgRemoveDkimPubKey{}
	_ sdk.Msg = &MsgRevokeDkimPubKey{}
	_ sdk.Msg = &MsgUpdateParams{}
)

// NewMsgAddDkimPubKey creates new instance of MsgAddDkimPubKey
func NewMsgAddDkimPubKeys(
	sender sdk.Address,
	dkimPubKeys []DkimPubKey,
) *MsgAddDkimPubKeys {
	return &MsgAddDkimPubKeys{
		Authority:   sender.String(),
		DkimPubkeys: dkimPubKeys,
	}
}

// Route returns the name of the module
func (msg MsgAddDkimPubKeys) Route() string { return ModuleName }

// Type returns the the action
func (msg MsgAddDkimPubKeys) Type() string { return "add_dkim_public_keys" }

// GetSigners returns the expected signers for a MsgAddDkimPubKey message.
func (msg *MsgAddDkimPubKeys) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}

// ValidateBasic does a sanity check on the provided data.
func (msg *MsgAddDkimPubKeys) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return errors.Wrap(err, "invalid authority address")
	}
	for _, dkimPubKey := range msg.DkimPubkeys {
		if err := ValidateDkimPubKey(dkimPubKey); err != nil {
			return errors.Wrapf(ErrInvalidPubKey, "error validating pubkeys: %v", err)
		}
	}
	return nil
}

// NewMsgRemoveDkimPubKey creates new instance of NewMsgRemoveDkimPubKey
func NewMsgRemoveDkimPubKey(
	sender sdk.Address,
	dkimPubKey DkimPubKey,
) *MsgRemoveDkimPubKey {
	return &MsgRemoveDkimPubKey{
		Authority: sender.String(),
		Selector:  dkimPubKey.Selector,
		Domain:    dkimPubKey.Domain,
	}
}

// Route returns the name of the module
func (msg MsgRemoveDkimPubKey) Route() string { return ModuleName }

// Type returns the the action
func (msg MsgRemoveDkimPubKey) Type() string { return "remove_dkim_public_keys" }

// GetSigners returns the expected signers for a MsgAddDkimPubKey message.
func (msg *MsgRemoveDkimPubKey) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}

// ValidateBasic does a sanity check on the provided data.
func (msg *MsgRemoveDkimPubKey) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return errors.Wrap(err, "invalid authority address")
	}
	if msg.Domain == "" {
		return errors.Wrap(sdkError.ErrInvalidRequest, "domain cannot be empty")
	}
	if msg.Selector == "" {
		return errors.Wrap(sdkError.ErrInvalidRequest, "selector cannot be empty")
	}
	return nil
}

// NewMsgRevokeDkimPubKey creates new instance of NewMsgRevokeDkimPubKey
// Private key is a pem encoded private key
func NewMsgRevokeDkimPubKey(
	sender sdk.Address,
	domain string,
	privKey []byte,
) *MsgRevokeDkimPubKey {
	return &MsgRevokeDkimPubKey{
		Signer:  sender.String(),
		Domain:  domain,
		PrivKey: privKey,
	}
}

// Route returns the name of the module
func (msg MsgRevokeDkimPubKey) Route() string { return ModuleName }

// Type returns the the action
func (msg MsgRevokeDkimPubKey) Type() string { return "remove_dkim_public_keys" }

// GetSigners returns the expected signers for a MsgAddDkimPubKey message.
func (msg *MsgRevokeDkimPubKey) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Signer)
	return []sdk.AccAddress{addr}
}

// ValidateBasic does a sanity check on the provided data.
func (msg *MsgRevokeDkimPubKey) ValidateBasic() error {
	// Reject empty domain to prevent full-table scans in the msg server's
	// NewPrefixedPairRange iteration.
	if msg.Domain == "" {
		return errors.Wrap(sdkError.ErrInvalidRequest, "domain cannot be empty")
	}
	// url pass the pubkey domain
	if _, err := url.Parse(msg.Domain); err != nil {
		return errors.Wrap(sdkError.ErrInvalidRequest, "dkim url key parsing failed "+err.Error())
	}
	d, _ := pem.Decode(msg.PrivKey)
	if d == nil {
		return errors.Wrap(ErrParsingPrivKey, "failed to decode pem private key")
	}
	if _, err := x509.ParsePKCS1PrivateKey(d.Bytes); err != nil {
		if key, err := x509.ParsePKCS8PrivateKey(d.Bytes); err != nil {
			return errors.Wrap(ErrParsingPrivKey, "failed to parse private key")
		} else {
			if _, ok := key.(*rsa.PrivateKey); !ok {
				return errors.Wrap(ErrParsingPrivKey, "key is not an RSA private key")
			}
			return nil
		}
	}
	return nil
}

// ValidateBasic does a sanity check on the provided data.
func (msg *MsgUpdateParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return errors.Wrap(err, "invalid authority address")
	}

	return msg.Params.Validate()
}

// GetSigners returns the expected signers for a MsgUpdateParams message.
func (msg *MsgUpdateParams) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}

// ValidateDkimPubKeys validates DKIM keys for genesis and state-loading paths.
// Unlike ValidateDkimPubKeysWithRevocation, it does NOT enforce a minimum RSA key
// size so that legacy keys (e.g. Yahoo's s1024 at 1024 bits) present in the default
// genesis are accepted. Use ValidateDkimPubKeysWithRevocation for message validation
// paths where new-key policy should be enforced.
func ValidateDkimPubKeys(dkimKeys []DkimPubKey, params Params) error {
	return ValidateDkimPubKeysWithRevocation(context.Background(), dkimKeys, params, nil, false)
}

// ValidateDkimPubKeysWithRevocation validates DKIM keys and optionally checks a revocation lookup.
// isRevoked should return true if the provided pubkey has been revoked.
// When enforceMinKeySize is true, additional size validation is applied
// (use for message validation). Genesis/state-loading paths should pass false
// to allow legacy keys such as Yahoo's s1024 selector.
func ValidateDkimPubKeysWithRevocation(
	ctx context.Context,
	dkimKeys []DkimPubKey,
	params Params,
	isRevoked func(context.Context, string) (bool, error),
	enforceMinKeySize bool,
) error {
	for _, dkimKey := range dkimKeys {
		if err := validateDkimPubKeyMetadata(dkimKey); err != nil {
			return err
		}

		pubKeyBytes, err := DecodePubKeyWithLimit(dkimKey.PubKey, params.MaxPubkeySizeBytes)
		if err != nil {
			return err
		}

		rsaPubKey, err := ParseRSAPublicKey(pubKeyBytes)
		if err != nil {
			return err
		}

		if isRevoked != nil {
			canonicalKey, err := CanonicalizeRSAPublicKey(rsaPubKey)
			if err != nil {
				return err
			}
			revoked, err := isRevoked(ctx, canonicalKey)
			if err != nil {
				return err
			}
			if revoked {
				return errors.Wrapf(ErrInvalidatedKey, "dkim public key for domain %s and selector %s has been revoked", dkimKey.Domain, dkimKey.Selector)
			}
		}
	}
	return nil
}

// ValidateDkimPubKey validates a DKIM public key entry for use in
// MsgAddDkimPubKeys.ValidateBasic. It uses ValidateBasicMaxPubKeySizeBytes as a
// generous ceiling since ValidateBasic is stateless. RSA key size enforcement
// happens in the msg server via ValidateDkimPubKeysWithRevocation.
func ValidateDkimPubKey(dkimKey DkimPubKey) error {
	if err := validateDkimPubKeyMetadata(dkimKey); err != nil {
		return err
	}

	pubKeyBytes, err := DecodePubKeyWithLimit(dkimKey.PubKey, ValidateBasicMaxPubKeySizeBytes)
	if err != nil {
		return err
	}

	_, err = ParseRSAPublicKey(pubKeyBytes)
	return err
}

func validateDkimPubKeyMetadata(dkimKey DkimPubKey) error {
	if dkimKey.KeyType != KeyType_KEY_TYPE_RSA_UNSPECIFIED {
		return ErrInvalidKeyType
	}

	if dkimKey.Version != Version_VERSION_DKIM1_UNSPECIFIED {
		return ErrInvalidVersion
	}

	return nil
}
