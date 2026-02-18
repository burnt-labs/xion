package types_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/burnt-labs/xion/x/zk/types"
)

func TestParamsValidate(t *testing.T) {
	defaultParams := types.DefaultParams()
	require.NoError(t, defaultParams.Validate())

	invalidParams := types.Params{}
	require.Error(t, invalidParams.Validate())

	invalidParams = types.Params{
		MaxVkeySizeBytes: 1,
		UploadChunkSize:  0,
		UploadChunkGas:   1,
	}
	require.Error(t, invalidParams.Validate())

	invalidParams = types.Params{
		MaxVkeySizeBytes: 1,
		UploadChunkSize:  1,
		UploadChunkGas:   0,
	}
	require.Error(t, invalidParams.Validate())
}

func TestGasCostForSize(t *testing.T) {
	params := types.DefaultParams()

	cost, err := params.GasCostForSize(20)
	require.NoError(t, err)
	require.Equal(t, uint64(10_000), cost)

	cost, err = params.GasCostForSize(21)
	require.NoError(t, err)
	require.Equal(t, uint64(20_000), cost)

	_, err = params.GasCostForSize(0)
	require.Error(t, err)

	_, err = params.GasCostForSize(params.MaxVkeySizeBytes + 1)
	require.ErrorIs(t, err, types.ErrVKeyTooLarge)
}

func TestParamsString(t *testing.T) {
	params := types.NewParams(42, 7, 9000)

	expected, err := json.Marshal(params)
	require.NoError(t, err)
	require.Equal(t, string(expected), params.String())
}
