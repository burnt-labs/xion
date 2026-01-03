package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/burnt-labs/xion/x/dkim/types"
)

func TestDefaultParams(t *testing.T) {
	params := types.DefaultParams()

	require.NotNil(t, params)
	require.Equal(t, uint64(1), params.VkeyIdentifier)
	require.Equal(t, types.DefaultMaxPubKeySizeBytes, params.MaxPubkeySizeBytes)
}

func TestParams_String(t *testing.T) {
	t.Run("default params to string", func(t *testing.T) {
		params := types.DefaultParams()
		str := params.String()

		require.NotEmpty(t, str)
		require.Contains(t, str, "vkey_identifier")
	})

	t.Run("empty params to string", func(t *testing.T) {
		params := types.Params{}
		str := params.String()

		require.NotEmpty(t, str)
		// Should be valid JSON even when empty
		require.Contains(t, str, "{")
		require.Contains(t, str, "}")
	})
}

func TestParams_Validate(t *testing.T) {
	t.Run("default params are valid", func(t *testing.T) {
		params := types.DefaultParams()
		err := params.Validate()
		require.NoError(t, err)
	})

	t.Run("empty params are valid", func(t *testing.T) {
		params := types.Params{MaxPubkeySizeBytes: types.DefaultMaxPubKeySizeBytes}
		err := params.Validate()
		require.NoError(t, err)
	})
}
