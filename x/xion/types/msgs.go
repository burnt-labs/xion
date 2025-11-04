package types

import (
	"errors"

	"github.com/btcsuite/btcd/btcutil/bech32"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// bank message types
const (
	TypeMsgSend                  = "send"
	TypeMsgMultiSend             = "multisend"
	TypeMsgSetPlatformPercentage = "setplatformpercentage"
	TypeMsgSetPlatformMinimum    = "setplatformminimum"
)

var (
	_ sdk.Msg = &MsgSend{}
	_ sdk.Msg = &MsgMultiSend{}
	_ sdk.Msg = &MsgSetPlatformPercentage{}
	_ sdk.Msg = &MsgSetPlatformMinimum{}
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
	// Use bech32.Decode which accepts any valid Bech32 prefix, not just "cosmos"
	if _, _, err := bech32.Decode(msg.FromAddress); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid from address: %s", err)
	}

	if _, _, err := bech32.Decode(msg.ToAddress); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid to address: %s", err)
	}

	if !msg.Amount.IsValid() {
		return errorsmod.Wrap(sdkerrors.ErrInvalidCoins, msg.Amount.String())
	}

	if !msg.Amount.IsAllPositive() {
		return errorsmod.Wrap(sdkerrors.ErrInvalidCoins, msg.Amount.String())
	}

	return nil
}

// GetSignBytes Implements Msg.
func (msg MsgSend) GetSignBytes() []byte {
	return sdk.MustSortJSON(amino.MustMarshalJSON(&msg))
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

	return banktypes.ValidateInputOutputs(msg.Inputs[0], msg.Outputs)
}

// GetSignBytes Implements Msg.
func (msg MsgMultiSend) GetSignBytes() []byte {
	return sdk.MustSortJSON(amino.MustMarshalJSON(&msg))
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
	return sdk.MustSortJSON(amino.MustMarshalJSON(&msg))
}

// GetSigners Implements Msg.
func (msg MsgSetPlatformPercentage) GetSigners() []sdk.AccAddress {
	// Extract the address bytes while preserving the original Bech32 prefix
	_, addrBytes, err := bech32.Decode(msg.Authority)
	if err != nil {
		// Fallback to standard conversion if decode fails
		addr, _ := sdk.AccAddressFromBech32(msg.Authority)
		return []sdk.AccAddress{addr}
	}
	// Convert from base32 to bytes
	converted, err := bech32.ConvertBits(addrBytes, 5, 8, false)
	if err != nil {
		// Fallback to standard conversion if conversion fails
		addr, _ := sdk.AccAddressFromBech32(msg.Authority)
		return []sdk.AccAddress{addr}
	}
	return []sdk.AccAddress{converted}
}

// NewMsgSetPlatformMinimum constructs a message to set platform minimums.
func NewMsgSetPlatformMinimum(authority sdk.AccAddress, minimums sdk.Coins) *MsgSetPlatformMinimum {
	return &MsgSetPlatformMinimum{Authority: authority.String(), Minimums: minimums}
}

// Route Implements Msg
func (msg MsgSetPlatformMinimum) Route() string { return RouterKey }

// Type Implements Msg
func (msg MsgSetPlatformMinimum) Type() string { return TypeMsgSetPlatformMinimum }

// ValidateBasic Implements Msg.
func (msg MsgSetPlatformMinimum) ValidateBasic() error {
	// Use bech32.Decode which accepts any valid Bech32 prefix, not just "cosmos"
	if _, _, err := bech32.Decode(msg.Authority); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid authority address: %s", err)
	}

	if !msg.Minimums.IsValid() {
		return errorsmod.Wrap(sdkerrors.ErrInvalidCoins, msg.Minimums.String())
	}
	// Minimums can be zero but never negative
	if msg.Minimums.IsAnyNegative() {
		return errorsmod.Wrap(sdkerrors.ErrInvalidCoins, msg.Minimums.String())
	}
	return nil
}

// GetSignBytes Implements Msg.
func (msg MsgSetPlatformMinimum) GetSignBytes() []byte {
	return sdk.MustSortJSON(amino.MustMarshalJSON(&msg))
}

// GetSigners Implements Msg.
func (msg MsgSetPlatformMinimum) GetSigners() []sdk.AccAddress {
	// Extract the address bytes while preserving the original Bech32 prefix
	_, addrBytes, err := bech32.Decode(msg.Authority)
	if err != nil {
		// Fallback to standard conversion if decode fails
		addr, _ := sdk.AccAddressFromBech32(msg.Authority)
		return []sdk.AccAddress{addr}
	}
	// Convert from base32 to bytes
	converted, err := bech32.ConvertBits(addrBytes, 5, 8, false)
	if err != nil {
		// Fallback to standard conversion if conversion fails
		addr, _ := sdk.AccAddressFromBech32(msg.Authority)
		return []sdk.AccAddress{addr}
	}
	return []sdk.AccAddress{converted}
}
