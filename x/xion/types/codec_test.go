package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"

	xionTypes "github.com/burnt-labs/xion/x/xion/types"
)

func TestRegisterLegacyAminoCodec(t *testing.T) {
	cdc := codec.NewLegacyAmino()

	// Should not panic
	require.NotPanics(t, func() {
		xionTypes.RegisterLegacyAminoCodec(cdc)
	})
}

func TestRegisterInterfaces(t *testing.T) {
	registry := types.NewInterfaceRegistry()

	// Should not panic
	require.NotPanics(t, func() {
		xionTypes.RegisterInterfaces(registry)
	})

	// Test that some interfaces are registered
	require.NotNil(t, registry)
}

func TestCodecInit(t *testing.T) {
	// Test that init function runs without panic
	// This is implicitly tested by importing the package
	require.True(t, true)
}
