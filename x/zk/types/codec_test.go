package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/zk/types"
)

func TestRegisterLegacyAminoCodec(t *testing.T) {
	t.Run("registers MsgAddVKey", func(t *testing.T) {
		cdc := codec.NewLegacyAmino()

		require.NotPanics(t, func() {
			types.RegisterLegacyAminoCodec(cdc)
		})

		msg := &types.MsgAddVKey{
			Authority: "xion1abc123",
		}

		bz, err := cdc.MarshalJSON(msg)
		require.NoError(t, err)
		require.NotEmpty(t, bz)

		var decoded types.MsgAddVKey
		err = cdc.UnmarshalJSON(bz, &decoded)
		require.NoError(t, err)
		require.Equal(t, msg.Authority, decoded.Authority)
	})

	t.Run("registers MsgUpdateVKey", func(t *testing.T) {
		cdc := codec.NewLegacyAmino()
		types.RegisterLegacyAminoCodec(cdc)

		msg := &types.MsgUpdateVKey{
			Authority: "xion1abc123",
		}

		bz, err := cdc.MarshalJSON(msg)
		require.NoError(t, err)
		require.NotEmpty(t, bz)

		var decoded types.MsgUpdateVKey
		err = cdc.UnmarshalJSON(bz, &decoded)
		require.NoError(t, err)
		require.Equal(t, msg.Authority, decoded.Authority)
	})

	t.Run("registers MsgRemoveVKey", func(t *testing.T) {
		cdc := codec.NewLegacyAmino()
		types.RegisterLegacyAminoCodec(cdc)

		msg := &types.MsgRemoveVKey{
			Authority: "xion1abc123",
		}

		bz, err := cdc.MarshalJSON(msg)
		require.NoError(t, err)
		require.NotEmpty(t, bz)

		var decoded types.MsgRemoveVKey
		err = cdc.UnmarshalJSON(bz, &decoded)
		require.NoError(t, err)
		require.Equal(t, msg.Authority, decoded.Authority)
	})
}

func TestRegisterInterfaces(t *testing.T) {
	t.Run("registers MsgAddVKey as sdk.Msg", func(t *testing.T) {
		registry := codectypes.NewInterfaceRegistry()

		require.NotPanics(t, func() {
			types.RegisterInterfaces(registry)
		})

		msg := &types.MsgAddVKey{
			Authority: "xion1abc123",
		}

		any, err := codectypes.NewAnyWithValue(msg)
		require.NoError(t, err)
		require.NotNil(t, any)

		var sdkMsg sdk.Msg
		err = registry.UnpackAny(any, &sdkMsg)
		require.NoError(t, err)
		require.NotNil(t, sdkMsg)

		unpacked, ok := sdkMsg.(*types.MsgAddVKey)
		require.True(t, ok)
		require.Equal(t, msg.Authority, unpacked.Authority)
	})

	t.Run("registers MsgUpdateVKey as sdk.Msg", func(t *testing.T) {
		registry := codectypes.NewInterfaceRegistry()
		types.RegisterInterfaces(registry)

		msg := &types.MsgUpdateVKey{
			Authority: "xion1abc123",
		}

		any, err := codectypes.NewAnyWithValue(msg)
		require.NoError(t, err)
		require.NotNil(t, any)

		var sdkMsg sdk.Msg
		err = registry.UnpackAny(any, &sdkMsg)
		require.NoError(t, err)
		require.NotNil(t, sdkMsg)

		unpacked, ok := sdkMsg.(*types.MsgUpdateVKey)
		require.True(t, ok)
		require.Equal(t, msg.Authority, unpacked.Authority)
	})

	t.Run("registers MsgRemoveVKey as sdk.Msg", func(t *testing.T) {
		registry := codectypes.NewInterfaceRegistry()
		types.RegisterInterfaces(registry)

		msg := &types.MsgRemoveVKey{
			Authority: "xion1abc123",
		}

		any, err := codectypes.NewAnyWithValue(msg)
		require.NoError(t, err)
		require.NotNil(t, any)

		var sdkMsg sdk.Msg
		err = registry.UnpackAny(any, &sdkMsg)
		require.NoError(t, err)
		require.NotNil(t, sdkMsg)

		unpacked, ok := sdkMsg.(*types.MsgRemoveVKey)
		require.True(t, ok)
		require.Equal(t, msg.Authority, unpacked.Authority)
	})
}
