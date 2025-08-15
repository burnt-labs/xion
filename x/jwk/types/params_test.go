package types_test

import (
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

func TestValidateDeploymentGas(t *testing.T) {
	tests := []struct {
		name      string
		input     interface{}
		expectErr bool
	}{
		{
			name:      "valid uint64",
			input:     uint64(10000),
			expectErr: false,
		},
		{
			name:      "zero value",
			input:     uint64(0),
			expectErr: true,
		},
		{
			name:      "wrong type string",
			input:     "not-uint64",
			expectErr: true,
		},
		{
			name:      "wrong type int",
			input:     10000,
			expectErr: true,
		},
		{
			name:      "nil input",
			input:     nil,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a param set to test the validation function
			params := types.DefaultParams()
			paramSet := params.ParamSetPairs()

			// Find the deployment gas validator
			var validator func(interface{}) error
			for _, pair := range paramSet {
				if string(pair.Key) == "DeploymentGas" {
					validator = pair.ValidatorFn
					break
				}
			}
			require.NotNil(t, validator)

			err := validator(tt.input)
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
		input     interface{}
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
			name:      "wrong type string",
			input:     "not-uint64",
			expectErr: true,
		},
		{
			name:      "wrong type int",
			input:     30000,
			expectErr: true,
		},
		{
			name:      "nil input",
			input:     nil,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a param set to test the validation function
			params := types.DefaultParams()
			paramSet := params.ParamSetPairs()

			// Find the time offset validator
			var validator func(interface{}) error
			for _, pair := range paramSet {
				if string(pair.Key) == "TimeOffset" {
					validator = pair.ValidatorFn
					break
				}
			}
			require.NotNil(t, validator)

			err := validator(tt.input)
			if tt.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestParamKeyTable(t *testing.T) {
	table := types.ParamKeyTable()
	require.NotNil(t, table)

	// Test that we can use the table to validate param pairs
	params := types.DefaultParams()
	paramSet := params.ParamSetPairs()
	require.Len(t, paramSet, 2)

	// Verify the param pairs have the expected keys
	keys := make([]string, len(paramSet))
	for i, pair := range paramSet {
		keys[i] = string(pair.Key)
	}
	require.Contains(t, keys, "DeploymentGas")
	require.Contains(t, keys, "TimeOffset")
}

func TestNewParams(t *testing.T) {
	params := types.NewParams(123, 456)
	require.Equal(t, uint64(123), params.TimeOffset)
	require.Equal(t, uint64(456), params.DeploymentGas)
}

func TestDefaultParams(t *testing.T) {
	params := types.DefaultParams()
	require.Equal(t, uint64(10_000), params.DeploymentGas)
	require.Equal(t, uint64(30_000), params.TimeOffset)

	// Default params should be valid
	require.NoError(t, params.Validate())
}
