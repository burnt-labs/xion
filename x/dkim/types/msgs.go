package types

import (
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
		if err := dkimPubKey.Validate(); err != nil {
			return err
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
