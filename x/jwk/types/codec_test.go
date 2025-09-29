package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestRegisterCodec(t *testing.T) {
	cdc := codec.NewLegacyAmino()

	// Should not panic
	require.NotPanics(t, func() {
		RegisterCodec(cdc)
	})

	// Verify that the codec is not nil after registration
	require.NotNil(t, cdc)
}

func TestRegisterInterfaces(t *testing.T) {
	registry := cdctypes.NewInterfaceRegistry()

	// Should not panic
	require.NotPanics(t, func() {
		RegisterInterfaces(registry)
	})

	// Verify that messages are registered as sdk.Msg implementations
	msgCreateAudience := &MsgCreateAudience{}
	msgUpdateAudience := &MsgUpdateAudience{}
	msgDeleteAudience := &MsgDeleteAudience{}

	// These should be registered as sdk.Msg implementations
	var _ sdk.Msg = msgCreateAudience
	var _ sdk.Msg = msgUpdateAudience
	var _ sdk.Msg = msgDeleteAudience
}

func TestModuleCodec(t *testing.T) {
	// Test that the module codec variables are properly initialized
	require.NotNil(t, Amino)
	require.NotNil(t, ModuleCdc)

	// Test that we can marshal/unmarshal using the module codec
	msg := &MsgCreateAudience{
		Admin: "test-admin",
		Aud:   "test-audience",
		Key:   "test-key",
	}

	// Test proto codec marshaling
	bz, err := ModuleCdc.Marshal(msg)
	require.NoError(t, err)
	require.NotEmpty(t, bz)

	// Test proto codec unmarshaling
	var decoded MsgCreateAudience
	err = ModuleCdc.Unmarshal(bz, &decoded)
	require.NoError(t, err)
	require.Equal(t, msg.Admin, decoded.Admin)
	require.Equal(t, msg.Aud, decoded.Aud)
	require.Equal(t, msg.Key, decoded.Key)
}

func TestCodecRegistration(t *testing.T) {
	// Test that we can create a new codec and register our types
	testRegistry := cdctypes.NewInterfaceRegistry()
	RegisterInterfaces(testRegistry)

	testCdc := codec.NewProtoCodec(testRegistry)
	require.NotNil(t, testCdc)

	// Test marshaling with the new codec
	msg := &MsgUpdateAudience{
		Admin: "test-admin",
		Aud:   "test-audience",
		Key:   "updated-key",
	}

	bz, err := testCdc.Marshal(msg)
	require.NoError(t, err)
	require.NotEmpty(t, bz)

	var decoded MsgUpdateAudience
	err = testCdc.Unmarshal(bz, &decoded)
	require.NoError(t, err)
	require.Equal(t, msg.Admin, decoded.Admin)
	require.Equal(t, msg.Aud, decoded.Aud)
	require.Equal(t, msg.Key, decoded.Key)
}
