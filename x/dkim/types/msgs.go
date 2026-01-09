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
			_ = key.(*rsa.PrivateKey)
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

func ValidateDkimPubKeys(dkimKeys []DkimPubKey, params Params) error {
	return ValidateDkimPubKeysWithRevocation(context.Background(), dkimKeys, params, nil)
}

// ValidateDkimPubKeysWithRevocation validates DKIM keys and optionally checks a revocation lookup.
// isRevoked should return true if the provided pubkey has been revoked.
func ValidateDkimPubKeysWithRevocation(
	ctx context.Context,
	dkimKeys []DkimPubKey,
	params Params,
	isRevoked func(context.Context, string) (bool, error),
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

// ValidateDkimPubKey validates a DKIM public key entry
func ValidateDkimPubKey(dkimKey DkimPubKey) error {
	if err := validateDkimPubKeyMetadata(dkimKey); err != nil {
		return err
	}

	// Validate PubKey is valid base64-encoded RSA public key
	pubKeyBytes, err := DecodePubKey(dkimKey.PubKey)
	if err != nil {
		return err
	}

	_, err = ParseRSAPublicKey(pubKeyBytes)
	return err
}

// ValidateRSAPubKey validates that the string is a valid base64-encoded RSA public key
func ValidateRSAPubKey(pubKeyStr string) error {
	pubKeyBytes, err := DecodePubKey(pubKeyStr)
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
