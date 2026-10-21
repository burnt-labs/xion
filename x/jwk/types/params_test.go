package types_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/burnt-labs/xion/x/jwk/types"
)

func TestParamsValidation(t *testing.T) {
	tests := []struct {
		name      string
		params    types.Params
		expectErr bool
	}{
		{
			name: "valid params",
			params: types.Params{
				DeploymentGas: 10000,
				TimeOffset:    30000,
			},
			expectErr: false,
		},
		{
			name: "zero deployment gas",
			params: types.Params{
				DeploymentGas: 0,
				TimeOffset:    30000,
			},
			expectErr: true,
		},
		{
			name: "zero time offset",
			params: types.Params{
				DeploymentGas: 10000,
				TimeOffset:    0,
			},
			expectErr: true,
		},
		{
			name: "both zero",
			params: types.Params{
				DeploymentGas: 0,
				TimeOffset:    0,
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.params.Validate()
			if tt.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateTimeOffset(t *testing.T) {
	tests := []struct {
		name      string
		input     uint64
		expectErr bool
	}{
		{
			name:      "valid uint64",
			input:     uint64(30000),
			expectErr: false,
		},
		{
			name:      "zero value",
			input:     uint64(0),
			expectErr: true,
		},
		{
			name:      "max int64 is valid",
			input:     uint64(math.MaxInt64),
			expectErr: false,
		},
		{
			name:      "exceeds max int64",
			input:     uint64(math.MaxInt64) + 1,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := types.DefaultParams()
			params.TimeOffset = tt.input
			err := params.Validate()
			if tt.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNewParams(t *testing.T) {
	params := types.NewParams(123, 456)
	require.Equal(t, uint64(123), params.TimeOffset)
	require.Equal(t, uint64(456), params.DeploymentGas)
}

func TestDefaultParams(t *testing.T) {
	params := types.DefaultParams()
	require.Equal(t, uint64(10_000), params.DeploymentGas)
	require.Equal(t, uint64(30_000_000_000), params.TimeOffset)

	// Default params should be valid
	require.NoError(t, params.Validate())
}
