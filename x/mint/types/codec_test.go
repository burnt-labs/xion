package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestRegisterLegacyAminoCodec(t *testing.T) {
	cdc := codec.NewLegacyAmino()

	// Test that RegisterLegacyAminoCodec doesn't panic
	require.NotPanics(t, func() {
		RegisterLegacyAminoCodec(cdc)
	})

	// Test that Params can be marshaled/unmarshaled
	params := DefaultParams()

	// Test marshaling
	bz, err := cdc.MarshalJSON(params)
	require.NoError(t, err)
	require.NotEmpty(t, bz)

	// Test unmarshaling
	var unmarshaled Params
	err = cdc.UnmarshalJSON(bz, &unmarshaled)
	require.NoError(t, err)
	require.Equal(t, params, unmarshaled)
}

func TestRegisterInterfaces(t *testing.T) {
	registry := types.NewInterfaceRegistry()

	// Test that RegisterInterfaces doesn't panic
	require.NotPanics(t, func() {
		RegisterInterfaces(registry)
	})

	// Test that MsgUpdateParams is registered as sdk.Msg
	msg := &MsgUpdateParams{}
	require.Implements(t, (*sdk.Msg)(nil), msg)

	// Test that we can pack/unpack the message
	any, err := types.NewAnyWithValue(msg)
	require.NoError(t, err)

	var unpacked sdk.Msg
	err = registry.UnpackAny(any, &unpacked)
	require.NoError(t, err)
	require.Equal(t, msg, unpacked)
}
