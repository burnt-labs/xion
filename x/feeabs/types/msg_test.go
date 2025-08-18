package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestMsgSendQueryIbcDenomTWAP(t *testing.T) {
	// Test valid address
	validAddr := "cosmos1fl48vsnmsdzcv85q5d2q4z5ajdha8yu34mf0eh"
	senderAddr, err := sdk.AccAddressFromBech32(validAddr)
	require.NoError(t, err)

	// Test NewMsgSendQueryIbcDenomTWAP
	msg := NewMsgSendQueryIbcDenomTWAP(senderAddr)
	require.NotNil(t, msg)
	require.Equal(t, validAddr, msg.Sender)

	// Test Route
	require.Equal(t, sdk.MsgTypeURL(msg), msg.Route())

	// Test Type
	require.Equal(t, sdk.MsgTypeURL(msg), msg.Type())

	// Test GetSigners
	signers := msg.GetSigners()
	require.Len(t, signers, 1)
	require.Equal(t, senderAddr, signers[0])

	// Test GetSignBytes
	signBytes := msg.GetSignBytes()
	require.NotNil(t, signBytes)
	require.Greater(t, len(signBytes), 0)

	// Test ValidateBasic - valid case
	err = msg.ValidateBasic()
	require.NoError(t, err)

	// Test ValidateBasic - invalid sender
	invalidMsg := &MsgSendQueryIbcDenomTWAP{Sender: "invalid-address"}
	err = invalidMsg.ValidateBasic()
	require.Error(t, err)
}

func TestMsgSendQueryIbcDenomTWAP_GetSignersPanic(t *testing.T) {
	// Test panic case in GetSigners with invalid address
	msg := &MsgSendQueryIbcDenomTWAP{Sender: "invalid-address"}

	require.Panics(t, func() {
		msg.GetSigners()
	})
}

func TestMsgSwapCrossChain(t *testing.T) {
	// Test valid address
	validAddr := "cosmos1fl48vsnmsdzcv85q5d2q4z5ajdha8yu34mf0eh"
	senderAddr, err := sdk.AccAddressFromBech32(validAddr)
	require.NoError(t, err)

	ibcDenom := "ibc/27394FB092D2ECCD56123C74F36E4C1F926001CEADA9CA97EA622B25F41E5EB2"

	// Test NewMsgSwapCrossChain
	msg := NewMsgSwapCrossChain(senderAddr, ibcDenom)
	require.NotNil(t, msg)
	require.Equal(t, validAddr, msg.Sender)
	require.Equal(t, ibcDenom, msg.IbcDenom)

	// Test Route
	require.Equal(t, sdk.MsgTypeURL(msg), msg.Route())

	// Test Type
	require.Equal(t, sdk.MsgTypeURL(msg), msg.Type())

	// Test GetSigners
	signers := msg.GetSigners()
	require.Len(t, signers, 1)
	require.Equal(t, senderAddr, signers[0])

	// Test GetSignBytes
	signBytes := msg.GetSignBytes()
	require.NotNil(t, signBytes)
	require.Greater(t, len(signBytes), 0)

	// Test ValidateBasic - valid case
	err = msg.ValidateBasic()
	require.NoError(t, err)

	// Test ValidateBasic - invalid sender
	invalidMsg := &MsgSwapCrossChain{Sender: "invalid-address", IbcDenom: ibcDenom}
	err = invalidMsg.ValidateBasic()
	require.Error(t, err)
}

func TestMsgSwapCrossChain_GetSignersPanic(t *testing.T) {
	// Test panic case in GetSigners with invalid address
	msg := &MsgSwapCrossChain{Sender: "invalid-address", IbcDenom: "some-denom"}

	require.Panics(t, func() {
		msg.GetSigners()
	})
}

func TestMsgFundFeeAbsModuleAccount(t *testing.T) {
	// Test valid address
	validAddr := "cosmos1fl48vsnmsdzcv85q5d2q4z5ajdha8yu34mf0eh"
	senderAddr, err := sdk.AccAddressFromBech32(validAddr)
	require.NoError(t, err)

	amount := sdk.NewCoins(sdk.NewCoin("stake", math.NewInt(1000)))

	// Test NewMsgFundFeeAbsModuleAccount
	msg := NewMsgFundFeeAbsModuleAccount(senderAddr, amount)
	require.NotNil(t, msg)
	require.Equal(t, validAddr, msg.Sender)
	require.Equal(t, amount, msg.Amount)

	// Test Route
	require.Equal(t, sdk.MsgTypeURL(msg), msg.Route())

	// Test Type
	require.Equal(t, sdk.MsgTypeURL(msg), msg.Type())

	// Test GetSigners
	signers := msg.GetSigners()
	require.Len(t, signers, 1)
	require.Equal(t, senderAddr, signers[0])

	// Test GetSignBytes
	signBytes := msg.GetSignBytes()
	require.NotNil(t, signBytes)
	require.Greater(t, len(signBytes), 0)

	// Test ValidateBasic - valid case
	err = msg.ValidateBasic()
	require.NoError(t, err)

	// Test ValidateBasic - invalid sender
	invalidMsg := &MsgFundFeeAbsModuleAccount{Sender: "invalid-address", Amount: amount}
	err = invalidMsg.ValidateBasic()
	require.Error(t, err)
}

func TestMsgFundFeeAbsModuleAccount_GetSignersPanic(t *testing.T) {
	// Test panic case in GetSigners with invalid address
	amount := sdk.NewCoins(sdk.NewCoin("stake", math.NewInt(1000)))
	msg := &MsgFundFeeAbsModuleAccount{Sender: "invalid-address", Amount: amount}

	require.Panics(t, func() {
		msg.GetSigners()
	})
}

// Test that all messages implement the sdk.Msg interface
func TestMsgInterfaces(t *testing.T) {
	var _ sdk.Msg = &MsgSendQueryIbcDenomTWAP{}
	var _ sdk.Msg = &MsgSwapCrossChain{}
	var _ sdk.Msg = &MsgFundFeeAbsModuleAccount{}
}
