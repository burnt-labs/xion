package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/dkim/types"
)

func TestRegisterLegacyAminoCodec(t *testing.T) {
	t.Run("registers MsgAddDkimPubKeys", func(t *testing.T) {
		cdc := codec.NewLegacyAmino()

		// Should not panic
		require.NotPanics(t, func() {
			types.RegisterLegacyAminoCodec(cdc)
		})

		// Verify the message can be marshaled/unmarshaled after registration
		msg := &types.MsgAddDkimPubKeys{
			Authority: "xion1abc123",
			DkimPubkeys: []types.DkimPubKey{
				{
					Domain:   "example.com",
					Selector: "selector1",
					PubKey:   "test-pubkey",
				},
			},
		}

		// Marshal should work after registration
		bz, err := cdc.MarshalJSON(msg)
		require.NoError(t, err)
		require.NotEmpty(t, bz)

		// Unmarshal should work
		var decoded types.MsgAddDkimPubKeys
		err = cdc.UnmarshalJSON(bz, &decoded)
		require.NoError(t, err)
		require.Equal(t, msg.Authority, decoded.Authority)
		require.Len(t, decoded.DkimPubkeys, 1)
		require.Equal(t, msg.DkimPubkeys[0].Domain, decoded.DkimPubkeys[0].Domain)
	})
}

func TestRegisterInterfaces(t *testing.T) {
	t.Run("registers sdk.Msg implementation", func(t *testing.T) {
		registry := codectypes.NewInterfaceRegistry()

		// Should not panic
		require.NotPanics(t, func() {
			types.RegisterInterfaces(registry)
		})

		// Verify MsgAddDkimPubKeys is registered as sdk.Msg
		msg := &types.MsgAddDkimPubKeys{
			Authority: "xion1abc123",
			DkimPubkeys: []types.DkimPubKey{
				{
					Domain:   "example.com",
					Selector: "selector1",
					PubKey:   "test-pubkey",
				},
			},
		}

		// Should be able to pack the message as Any
		any, err := codectypes.NewAnyWithValue(msg)
		require.NoError(t, err)
		require.NotNil(t, any)

		// Should be able to unpack
		var sdkMsg sdk.Msg
		err = registry.UnpackAny(any, &sdkMsg)
		require.NoError(t, err)
		require.NotNil(t, sdkMsg)

		// Verify the unpacked message is correct type
		unpacked, ok := sdkMsg.(*types.MsgAddDkimPubKeys)
		require.True(t, ok)
		require.Equal(t, msg.Authority, unpacked.Authority)
	})

	t.Run("multiple registrations do not panic", func(t *testing.T) {
		registry := codectypes.NewInterfaceRegistry()

		require.NotPanics(t, func() {
			types.RegisterInterfaces(registry)
			types.RegisterInterfaces(registry)
		})
	})
}
