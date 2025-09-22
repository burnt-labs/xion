package types

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestNextInflation(t *testing.T) {
	minter := DefaultInitialMinter()
	params := DefaultParams()
	blocksPerYr := math.NewIntFromUint64(params.BlocksPerYear).ToLegacyDec()

	// Governing Mechanism:
	//    inflationRateChangePerYear = (1- BondedRatio/ GoalBonded) * MaxInflationRateChange

	tests := []struct {
		bondedRatio, setInflation, expChange math.LegacyDec
	}{
		// with 0% bonded atom supply the inflation should increase by InflationRateChange
		{math.LegacyZeroDec(), math.LegacyNewDecWithPrec(7, 2), params.InflationRateChange.Quo(blocksPerYr)},

		// 100% bonded, starting at 20% inflation and being reduced
		// (1 - (1/0.67))*(0.13/8667)
		{
			math.LegacyOneDec(), math.LegacyNewDecWithPrec(20, 2),
			math.LegacyOneDec().Sub(math.LegacyOneDec().Quo(params.GoalBonded)).Mul(params.InflationRateChange).Quo(blocksPerYr),
		},

		// 50% bonded, starting at 10% inflation and being increased
		{
			math.LegacyNewDecWithPrec(5, 1), math.LegacyNewDecWithPrec(10, 2),
			math.LegacyOneDec().Sub(math.LegacyNewDecWithPrec(5, 1).Quo(params.GoalBonded)).Mul(params.InflationRateChange).Quo(blocksPerYr),
		},

		// test 7% minimum stop (testing with 100% bonded)
		{math.LegacyOneDec(), math.LegacyNewDecWithPrec(7, 2), math.LegacyZeroDec()},
		{math.LegacyOneDec(), math.LegacyNewDecWithPrec(700000001, 10), math.LegacyNewDecWithPrec(-1, 10)},

		// test 20% maximum stop (testing with 0% bonded)
		{math.LegacyZeroDec(), math.LegacyNewDecWithPrec(20, 2), math.LegacyZeroDec()},
		{math.LegacyZeroDec(), math.LegacyNewDecWithPrec(1999999999, 10), math.LegacyNewDecWithPrec(1, 10)},

		// perfect balance shouldn't change inflation
		{math.LegacyNewDecWithPrec(67, 2), math.LegacyNewDecWithPrec(15, 2), math.LegacyZeroDec()},
	}
	for i, tc := range tests {
		minter.Inflation = tc.setInflation

		inflation := minter.NextInflationRate(params, tc.bondedRatio)
		diffInflation := inflation.Sub(tc.setInflation)

		require.True(t, diffInflation.Equal(tc.expChange),
			"Test Index: %v\nDiff:  %v\nExpected: %v\n", i, diffInflation, tc.expChange)
	}
}

func TestBlockProvision(t *testing.T) {
	minter := InitialMinter(math.LegacyNewDecWithPrec(1, 1))
	params := DefaultParams()

	secondsPerYear := int64(60 * 60 * 8766)

	tests := []struct {
		annualProvisions int64
		expProvisions    int64
	}{
		{secondsPerYear / 5, 1},
		{secondsPerYear/5 + 1, 1},
		{(secondsPerYear / 5) * 2, 2},
		{(secondsPerYear / 5) / 2, 0},
	}
	for i, tc := range tests {
		minter.AnnualProvisions = math.LegacyNewDec(tc.annualProvisions)
		provisions := minter.BlockProvision(params)

		expProvisions := sdk.NewCoin(params.MintDenom,
			math.NewInt(tc.expProvisions))

		require.True(t, expProvisions.IsEqual(provisions),
			"test: %v\n\tExp: %v\n\tGot: %v\n",
			i, tc.expProvisions, provisions)
	}
}

func TestValidateMinter(t *testing.T) {
	tests := []struct {
		name        string
		minter      Minter
		expectedErr bool
	}{
		{
			name:        "valid default minter",
			minter:      DefaultInitialMinter(),
			expectedErr: false,
		},
		{
			name: "valid custom minter",
			minter: NewMinter(
				math.LegacyNewDecWithPrec(1, 1), // 0.1 inflation
				math.LegacyNewDec(1000000),      // annual provisions
			),
			expectedErr: false,
		},
		{
			name: "valid zero inflation",
			minter: NewMinter(
				math.LegacyZeroDec(),       // 0 inflation
				math.LegacyNewDec(1000000), // annual provisions
			),
			expectedErr: false,
		},
		{
			name: "invalid negative inflation",
			minter: NewMinter(
				math.LegacyNewDec(-1),      // negative inflation
				math.LegacyNewDec(1000000), // annual provisions
			),
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMinter(tt.minter)
			if tt.expectedErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), "should be positive")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNextAnnualProvisions(t *testing.T) {
	tests := []struct {
		name        string
		minter      Minter
		totalSupply math.Int
		expected    math.LegacyDec
	}{
		{
			name: "10% inflation with 1M total supply",
			minter: NewMinter(
				math.LegacyNewDecWithPrec(1, 1), // 0.1 (10%)
				math.LegacyZeroDec(),            // annual provisions (not used in calculation)
			),
			totalSupply: math.NewInt(1000000),
			expected:    math.LegacyNewDec(100000), // 10% of 1M
		},
		{
			name: "5% inflation with 2M total supply",
			minter: NewMinter(
				math.LegacyNewDecWithPrec(5, 2), // 0.05 (5%)
				math.LegacyZeroDec(),            // annual provisions (not used in calculation)
			),
			totalSupply: math.NewInt(2000000),
			expected:    math.LegacyNewDec(100000), // 5% of 2M
		},
		{
			name: "zero inflation",
			minter: NewMinter(
				math.LegacyZeroDec(), // 0%
				math.LegacyZeroDec(), // annual provisions (not used in calculation)
			),
			totalSupply: math.NewInt(1000000),
			expected:    math.LegacyZeroDec(), // 0% of 1M
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := DefaultParams() // params not used in NextAnnualProvisions
			result := tt.minter.NextAnnualProvisions(params, tt.totalSupply)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestDefaultInitialMinter(t *testing.T) {
	minter := DefaultInitialMinter()

	require.Equal(t, math.LegacyNewDecWithPrec(13, 2), minter.Inflation)
	require.Equal(t, math.LegacyZeroDec(), minter.AnnualProvisions)

	// Test that it's a valid minter
	err := ValidateMinter(minter)
	require.NoError(t, err)
}

// Benchmarking :)
// previously using math.Int operations:
// BenchmarkBlockProvision-4 5000000 220 ns/op
//
// using math.LegacyDec operations: (current implementation)
// BenchmarkBlockProvision-4 3000000 429 ns/op
func BenchmarkBlockProvision(b *testing.B) {
	b.ReportAllocs()
	minter := InitialMinter(math.LegacyNewDecWithPrec(1, 1))
	params := DefaultParams()

	s1 := rand.NewSource(100)
	//nolint:gosec // G404: Use of weak random number generator (math/rand instead of crypto/rand)
	r1 := rand.New(s1)
	minter.AnnualProvisions = math.LegacyNewDec(r1.Int63n(1000000))

	// run the BlockProvision function b.N times
	for n := 0; n < b.N; n++ {
		minter.BlockProvision(params)
	}
}

// Next inflation benchmarking
// BenchmarkNextInflation-4 1000000 1828 ns/op
func BenchmarkNextInflation(b *testing.B) {
	b.ReportAllocs()
	minter := InitialMinter(math.LegacyNewDecWithPrec(1, 1))
	params := DefaultParams()
	bondedRatio := math.LegacyNewDecWithPrec(1, 1)

	// run the NextInflationRate function b.N times
	for n := 0; n < b.N; n++ {
		minter.NextInflationRate(params, bondedRatio)
	}
}

// Next annual provisions benchmarking
// BenchmarkNextAnnualProvisions-4 5000000 251 ns/op
func BenchmarkNextAnnualProvisions(b *testing.B) {
	b.ReportAllocs()
	minter := InitialMinter(math.LegacyNewDecWithPrec(1, 1))
	params := DefaultParams()
	totalSupply := math.NewInt(100000000000000)

	// run the NextAnnualProvisions function b.N times
	for n := 0; n < b.N; n++ {
		minter.NextAnnualProvisions(params, totalSupply)
	}
}
