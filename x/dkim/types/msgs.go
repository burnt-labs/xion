package types

import (
	"cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	_ sdk.Msg = &MsgUpdateParams{}
	_ sdk.Msg = &MsgAddDkimPubKey{}
)

// NewMsgUpdateParams creates new instance of MsgUpdateParams
func NewMsgUpdateParams(
	sender sdk.Address,
) *MsgUpdateParams {
	return &MsgUpdateParams{
		Authority: sender.String(),
		Params:    Params{},
	}
}

// Route returns the name of the module
func (msg MsgUpdateParams) Route() string { return ModuleName }

// Type returns the the action
func (msg MsgUpdateParams) Type() string { return "update_params" }

// GetSigners returns the expected signers for a MsgUpdateParams message.
func (msg *MsgUpdateParams) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}

// ValidateBasic does a sanity check on the provided data.
func (msg *MsgUpdateParams) Validate() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return errors.Wrap(err, "invalid authority address")
	}

	return msg.Params.Validate()
}

// NewMsgAddDkimPubKey creates new instance of MsgAddDkimPubKey
func NewMsgAddDkimPubKey(
	sender sdk.Address,
	dkimPubKeys []DkimPubKey,
) *MsgAddDkimPubKey {
	return &MsgAddDkimPubKey{
		Authority:   sender.String(),
		DkimPubkeys: dkimPubKeys,
	}
}

// Route returns the name of the module
func (msg MsgAddDkimPubKey) Route() string { return ModuleName }

// Type returns the the action
func (msg MsgAddDkimPubKey) Type() string { return "add_dkim_public_keys" }

// GetSigners returns the expected signers for a MsgAddDkimPubKey message.
func (msg *MsgAddDkimPubKey) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}

// ValidateBasic does a sanity check on the provided data.
func (msg *MsgAddDkimPubKey) Validate() error {
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
