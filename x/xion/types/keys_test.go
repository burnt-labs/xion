package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/burnt-labs/xion/x/xion/types"
)

func TestKeys(t *testing.T) {
	// Test PlatformPercentageKey
	require.NotNil(t, types.PlatformPercentageKey)
	require.Equal(t, []byte{0x00}, types.PlatformPercentageKey)

	// Test PlatformMinimumKey
	require.NotNil(t, types.PlatformMinimumKey)
	require.Equal(t, []byte{0x01}, types.PlatformMinimumKey)
}

func TestConstants(t *testing.T) {
	// Test ModuleName
	require.Equal(t, "xion", types.ModuleName)

	// Test StoreKey
	require.Equal(t, types.ModuleName, types.StoreKey)
	require.Equal(t, "xion", types.StoreKey)

	// Test RouterKey
	require.Equal(t, types.ModuleName, types.RouterKey)
	require.Equal(t, "xion", types.RouterKey)

	// Test QuerierRoute
	require.Equal(t, types.ModuleName, types.QuerierRoute)
	require.Equal(t, "xion", types.QuerierRoute)

	// Test all constants are consistent
	require.Equal(t, types.ModuleName, types.StoreKey)
	require.Equal(t, types.ModuleName, types.RouterKey)
	require.Equal(t, types.ModuleName, types.QuerierRoute)
}

func TestKeyUniqueness(t *testing.T) {
	// Ensure keys are unique
	require.NotEqual(t, types.PlatformPercentageKey, types.PlatformMinimumKey)
}
