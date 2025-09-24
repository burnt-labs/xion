package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestModuleConstants(t *testing.T) {
	require.Equal(t, "globalfee", ModuleName)
	require.Equal(t, "globalfee", StoreKey)
	require.Equal(t, "globalfee", QuerierRoute)
}

func TestConstantsAreConsistent(t *testing.T) {
	// Ensure all constants are consistent with each other
	require.Equal(t, ModuleName, StoreKey)
	require.Equal(t, ModuleName, QuerierRoute)
}