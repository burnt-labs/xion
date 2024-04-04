package types

import (
	"errors"

	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authztypes "github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// bank message types
const (
	TypeMsgSend                  = "send"
	TypeMsgMultiSend             = "multisend"
	TypeMsgSetPlatformPercentage = "setplatformpercentage"
)

var (
	_ sdk.Msg = &MsgSend{}
	_ sdk.Msg = &MsgMultiSend{}
	_ sdk.Msg = &MsgSetPlatformPercentage{}
	_ sdk.Msg = &MsgExec{}
)

// NewMsgSend - construct a msg to send coins from one account to another.
//
//nolint:interfacer
func NewMsgSend(fromAddr, toAddr sdk.AccAddress, amount sdk.Coins) *MsgSend {
	return &MsgSend{FromAddress: fromAddr.String(), ToAddress: toAddr.String(), Amount: amount}
}

// Route Implements Msg.
func (msg MsgSend) Route() string { return RouterKey }

// Type Implements Msg.
func (msg MsgSend) Type() string { return TypeMsgSend }

// ValidateBasic Implements Msg.
func (msg MsgSend) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.FromAddress); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid from address: %s", err)
	}

	if _, err := sdk.AccAddressFromBech32(msg.ToAddress); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid to address: %s", err)
	}

	if !msg.Amount.IsValid() {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidCoins, msg.Amount.String())
	}

	if !msg.Amount.IsAllPositive() {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidCoins, msg.Amount.String())
	}

	return nil
}

// GetSignBytes Implements Msg.
func (msg MsgSend) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}

// GetSigners Implements Msg.
func (msg MsgSend) GetSigners() []sdk.AccAddress {
	fromAddress, _ := sdk.AccAddressFromBech32(msg.FromAddress)
	return []sdk.AccAddress{fromAddress}
}

// NewMsgMultiSend - construct arbitrary multi-in, multi-out send msg.
func NewMsgMultiSend(in []banktypes.Input, out []banktypes.Output) *MsgMultiSend {
	return &MsgMultiSend{Inputs: in, Outputs: out}
}

// Route Implements Msg
func (msg MsgMultiSend) Route() string { return RouterKey }

// Type Implements Msg
func (msg MsgMultiSend) Type() string { return TypeMsgMultiSend }

// ValidateBasic Implements Msg.
func (msg MsgMultiSend) ValidateBasic() error {
	// this just makes sure the input and all the outputs are properly formatted,
	// not that they actually have the money inside

	if len(msg.Inputs) == 0 {
		return banktypes.ErrNoInputs
	}

	if len(msg.Inputs) != 1 {
		return banktypes.ErrMultipleSenders
	}

	if len(msg.Outputs) == 0 {
		return banktypes.ErrNoOutputs
	}

	return banktypes.ValidateInputsOutputs(msg.Inputs, msg.Outputs)
}

// GetSignBytes Implements Msg.
func (msg MsgMultiSend) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}

// GetSigners Implements Msg.
func (msg MsgMultiSend) GetSigners() []sdk.AccAddress {
	addrs := make([]sdk.AccAddress, len(msg.Inputs))
	for i, in := range msg.Inputs {
		inAddr, _ := sdk.AccAddressFromBech32(in.Address)
		addrs[i] = inAddr
	}

	return addrs
}

// NewMsgMultiSend - construct arbitrary multi-in, multi-out send msg.
func NewMsgSetPlatformPercentage(percentage uint32) *MsgSetPlatformPercentage {
	return &MsgSetPlatformPercentage{PlatformPercentage: percentage}
}

// Route Implements Msg
func (msg MsgSetPlatformPercentage) Route() string { return RouterKey }

// Type Implements Msg
func (msg MsgSetPlatformPercentage) Type() string { return TypeMsgSetPlatformPercentage }

// ValidateBasic Implements Msg.
func (msg MsgSetPlatformPercentage) ValidateBasic() error {
	// this just makes sure the input and all the outputs are properly formatted,
	// not that they actually have the money inside

	if msg.PlatformPercentage > 10000 {
		return errors.New("unable to have a platform percentage that exceeds 100%")
	}

	return nil
}

// GetSignBytes Implements Msg.
func (msg MsgSetPlatformPercentage) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&msg))
}

// GetSigners Implements Msg.
func (msg MsgSetPlatformPercentage) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}

// NewMsgExec creates a new MsgExecAuthorized
//
//nolint:interfacer
func NewMsgExec(grantee sdk.AccAddress, msgs []sdk.Msg) MsgExec {
	msgsAny := make([]*cdctypes.Any, len(msgs))
	for i, msg := range msgs {
		any, err := cdctypes.NewAnyWithValue(msg)
		if err != nil {
			panic(err)
		}

		msgsAny[i] = any
	}

	return MsgExec{
		Grantee: grantee.String(),
		Msgs:    msgsAny,
	}
}

// GetMessages returns the cache values from the MsgExecAuthorized.Msgs if present.
func (msg MsgExec) GetMessages() ([]sdk.Msg, error) {

	grantMsg := authztypes.MsgExec{
		Grantee: msg.Grantee,
		Msgs:    msg.Msgs,
	}
	return grantMsg.GetMessages()
}

// GetSigners implements Msg
func (msg MsgExec) GetSigners() []sdk.AccAddress {
	grantee, _ := sdk.AccAddressFromBech32(msg.Grantee)
	return []sdk.AccAddress{grantee}
}

// ValidateBasic implements Msg
func (msg MsgExec) ValidateBasic() error {
	grantMsg := authztypes.MsgExec{
		Grantee: msg.Grantee,
		Msgs:    msg.Msgs,
	}
	return grantMsg.ValidateBasic()
}

// Type implements the LegacyMsg.Type method.
func (msg MsgExec) Type() string {
	return sdk.MsgTypeURL(&msg)
}

// Route implements the LegacyMsg.Route method.
func (msg MsgExec) Route() string {
	return sdk.MsgTypeURL(&msg)
}

// GetSignBytes implements the LegacyMsg.GetSignBytes method.
func (msg MsgExec) GetSignBytes() []byte {
	grantMsg := authztypes.MsgExec{
		Grantee: msg.Grantee,
		Msgs:    msg.Msgs,
	}
	return grantMsg.GetSignBytes()
}
