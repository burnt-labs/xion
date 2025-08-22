package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/burnt-labs/xion/x/xion/types"
)

func TestErrors(t *testing.T) {
	// Test that error constants are properly defined
	require.NotNil(t, types.ErrNoAllowedContracts)
	require.NotNil(t, types.ErrNoValidAllowances)
	require.NotNil(t, types.ErrInconsistentExpiry)
	require.NotNil(t, types.ErrMinimumNotMet)
	require.NotNil(t, types.ErrNoValidWebAuth)

	// Test error messages
	require.Contains(t, types.ErrNoAllowedContracts.Error(), "no contract addresses specified")
	require.Contains(t, types.ErrNoValidAllowances.Error(), "none of the allowances accepted the msg")
	require.Contains(t, types.ErrInconsistentExpiry.Error(), "multi allowances must all expire together")
	require.Contains(t, types.ErrMinimumNotMet.Error(), "minimum send amount not met")
	require.Contains(t, types.ErrNoValidWebAuth.Error(), "Web auth is not valid")

	// Test codespace
	require.Equal(t, types.ModuleName, types.DefaultCodespace)
}
