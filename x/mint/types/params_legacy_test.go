package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

func TestParamKeyTable(t *testing.T) {
	table := ParamKeyTable()
	require.NotNil(t, table)

	// We can't directly access the internal structure, but we can test that it doesn't panic
	require.NotPanics(t, func() {
		ParamKeyTable()
	})
}

func TestParamSetPairs(t *testing.T) {
	params := DefaultParams()
	pairs := params.ParamSetPairs()

	require.Len(t, pairs, 6)

	// Check that all expected parameter keys are present
	expectedKeys := [][]byte{
		KeyMintDenom,
		KeyInflationRateChange,
		KeyInflationMax,
		KeyInflationMin,
		KeyGoalBonded,
		KeyBlocksPerYear,
	}

	pairKeys := make([][]byte, len(pairs))
	for i, pair := range pairs {
		pairKeys[i] = pair.Key
	}

	for _, expectedKey := range expectedKeys {
		found := false
		for _, pairKey := range pairKeys {
			if string(expectedKey) == string(pairKey) {
				found = true
				break
			}
		}
		require.True(t, found, "Expected key %s not found in param set pairs", string(expectedKey))
	}
}

func TestParamSetPairsValidation(t *testing.T) {
	params := DefaultParams()
	pairs := params.ParamSetPairs()

	// Test that each pair has a validation function
	for _, pair := range pairs {
		require.NotNil(t, pair.ValidatorFn, "Validation function should not be nil for key %s", string(pair.Key))

		// Test that validation functions work with default values
		switch string(pair.Key) {
		case string(KeyMintDenom):
			err := pair.ValidatorFn(params.MintDenom)
			require.NoError(t, err)
		case string(KeyInflationRateChange):
			err := pair.ValidatorFn(params.InflationRateChange)
			require.NoError(t, err)
		case string(KeyInflationMax):
			err := pair.ValidatorFn(params.InflationMax)
			require.NoError(t, err)
		case string(KeyInflationMin):
			err := pair.ValidatorFn(params.InflationMin)
			require.NoError(t, err)
		case string(KeyGoalBonded):
			err := pair.ValidatorFn(params.GoalBonded)
			require.NoError(t, err)
		case string(KeyBlocksPerYear):
			err := pair.ValidatorFn(params.BlocksPerYear)
			require.NoError(t, err)
		}
	}
}

func TestParamSetImplementation(t *testing.T) {
	params := &Params{}

	// Test that Params implements paramtypes.ParamSet
	require.Implements(t, (*paramtypes.ParamSet)(nil), params)
}
