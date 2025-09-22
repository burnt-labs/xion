package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestNewParams(t *testing.T) {
	mintDenom := "uxion"
	inflationRateChange := math.LegacyNewDecWithPrec(13, 2)
	inflationMax := math.LegacyNewDecWithPrec(20, 2)
	inflationMin := math.LegacyNewDecWithPrec(7, 2)
	goalBonded := math.LegacyNewDecWithPrec(67, 2)
	blocksPerYear := uint64(6311520)

	params := NewParams(mintDenom, inflationRateChange, inflationMax, inflationMin, goalBonded, blocksPerYear)

	require.Equal(t, mintDenom, params.MintDenom)
	require.Equal(t, inflationRateChange, params.InflationRateChange)
	require.Equal(t, inflationMax, params.InflationMax)
	require.Equal(t, inflationMin, params.InflationMin)
	require.Equal(t, goalBonded, params.GoalBonded)
	require.Equal(t, blocksPerYear, params.BlocksPerYear)
}

func TestDefaultParams(t *testing.T) {
	params := DefaultParams()

	require.Equal(t, sdk.DefaultBondDenom, params.MintDenom)
	require.Equal(t, math.LegacyNewDecWithPrec(13, 2), params.InflationRateChange)
	require.Equal(t, math.LegacyNewDecWithPrec(20, 2), params.InflationMax)
	require.Equal(t, math.LegacyNewDecWithPrec(7, 2), params.InflationMin)
	require.Equal(t, math.LegacyNewDecWithPrec(67, 2), params.GoalBonded)
	require.Equal(t, uint64(60*60*8766/5), params.BlocksPerYear)

	// Test that default params are valid
	err := params.Validate()
	require.NoError(t, err)
}

func TestParamsValidate(t *testing.T) {
	tests := []struct {
		name        string
		params      Params
		expectedErr bool
		errContains string
	}{
		{
			name:        "valid default params",
			params:      DefaultParams(),
			expectedErr: false,
		},
		{
			name: "valid custom params",
			params: NewParams(
				"uxion",
				math.LegacyNewDecWithPrec(15, 2),
				math.LegacyNewDecWithPrec(25, 2),
				math.LegacyNewDecWithPrec(5, 2),
				math.LegacyNewDecWithPrec(70, 2),
				6000000,
			),
			expectedErr: false,
		},
		{
			name: "empty mint denom",
			params: Params{
				MintDenom:           "",
				InflationRateChange: math.LegacyNewDecWithPrec(13, 2),
				InflationMax:        math.LegacyNewDecWithPrec(20, 2),
				InflationMin:        math.LegacyNewDecWithPrec(7, 2),
				GoalBonded:          math.LegacyNewDecWithPrec(67, 2),
				BlocksPerYear:       6311520,
			},
			expectedErr: true,
			errContains: "mint denom cannot be blank",
		},
		{
			name: "negative inflation rate change",
			params: Params{
				MintDenom:           "uxion",
				InflationRateChange: math.LegacyNewDec(-1),
				InflationMax:        math.LegacyNewDecWithPrec(20, 2),
				InflationMin:        math.LegacyNewDecWithPrec(7, 2),
				GoalBonded:          math.LegacyNewDecWithPrec(67, 2),
				BlocksPerYear:       6311520,
			},
			expectedErr: true,
			errContains: "inflation rate change cannot be negative",
		},
		{
			name: "negative inflation max",
			params: Params{
				MintDenom:           "uxion",
				InflationRateChange: math.LegacyNewDecWithPrec(13, 2),
				InflationMax:        math.LegacyNewDec(-1),
				InflationMin:        math.LegacyNewDecWithPrec(7, 2),
				GoalBonded:          math.LegacyNewDecWithPrec(67, 2),
				BlocksPerYear:       6311520,
			},
			expectedErr: true,
			errContains: "max inflation cannot be negative",
		},
		{
			name: "negative inflation min",
			params: Params{
				MintDenom:           "uxion",
				InflationRateChange: math.LegacyNewDecWithPrec(13, 2),
				InflationMax:        math.LegacyNewDecWithPrec(20, 2),
				InflationMin:        math.LegacyNewDec(-1),
				GoalBonded:          math.LegacyNewDecWithPrec(67, 2),
				BlocksPerYear:       6311520,
			},
			expectedErr: true,
			errContains: "min inflation cannot be negative",
		},
		{
			name: "negative goal bonded",
			params: Params{
				MintDenom:           "uxion",
				InflationRateChange: math.LegacyNewDecWithPrec(13, 2),
				InflationMax:        math.LegacyNewDecWithPrec(20, 2),
				InflationMin:        math.LegacyNewDecWithPrec(7, 2),
				GoalBonded:          math.LegacyNewDec(-1),
				BlocksPerYear:       6311520,
			},
			expectedErr: true,
			errContains: "goal bonded must be positive",
		},
		{
			name: "zero blocks per year",
			params: Params{
				MintDenom:           "uxion",
				InflationRateChange: math.LegacyNewDecWithPrec(13, 2),
				InflationMax:        math.LegacyNewDecWithPrec(20, 2),
				InflationMin:        math.LegacyNewDecWithPrec(7, 2),
				GoalBonded:          math.LegacyNewDecWithPrec(67, 2),
				BlocksPerYear:       0,
			},
			expectedErr: true,
			errContains: "blocks per year must be positive",
		},
		{
			name: "inflation max less than inflation min",
			params: Params{
				MintDenom:           "uxion",
				InflationRateChange: math.LegacyNewDecWithPrec(13, 2),
				InflationMax:        math.LegacyNewDecWithPrec(5, 2),  // 5%
				InflationMin:        math.LegacyNewDecWithPrec(10, 2), // 10%
				GoalBonded:          math.LegacyNewDecWithPrec(67, 2),
				BlocksPerYear:       6311520,
			},
			expectedErr: true,
			errContains: "max inflation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.params.Validate()

			if tt.expectedErr {
				require.Error(t, err)
				if tt.errContains != "" {
					require.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestParamsString(t *testing.T) {
	params := DefaultParams()
	str := params.String()

	require.NotEmpty(t, str)
	require.Contains(t, str, "mint_denom")
	require.Contains(t, str, "inflation_rate_change")
	require.Contains(t, str, "inflation_max")
	require.Contains(t, str, "inflation_min")
	require.Contains(t, str, "goal_bonded")
	require.Contains(t, str, "blocks_per_year")
}

func TestValidateMintDenom(t *testing.T) {
	tests := []struct {
		name        string
		denom       interface{}
		expectedErr bool
		errContains string
	}{
		{
			name:        "valid denom",
			denom:       "uxion",
			expectedErr: false,
		},
		{
			name:        "valid stake denom",
			denom:       "stake",
			expectedErr: false,
		},
		{
			name:        "empty denom",
			denom:       "",
			expectedErr: true,
			errContains: "mint denom cannot be blank",
		},
		{
			name:        "whitespace denom",
			denom:       "   ",
			expectedErr: true,
			errContains: "mint denom cannot be blank",
		},
		{
			name:        "invalid type",
			denom:       123,
			expectedErr: true,
			errContains: "invalid parameter type",
		},
		{
			name:        "invalid denom format",
			denom:       "1invalid",
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMintDenom(tt.denom)

			if tt.expectedErr {
				require.Error(t, err)
				if tt.errContains != "" {
					require.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateInflationRateChange(t *testing.T) {
	tests := []struct {
		name        string
		value       interface{}
		expectedErr bool
		errContains string
	}{
		{
			name:        "valid positive value",
			value:       math.LegacyNewDecWithPrec(13, 2),
			expectedErr: false,
		},
		{
			name:        "valid zero value",
			value:       math.LegacyZeroDec(),
			expectedErr: false,
		},
		{
			name:        "negative value",
			value:       math.LegacyNewDec(-1),
			expectedErr: true,
			errContains: "inflation rate change cannot be negative",
		},
		{
			name:        "invalid type",
			value:       "invalid",
			expectedErr: true,
			errContains: "invalid parameter type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateInflationRateChange(tt.value)

			if tt.expectedErr {
				require.Error(t, err)
				if tt.errContains != "" {
					require.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateInflationMax(t *testing.T) {
	tests := []struct {
		name        string
		value       interface{}
		expectedErr bool
		errContains string
	}{
		{
			name:        "valid positive value",
			value:       math.LegacyNewDecWithPrec(20, 2),
			expectedErr: false,
		},
		{
			name:        "valid zero value",
			value:       math.LegacyZeroDec(),
			expectedErr: false,
		},
		{
			name:        "negative value",
			value:       math.LegacyNewDec(-1),
			expectedErr: true,
			errContains: "max inflation cannot be negative",
		},
		{
			name:        "invalid type",
			value:       "invalid",
			expectedErr: true,
			errContains: "invalid parameter type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateInflationMax(tt.value)

			if tt.expectedErr {
				require.Error(t, err)
				if tt.errContains != "" {
					require.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateInflationMin(t *testing.T) {
	tests := []struct {
		name        string
		value       interface{}
		expectedErr bool
		errContains string
	}{
		{
			name:        "valid positive value",
			value:       math.LegacyNewDecWithPrec(7, 2),
			expectedErr: false,
		},
		{
			name:        "valid zero value",
			value:       math.LegacyZeroDec(),
			expectedErr: false,
		},
		{
			name:        "negative value",
			value:       math.LegacyNewDec(-1),
			expectedErr: true,
			errContains: "min inflation cannot be negative",
		},
		{
			name:        "invalid type",
			value:       "invalid",
			expectedErr: true,
			errContains: "invalid parameter type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateInflationMin(tt.value)

			if tt.expectedErr {
				require.Error(t, err)
				if tt.errContains != "" {
					require.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateGoalBonded(t *testing.T) {
	tests := []struct {
		name        string
		value       interface{}
		expectedErr bool
		errContains string
	}{
		{
			name:        "valid positive value",
			value:       math.LegacyNewDecWithPrec(67, 2),
			expectedErr: false,
		},
		{
			name:        "zero value",
			value:       math.LegacyZeroDec(),
			expectedErr: true,
			errContains: "goal bonded must be positive",
		},
		{
			name:        "negative value",
			value:       math.LegacyNewDec(-1),
			expectedErr: true,
			errContains: "goal bonded must be positive",
		},
		{
			name:        "invalid type",
			value:       "invalid",
			expectedErr: true,
			errContains: "invalid parameter type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGoalBonded(tt.value)

			if tt.expectedErr {
				require.Error(t, err)
				if tt.errContains != "" {
					require.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateBlocksPerYear(t *testing.T) {
	tests := []struct {
		name        string
		value       interface{}
		expectedErr bool
		errContains string
	}{
		{
			name:        "valid positive value",
			value:       uint64(6311520),
			expectedErr: false,
		},
		{
			name:        "zero value",
			value:       uint64(0),
			expectedErr: true,
			errContains: "blocks per year must be positive",
		},
		{
			name:        "invalid type",
			value:       "invalid",
			expectedErr: true,
			errContains: "invalid parameter type",
		},
		{
			name:        "invalid negative int",
			value:       -1,
			expectedErr: true,
			errContains: "invalid parameter type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBlocksPerYear(tt.value)

			if tt.expectedErr {
				require.Error(t, err)
				if tt.errContains != "" {
					require.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}
