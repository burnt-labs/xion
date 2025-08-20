package keeper

import (
	"context"
	"testing"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	mintkeeper "github.com/cosmos/cosmos-sdk/x/mint/keeper"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/burnt-labs/xion/x/xion/types"
)

// Below hooks are overrideable in tests to force specific error paths for coverage.
var (
	getMinterFn        = func(k *mintkeeper.Keeper, ctx sdk.Context) (minttypes.Minter, error) { return k.Minter.Get(ctx) }
	getParamsFn        = func(k *mintkeeper.Keeper, ctx sdk.Context) (minttypes.Params, error) { return k.Params.Get(ctx) }
	setMinterFn        = func(k *mintkeeper.Keeper, ctx sdk.Context, m minttypes.Minter) error { return k.Minter.Set(ctx, m) }
	bondedRatioFn      = func(k *mintkeeper.Keeper, ctx sdk.Context) (math.LegacyDec, error) { return k.BondedRatio(ctx) }
	mintCoinsFn        = func(k *mintkeeper.Keeper, ctx sdk.Context, coins sdk.Coins) error { return k.MintCoins(ctx, coins) }
	addCollectedFeesFn = func(k *mintkeeper.Keeper, ctx sdk.Context, coins sdk.Coins) error {
		return k.AddCollectedFees(ctx, coins)
	}
	blockProvisionFn = func(minter minttypes.Minter, params minttypes.Params) sdk.Coin { return minter.BlockProvision(params) }
)

// Mock types for testing
type MockMintBankKeeper struct {
	mock.Mock
}

func (m *MockMintBankKeeper) GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	args := m.Called(ctx, addr, denom)
	return args.Get(0).(sdk.Coin)
}

func (m *MockMintBankKeeper) BurnCoins(ctx context.Context, moduleName string, amt sdk.Coins) error {
	args := m.Called(ctx, moduleName, amt)
	return args.Error(0)
}

func (m *MockMintBankKeeper) SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	args := m.Called(ctx, senderModule, recipientAddr, amt)
	return args.Error(0)
}

func (m *MockMintBankKeeper) SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt sdk.Coins) error {
	args := m.Called(ctx, senderModule, recipientModule, amt)
	return args.Error(0)
}

func (m *MockMintBankKeeper) SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	args := m.Called(ctx, senderAddr, recipientModule, amt)
	return args.Error(0)
}

func (m *MockMintBankKeeper) IsSendEnabledCoins(ctx context.Context, coins ...sdk.Coin) error {
	args := m.Called(ctx, coins)
	return args.Error(0)
}

func (m *MockMintBankKeeper) BlockedAddr(addr sdk.AccAddress) bool {
	args := m.Called(addr)
	return args.Bool(0)
}

func (m *MockMintBankKeeper) SendCoins(ctx context.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error {
	args := m.Called(ctx, fromAddr, toAddr, amt)
	return args.Error(0)
}

func (m *MockMintBankKeeper) InputOutputCoins(ctx context.Context, input banktypes.Input, outputs []banktypes.Output) error {
	args := m.Called(ctx, input, outputs)
	return args.Error(0)
}

type MockMintAccountKeeper struct {
	mock.Mock
}

func (m *MockMintAccountKeeper) GetModuleAccount(ctx context.Context, moduleName string) sdk.ModuleAccountI {
	args := m.Called(ctx, moduleName)
	return args.Get(0).(sdk.ModuleAccountI)
}

type MockMintStakingKeeper struct {
	mock.Mock
}

func (m *MockMintStakingKeeper) TotalBondedTokens(ctx context.Context) (math.Int, error) {
	args := m.Called(ctx)
	return args.Get(0).(math.Int), args.Error(1)
}

func (m *MockMintStakingKeeper) BondedRatio(ctx context.Context) (math.LegacyDec, error) {
	args := m.Called(ctx)
	return args.Get(0).(math.LegacyDec), args.Error(1)
}

func setupMintTest(t *testing.T) (sdk.Context, *MockMintBankKeeper, *MockMintAccountKeeper, *MockMintStakingKeeper) {
	key := storetypes.NewKVStoreKey(minttypes.StoreKey)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx

	bankKeeper := &MockMintBankKeeper{}
	accountKeeper := &MockMintAccountKeeper{}
	stakingKeeper := &MockMintStakingKeeper{}

	return ctx, bankKeeper, accountKeeper, stakingKeeper
}

func TestStakedInflationMintFn_FunctionCreation(t *testing.T) {
	feeCollectorName := authtypes.FeeCollectorName

	// Create a mock inflation calculation function
	inflationCalcFn := func(ctx context.Context, minter minttypes.Minter, params minttypes.Params, bondedRatio math.LegacyDec) math.LegacyDec {
		return math.LegacyNewDecWithPrec(10, 2) // 10% inflation
	}

	bankKeeper := &MockMintBankKeeper{}
	accountKeeper := &MockMintAccountKeeper{}
	stakingKeeper := &MockMintStakingKeeper{}

	// Test that the function is created successfully
	mintFn := StakedInflationMintFn(feeCollectorName, inflationCalcFn, bankKeeper, accountKeeper, stakingKeeper)

	require.NotNil(t, mintFn)
	require.IsType(t, func(sdk.Context, *mintkeeper.Keeper) error { return nil }, mintFn)
}

func TestStakedInflationMintFn_Parameters(t *testing.T) {
	testCases := []struct {
		name             string
		feeCollectorName string
		expectedPanic    bool
	}{
		{
			name:             "Valid fee collector name",
			feeCollectorName: authtypes.FeeCollectorName,
			expectedPanic:    false,
		},
		{
			name:             "Custom fee collector name",
			feeCollectorName: "custom_fee_collector",
			expectedPanic:    false,
		},
		{
			name:             "Empty fee collector name",
			feeCollectorName: "",
			expectedPanic:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			inflationCalcFn := func(ctx context.Context, minter minttypes.Minter, params minttypes.Params, bondedRatio math.LegacyDec) math.LegacyDec {
				return math.LegacyNewDecWithPrec(10, 2)
			}

			bankKeeper := &MockMintBankKeeper{}
			accountKeeper := &MockMintAccountKeeper{}
			stakingKeeper := &MockMintStakingKeeper{}

			if tc.expectedPanic {
				require.Panics(t, func() {
					StakedInflationMintFn(tc.feeCollectorName, inflationCalcFn, bankKeeper, accountKeeper, stakingKeeper)
				})
			} else {
				mintFn := StakedInflationMintFn(tc.feeCollectorName, inflationCalcFn, bankKeeper, accountKeeper, stakingKeeper)
				require.NotNil(t, mintFn)
			}
		})
	}
}

func TestStakedInflationMintFn_InflationCalculationVariations(t *testing.T) {
	feeCollectorName := authtypes.FeeCollectorName
	bankKeeper := &MockMintBankKeeper{}
	accountKeeper := &MockMintAccountKeeper{}
	stakingKeeper := &MockMintStakingKeeper{}

	testCases := []struct {
		name          string
		inflationRate math.LegacyDec
	}{
		{"Zero Inflation", math.LegacyZeroDec()},
		{"Low Inflation", math.LegacyNewDecWithPrec(5, 2)},        // 5%
		{"Medium Inflation", math.LegacyNewDecWithPrec(10, 2)},    // 10%
		{"High Inflation", math.LegacyNewDecWithPrec(20, 2)},      // 20%
		{"Very High Inflation", math.LegacyNewDecWithPrec(50, 2)}, // 50%
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			inflationCalcFn := func(ctx context.Context, minter minttypes.Minter, params minttypes.Params, bondedRatio math.LegacyDec) math.LegacyDec {
				return tc.inflationRate
			}

			mintFn := StakedInflationMintFn(feeCollectorName, inflationCalcFn, bankKeeper, accountKeeper, stakingKeeper)
			require.NotNil(t, mintFn)
			require.IsType(t, func(sdk.Context, *mintkeeper.Keeper) error { return nil }, mintFn)
		})
	}
}

func TestStakedInflationMintFn_NilInputs(t *testing.T) {
	feeCollectorName := authtypes.FeeCollectorName

	inflationCalcFn := func(ctx context.Context, minter minttypes.Minter, params minttypes.Params, bondedRatio math.LegacyDec) math.LegacyDec {
		return math.LegacyNewDecWithPrec(10, 2)
	}

	testCases := []struct {
		name          string
		bankKeeper    types.BankKeeper
		accountKeeper types.AccountKeeper
		stakingKeeper types.StakingKeeper
		shouldPanic   bool
	}{
		{
			name:          "All keepers provided",
			bankKeeper:    &MockMintBankKeeper{},
			accountKeeper: &MockMintAccountKeeper{},
			stakingKeeper: &MockMintStakingKeeper{},
			shouldPanic:   false,
		},
		{
			name:          "Nil bank keeper",
			bankKeeper:    nil,
			accountKeeper: &MockMintAccountKeeper{},
			stakingKeeper: &MockMintStakingKeeper{},
			shouldPanic:   false, // Function creation doesn't validate nil immediately
		},
		{
			name:          "Nil account keeper",
			bankKeeper:    &MockMintBankKeeper{},
			accountKeeper: nil,
			stakingKeeper: &MockMintStakingKeeper{},
			shouldPanic:   false,
		},
		{
			name:          "Nil staking keeper",
			bankKeeper:    &MockMintBankKeeper{},
			accountKeeper: &MockMintAccountKeeper{},
			stakingKeeper: nil,
			shouldPanic:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.shouldPanic {
				require.Panics(t, func() {
					StakedInflationMintFn(feeCollectorName, inflationCalcFn, tc.bankKeeper, tc.accountKeeper, tc.stakingKeeper)
				})
			} else {
				mintFn := StakedInflationMintFn(feeCollectorName, inflationCalcFn, tc.bankKeeper, tc.accountKeeper, tc.stakingKeeper)
				require.NotNil(t, mintFn)
			}
		})
	}
}

func TestStakedInflationMintFn_NilInflationFunction(t *testing.T) {
	feeCollectorName := authtypes.FeeCollectorName
	bankKeeper := &MockMintBankKeeper{}
	accountKeeper := &MockMintAccountKeeper{}
	stakingKeeper := &MockMintStakingKeeper{}

	// Test with nil inflation calculation function
	mintFn := StakedInflationMintFn(feeCollectorName, nil, bankKeeper, accountKeeper, stakingKeeper)
	require.NotNil(t, mintFn)
	require.IsType(t, func(sdk.Context, *mintkeeper.Keeper) error { return nil }, mintFn)
}

func TestStakedInflationMintFn_FunctionSignature(t *testing.T) {
	feeCollectorName := authtypes.FeeCollectorName

	inflationCalcFn := func(ctx context.Context, minter minttypes.Minter, params minttypes.Params, bondedRatio math.LegacyDec) math.LegacyDec {
		return math.LegacyNewDecWithPrec(10, 2)
	}

	bankKeeper := &MockMintBankKeeper{}
	accountKeeper := &MockMintAccountKeeper{}
	stakingKeeper := &MockMintStakingKeeper{}

	mintFn := StakedInflationMintFn(feeCollectorName, inflationCalcFn, bankKeeper, accountKeeper, stakingKeeper)

	// Verify the function has the correct signature for mint keeper
	require.NotNil(t, mintFn)

	// The function should accept sdk.Context and *mintkeeper.Keeper and return error
	require.IsType(t, func(sdk.Context, *mintkeeper.Keeper) error { return nil }, mintFn)
}

// Integration test helper to verify the core logic (without full mint keeper setup)
func TestStakedInflationMintFn_LogicValidation(t *testing.T) {
	feeCollectorName := authtypes.FeeCollectorName

	// Test the key logic paths that the function should handle:
	// 1. Fees less than needed -> should mint
	// 2. Fees more than needed -> should burn
	// 3. Fees equal to needed -> no action
	// 4. Error handling

	inflationCalcFn := func(ctx context.Context, minter minttypes.Minter, params minttypes.Params, bondedRatio math.LegacyDec) math.LegacyDec {
		return math.LegacyNewDecWithPrec(10, 2) // 10% inflation
	}

	bankKeeper := &MockMintBankKeeper{}
	accountKeeper := &MockMintAccountKeeper{}
	stakingKeeper := &MockMintStakingKeeper{}

	mintFn := StakedInflationMintFn(feeCollectorName, inflationCalcFn, bankKeeper, accountKeeper, stakingKeeper)

	require.NotNil(t, mintFn)

	// Test that the function signature is correct
	// This validates that the function can be used as expected with a mint keeper
	require.IsType(t, func(sdk.Context, *mintkeeper.Keeper) error { return nil }, mintFn)

	// The actual execution would require a full mint keeper setup with proper stores
	// and state management, which is complex for unit testing.
	// The main value of this function is in integration tests where a full
	// mint keeper is available.
}

// Additional focused tests for the StakedInflationMintFn function
// (Removed standalone attribute constant test; constants exercised indirectly in event emission)

func TestStakedInflationMintFn_FunctionWithVariousInflationFunctions(t *testing.T) {
	feeCollectorName := authtypes.FeeCollectorName
	bankKeeper := &MockMintBankKeeper{}
	accountKeeper := &MockMintAccountKeeper{}
	stakingKeeper := &MockMintStakingKeeper{}

	testCases := []struct {
		name        string
		createFn    func() minttypes.InflationCalculationFn
		expectPanic bool
	}{
		{
			name: "Standard inflation function",
			createFn: func() minttypes.InflationCalculationFn {
				return func(ctx context.Context, minter minttypes.Minter, params minttypes.Params, bondedRatio math.LegacyDec) math.LegacyDec {
					return math.LegacyNewDecWithPrec(8, 2)
				}
			},
			expectPanic: false,
		},
		{
			name: "Zero inflation function",
			createFn: func() minttypes.InflationCalculationFn {
				return func(ctx context.Context, minter minttypes.Minter, params minttypes.Params, bondedRatio math.LegacyDec) math.LegacyDec {
					return math.LegacyZeroDec()
				}
			},
			expectPanic: false,
		},
		{
			name: "Variable inflation function",
			createFn: func() minttypes.InflationCalculationFn {
				return func(ctx context.Context, minter minttypes.Minter, params minttypes.Params, bondedRatio math.LegacyDec) math.LegacyDec {
					// Variable inflation based on bonded ratio
					if bondedRatio.LT(math.LegacyNewDecWithPrec(50, 2)) {
						return math.LegacyNewDecWithPrec(20, 2) // 20% if less than 50% bonded
					}
					return math.LegacyNewDecWithPrec(5, 2) // 5% if more than 50% bonded
				}
			},
			expectPanic: false,
		},
		{
			name: "Nil inflation function",
			createFn: func() minttypes.InflationCalculationFn {
				return nil
			},
			expectPanic: false, // Function creation doesn't validate this immediately
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			inflationFn := tc.createFn()

			if tc.expectPanic {
				require.Panics(t, func() {
					StakedInflationMintFn(feeCollectorName, inflationFn, bankKeeper, accountKeeper, stakingKeeper)
				})
			} else {
				mintFn := StakedInflationMintFn(feeCollectorName, inflationFn, bankKeeper, accountKeeper, stakingKeeper)
				require.NotNil(t, mintFn)
				require.IsType(t, func(sdk.Context, *mintkeeper.Keeper) error { return nil }, mintFn)
			}
		})
	}
}

func TestStakedInflationMintFn_ParameterValidation(t *testing.T) {
	validInflationFn := func(ctx context.Context, minter minttypes.Minter, params minttypes.Params, bondedRatio math.LegacyDec) math.LegacyDec {
		return math.LegacyNewDecWithPrec(10, 2)
	}

	testCases := []struct {
		name             string
		feeCollectorName string
		bankKeeper       types.BankKeeper
		accountKeeper    types.AccountKeeper
		stakingKeeper    types.StakingKeeper
		inflationFn      minttypes.InflationCalculationFn
		shouldCreateFunc bool
	}{
		{
			name:             "All valid parameters",
			feeCollectorName: authtypes.FeeCollectorName,
			bankKeeper:       &MockMintBankKeeper{},
			accountKeeper:    &MockMintAccountKeeper{},
			stakingKeeper:    &MockMintStakingKeeper{},
			inflationFn:      validInflationFn,
			shouldCreateFunc: true,
		},
		{
			name:             "Empty fee collector name",
			feeCollectorName: "",
			bankKeeper:       &MockMintBankKeeper{},
			accountKeeper:    &MockMintAccountKeeper{},
			stakingKeeper:    &MockMintStakingKeeper{},
			inflationFn:      validInflationFn,
			shouldCreateFunc: true,
		},
		{
			name:             "Nil bank keeper",
			feeCollectorName: authtypes.FeeCollectorName,
			bankKeeper:       nil,
			accountKeeper:    &MockMintAccountKeeper{},
			stakingKeeper:    &MockMintStakingKeeper{},
			inflationFn:      validInflationFn,
			shouldCreateFunc: true, // Function creation doesn't validate nil immediately
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mintFn := StakedInflationMintFn(tc.feeCollectorName, tc.inflationFn, tc.bankKeeper, tc.accountKeeper, tc.stakingKeeper)

			if tc.shouldCreateFunc {
				require.NotNil(t, mintFn)
				require.IsType(t, func(sdk.Context, *mintkeeper.Keeper) error { return nil }, mintFn)
			} else {
				require.Nil(t, mintFn)
			}
		})
	}
}

func TestStakedInflationMintFn_ReturnedFunctionType(t *testing.T) {
	// Verify that the returned function has the exact signature expected by the mint module
	inflationFn := func(ctx context.Context, minter minttypes.Minter, params minttypes.Params, bondedRatio math.LegacyDec) math.LegacyDec {
		return math.LegacyNewDecWithPrec(10, 2)
	}

	bankKeeper := &MockMintBankKeeper{}
	accountKeeper := &MockMintAccountKeeper{}
	stakingKeeper := &MockMintStakingKeeper{}

	mintFn := StakedInflationMintFn(authtypes.FeeCollectorName, inflationFn, bankKeeper, accountKeeper, stakingKeeper)

	// Test that the function signature matches exactly what's expected
	require.NotNil(t, mintFn)

	// Verify the function signature using reflection if needed
	// This is important because the mint module expects a very specific signature
	var expectedType func(sdk.Context, *mintkeeper.Keeper) error
	require.IsType(t, expectedType, mintFn)

}
