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

	invalidParams = types.DefaultParams()
	invalidParams.MaxGroth16ProofSizeBytes = 0
	require.Error(t, invalidParams.Validate())

	invalidParams = types.DefaultParams()
	invalidParams.MaxGroth16PublicInputSizeBytes = 0
	require.Error(t, invalidParams.Validate())

	invalidParams = types.DefaultParams()
	invalidParams.MaxUltraHonkProofSizeBytes = 0
	require.Error(t, invalidParams.Validate())

	invalidParams = types.DefaultParams()
	invalidParams.MaxUltraHonkPublicInputSizeBytes = 0
	require.Error(t, invalidParams.Validate())

	// Proof/input params must not exceed 512 KiB ceiling.
	invalidParams = types.DefaultParams()
	invalidParams.MaxGroth16ProofSizeBytes = types.MaxAllowedProofOrInputSizeBytes + 1
	require.Error(t, invalidParams.Validate())

	invalidParams = types.DefaultParams()
	invalidParams.MaxGroth16PublicInputSizeBytes = types.MaxAllowedProofOrInputSizeBytes + 1
	require.Error(t, invalidParams.Validate())

	invalidParams = types.DefaultParams()
	invalidParams.MaxUltraHonkProofSizeBytes = types.MaxAllowedProofOrInputSizeBytes + 1
	require.Error(t, invalidParams.Validate())

	invalidParams = types.DefaultParams()
	invalidParams.MaxUltraHonkPublicInputSizeBytes = types.MaxAllowedProofOrInputSizeBytes + 1
	require.Error(t, invalidParams.Validate())

	// VKey param must not exceed 1 MiB ceiling.
	invalidParams = types.DefaultParams()
	invalidParams.MaxVkeySizeBytes = types.MaxAllowedVKeySizeBytes + 1
	require.Error(t, invalidParams.Validate())

	// VKey param at exactly 1 MiB ceiling is valid.
	validParams := types.DefaultParams()
	validParams.MaxVkeySizeBytes = types.MaxAllowedVKeySizeBytes
	require.NoError(t, validParams.Validate())

	// Proof param at exactly 512 KiB ceiling is valid.
	validParams = types.DefaultParams()
	validParams.MaxGroth16ProofSizeBytes = types.MaxAllowedProofOrInputSizeBytes
	require.NoError(t, validParams.Validate())
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

func TestWithMaxLimitDefaults(t *testing.T) {
	t.Run("fills zero-value max limits", func(t *testing.T) {
		params := types.Params{
			MaxVkeySizeBytes: 1,
			UploadChunkSize:  2,
			UploadChunkGas:   3,
		}

		got := params.WithMaxLimitDefaults()
		require.Equal(t, types.DefaultMaxGroth16ProofSizeBytes, got.MaxGroth16ProofSizeBytes)
		require.Equal(t, types.DefaultMaxGroth16PublicInputSizeBytes, got.MaxGroth16PublicInputSizeBytes)
		require.Equal(t, types.DefaultMaxUltraHonkProofSizeBytes, got.MaxUltraHonkProofSizeBytes)
		require.Equal(t, types.DefaultMaxUltraHonkPublicInputSizeBytes, got.MaxUltraHonkPublicInputSizeBytes)
	})

	t.Run("does not overwrite already-set max limits", func(t *testing.T) {
		params := types.DefaultParams()
		params.MaxGroth16ProofSizeBytes = 11
		params.MaxGroth16PublicInputSizeBytes = 12
		params.MaxUltraHonkProofSizeBytes = 13
		params.MaxUltraHonkPublicInputSizeBytes = 14

		got := params.WithMaxLimitDefaults()
		require.Equal(t, uint64(11), got.MaxGroth16ProofSizeBytes)
		require.Equal(t, uint64(12), got.MaxGroth16PublicInputSizeBytes)
		require.Equal(t, uint64(13), got.MaxUltraHonkProofSizeBytes)
		require.Equal(t, uint64(14), got.MaxUltraHonkPublicInputSizeBytes)
	})
}
