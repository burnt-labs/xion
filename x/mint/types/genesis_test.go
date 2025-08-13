package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestDefaultInflationCalculationFn(t *testing.T) {
	ctx := sdk.Context{}
	minter := DefaultInitialMinter()
	params := DefaultParams()
	bondedRatio := math.LegacyNewDecWithPrec(5, 1) // 0.5

	// Test that the function returns the same as minter.NextInflationRate
	expected := minter.NextInflationRate(params, bondedRatio)
	actual := DefaultInflationCalculationFn(ctx, minter, params, bondedRatio)

	require.Equal(t, expected, actual)
	require.True(t, actual.GTE(params.InflationMin))
	require.True(t, actual.LTE(params.InflationMax))
}

func TestNewGenesisState(t *testing.T) {
	minter := DefaultInitialMinter()
	params := DefaultParams()

	genesisState := NewGenesisState(minter, params)

	require.NotNil(t, genesisState)
	require.Equal(t, minter, genesisState.Minter)
	require.Equal(t, params, genesisState.Params)
}

func TestDefaultGenesisState(t *testing.T) {
	genesisState := DefaultGenesisState()

	require.NotNil(t, genesisState)
	require.Equal(t, DefaultInitialMinter(), genesisState.Minter)
	require.Equal(t, DefaultParams(), genesisState.Params)
}

func TestValidateGenesis(t *testing.T) {
	tests := []struct {
		name          string
		genesisState  GenesisState
		expectedError bool
	}{
		{
			name:          "valid default genesis",
			genesisState:  *DefaultGenesisState(),
			expectedError: false,
		},
		{
			name: "valid custom genesis",
			genesisState: GenesisState{
				Minter: NewMinter(
					math.LegacyNewDecWithPrec(5, 2), // 0.05 inflation
					math.LegacyNewDec(1000000),      // annual provisions
				),
				Params: NewParams(
					"uxion",
					math.LegacyNewDecWithPrec(13, 2), // 0.13 inflation rate change
					math.LegacyNewDecWithPrec(20, 2), // 0.20 inflation max
					math.LegacyNewDecWithPrec(7, 2),  // 0.07 inflation min
					math.LegacyNewDecWithPrec(67, 2), // 0.67 goal bonded
					uint64(6311520),                  // blocks per year
				),
			},
			expectedError: false,
		},
		{
			name: "invalid params - negative inflation max",
			genesisState: GenesisState{
				Minter: DefaultInitialMinter(),
				Params: Params{
					MintDenom:           "uxion",
					InflationRateChange: math.LegacyNewDecWithPrec(13, 2),
					InflationMax:        math.LegacyNewDec(-1), // Invalid: negative
					InflationMin:        math.LegacyNewDecWithPrec(7, 2),
					GoalBonded:          math.LegacyNewDecWithPrec(67, 2),
					BlocksPerYear:       uint64(6311520),
				},
			},
			expectedError: true,
		},
		{
			name: "invalid minter - negative inflation",
			genesisState: GenesisState{
				Minter: NewMinter(
					math.LegacyNewDec(-1), // Invalid: negative inflation
					math.LegacyNewDec(1000000),
				),
				Params: DefaultParams(),
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGenesis(tt.genesisState)
			if tt.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
