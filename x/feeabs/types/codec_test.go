package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestRegisterCodec(t *testing.T) {
	cdc := codec.NewLegacyAmino()

	// This should not panic
	require.NotPanics(t, func() {
		RegisterCodec(cdc)
	})
}

func TestRegisterInterfaces(t *testing.T) {
	registry := types.NewInterfaceRegistry()

	// This should not panic
	require.NotPanics(t, func() {
		RegisterInterfaces(registry)
	})

	// Just verify that the function runs without panic
	// The actual interface registration testing is complex and would require
	// a full setup with proper protobuf message registration
}

func TestAminoCodec(t *testing.T) {
	// Test that the amino codec is properly initialized
	require.NotNil(t, amino)

	// Test encoding/decoding a message

	// nolint: goconst
	validAddr := "cosmos1fl48vsnmsdzcv85q5d2q4z5ajdha8yu34mf0eh"
	senderAddr, err := sdk.AccAddressFromBech32(validAddr)
	require.NoError(t, err)

	msg := NewMsgSendQueryIbcDenomTWAP(senderAddr)

	// Should be able to marshal and unmarshal
	bz, err := amino.MarshalJSON(msg)
	require.NoError(t, err)
	require.NotEmpty(t, bz)

	var decoded MsgSendQueryIbcDenomTWAP
	err = amino.UnmarshalJSON(bz, &decoded)
	require.NoError(t, err)
	require.Equal(t, msg.Sender, decoded.Sender)
}

func TestInit(t *testing.T) {
	// The init function should have been called during package initialization
	// We can test that amino codec is properly set up
	require.NotNil(t, amino)

	// Test that we can encode a basic SDK message

	// nolint: goconst
	validAddr := "cosmos1fl48vsnmsdzcv85q5d2q4z5ajdha8yu34mf0eh"
	senderAddr, err := sdk.AccAddressFromBech32(validAddr)
	require.NoError(t, err)

	msg := NewMsgSendQueryIbcDenomTWAP(senderAddr)
	bz := msg.GetSignBytes()
	require.NotEmpty(t, bz)
}
