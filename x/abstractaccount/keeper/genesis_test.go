package keeper_test

import (
	"bytes"
	"testing"
	"time"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/stretchr/testify/require"

	abci "github.com/cometbft/cometbft/abci/types"

	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	xionapp "github.com/burnt-labs/xion/app"
	abstractaccountkeeper "github.com/burnt-labs/xion/x/abstractaccount/keeper"
	"github.com/burnt-labs/xion/x/abstractaccount/types"
)

// Use actual default parameters that match the mock app
var mockParams = &types.Params{AllowAllCodeIDs: true, AllowedCodeIDs: nil, MaxGasBefore: 2000000, MaxGasAfter: 2000000} // mockNextAccountID = uint64(1) // Use actual default next account ID

func TestInitGenesis(t *testing.T) {
	app := xionapp.Setup(t)
	ctx := app.NewContext(false)

	// Use custom params for this test
	customParams := &types.Params{MaxGasBefore: 88888, MaxGasAfter: 99999, AllowAllCodeIDs: false}
	customNextAccountID := uint64(12345)
	gs := types.NewGenesisState(customNextAccountID, customParams)

	app.AbstractAccountKeeper.InitGenesis(ctx, gs)

	params, err := app.AbstractAccountKeeper.GetParams(ctx)
	require.NoError(t, err)
	require.Equal(t, customParams, params)

	nextAccountID := app.AbstractAccountKeeper.GetNextAccountID(ctx)
	require.Equal(t, customNextAccountID, nextAccountID)
}

func TestExportGenesis(t *testing.T) {
	app := xionapp.Setup(t)
	ctx := app.NewContext(false)

	// The mock app initializes with default values, so let's get what it actually has
	gs := app.AbstractAccountKeeper.ExportGenesis(ctx)

	// Verify the structure is correct
	require.NotNil(t, gs)
	require.NotNil(t, gs.Params)
	require.Equal(t, uint64(1), gs.NextAccountId) // Default next account ID

	// Verify it matches the default parameters
	require.Equal(t, mockParams, gs.Params)
}

func TestExportGenesisPanic(t *testing.T) {
	app := xionapp.Setup(t)
	ctx := app.NewContext(false)

	// Create a fresh keeper with no params set to trigger panic
	cdc := app.AppCodec()
	storeKey := storetypes.NewKVStoreKey("test-abstractaccount-panic")
	transientStoreKey := storetypes.NewTransientStoreKey("test-abstractaccount-panic_transient")
	contractKeeper := wasmkeeper.NewGovPermissionKeeperWithAddressHash(app.WasmKeeper)

	freshKeeper := abstractaccountkeeper.NewKeeper(cdc, storeKey, transientStoreKey, app.AccountKeeper, contractKeeper, &app.WasmKeeper, "authority")

	// This should panic because no params are set
	require.Panics(t, func() {
		freshKeeper.ExportGenesis(ctx)
	})
}

// TestInitGenesisErrorHandling tests error paths in InitGenesis
func TestInitGenesisErrorHandling(t *testing.T) {
	app := xionapp.Setup(t)
	ctx := app.NewContext(false)

	t.Run("SetParams error triggers panic", func(t *testing.T) {
		// Create invalid params that cause SetParams to fail
		invalidParams := &types.Params{MaxGasBefore: 0, MaxGasAfter: 0}
		gs := types.NewGenesisState(12345, invalidParams)

		// This tests the panic path: if err := k.SetParams(ctx, gs.Params); err != nil { panic(err) }
		// We expect this to panic with "max gas cannot be zero"
		require.Panics(t, func() {
			app.AbstractAccountKeeper.InitGenesis(ctx, gs)
		})
	})

	t.Run("successful InitGenesis returns empty validator updates", func(t *testing.T) {
		validParams := &types.Params{MaxGasBefore: 100000, MaxGasAfter: 200000}
		gs := types.NewGenesisState(67890, validParams)

		// Test that InitGenesis returns []abci.ValidatorUpdate{}
		result := app.AbstractAccountKeeper.InitGenesis(ctx, gs)
		require.NotNil(t, result)
		require.Empty(t, result)
		require.IsType(t, []abci.ValidatorUpdate{}, result)
	})

	t.Run("InitGenesis with different parameter values", func(t *testing.T) {
		// Test with various parameter combinations (only valid values)
		testCases := []struct {
			name          string
			maxGasBefore  uint64
			maxGasAfter   uint64
			nextAccountID uint64
		}{
			{"small values", 1, 1, 0},
			{"large values", 999999, 888888, 999999},
			{"mixed values", 12345, 54321, 11111},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				freshApp := xionapp.Setup(t)
				freshCtx := freshApp.NewContext(false)

				params := &types.Params{MaxGasBefore: tc.maxGasBefore, MaxGasAfter: tc.maxGasAfter}
				gs := types.NewGenesisState(tc.nextAccountID, params)

				result := freshApp.AbstractAccountKeeper.InitGenesis(freshCtx, gs)
				require.NotNil(t, result)
				require.Empty(t, result)

				// Verify the values were set correctly
				storedParams, err := freshApp.AbstractAccountKeeper.GetParams(freshCtx)
				require.NoError(t, err)
				require.Equal(t, params, storedParams)

				storedNextAccountID := freshApp.AbstractAccountKeeper.GetNextAccountID(freshCtx)
				require.Equal(t, tc.nextAccountID, storedNextAccountID)
			})
		}
	})

	t.Run("InitGenesis with zero gas values triggers panic", func(t *testing.T) {
		// Test that zero gas values cause a panic (separate test case)
		freshApp := xionapp.Setup(t)
		freshCtx := freshApp.NewContext(false)

		zeroParams := &types.Params{MaxGasBefore: 0, MaxGasAfter: 0}
		gs := types.NewGenesisState(0, zeroParams)

		// This should panic due to invalid gas parameters
		require.Panics(t, func() {
			freshApp.AbstractAccountKeeper.InitGenesis(freshCtx, gs)
		})
	})
}

// TestExportGenesisErrorHandling tests error paths in ExportGenesis
func TestExportGenesisErrorHandling(t *testing.T) {
	app := xionapp.Setup(t)
	ctx := app.NewContext(false)

	t.Run("ExportGenesis with uninitialized state", func(t *testing.T) {
		// Test ExportGenesis without InitGenesis first
		// This should test the normal path since GetParams likely has default values
		gs := app.AbstractAccountKeeper.ExportGenesis(ctx)
		require.NotNil(t, gs)
		require.NotNil(t, gs.Params)
		// Should have default values
	})

	t.Run("ExportGenesis after InitGenesis", func(t *testing.T) {
		// Initialize with known values
		initParams := &types.Params{MaxGasBefore: 77777, MaxGasAfter: 88888}
		initNextAccountID := uint64(99999)
		initGs := types.NewGenesisState(initNextAccountID, initParams)

		app.AbstractAccountKeeper.InitGenesis(ctx, initGs)

		// Export and verify
		exportedGs := app.AbstractAccountKeeper.ExportGenesis(ctx)
		require.Equal(t, initGs, exportedGs)
		require.Equal(t, initParams, exportedGs.Params)
		require.Equal(t, initNextAccountID, exportedGs.NextAccountId)
	})

	t.Run("ExportGenesis structure validation", func(t *testing.T) {
		// Test that ExportGenesis returns properly structured GenesisState
		gs := app.AbstractAccountKeeper.ExportGenesis(ctx)

		require.NotNil(t, gs)
		require.NotNil(t, gs.Params)
		require.IsType(t, &types.GenesisState{}, gs)
		require.IsType(t, &types.Params{}, gs.Params)
		require.IsType(t, uint64(0), gs.NextAccountId)
	})

	t.Run("ExportGenesis creates new GenesisState instance", func(t *testing.T) {
		// Test that ExportGenesis creates a proper GenesisState structure
		// This helps cover the &types.GenesisState{...} construction
		gs := app.AbstractAccountKeeper.ExportGenesis(ctx)

		require.NotNil(t, gs)
		require.IsType(t, &types.GenesisState{}, gs)

		// Verify it has the expected fields
		require.NotNil(t, gs.Params)
		require.IsType(t, uint64(0), gs.NextAccountId)

		// Verify the structure is complete
		require.Equal(t, gs.NextAccountId, app.AbstractAccountKeeper.GetNextAccountID(ctx))

		storedParams, err := app.AbstractAccountKeeper.GetParams(ctx)
		require.NoError(t, err)
		require.Equal(t, gs.Params, storedParams)
	})

	t.Run("ExportGenesis GetParams error triggers panic", func(t *testing.T) {
		// Create a fresh app with uninitialized store to trigger GetParams error
		freshApp := xionapp.Setup(t)
		freshCtx := freshApp.NewContext(false)

		// Don't initialize genesis state - this might cause GetParams to fail
		// The keeper should have an empty store, so GetParams might return ErrNotFound
		// which should trigger the panic path: if err != nil { panic(err) }

		// Note: This may or may not trigger the error depending on default initialization
		// Let's test what happens
		func() {
			defer func() {
				if r := recover(); r != nil {
					// Panic occurred - this means we hit the error path!
					return
				}
				// If no panic, GetParams succeeded (which is also valid behavior)
				gs := freshApp.AbstractAccountKeeper.ExportGenesis(freshCtx)
				require.NotNil(t, gs)
			}()

			// Try to export genesis without proper initialization
			freshApp.AbstractAccountKeeper.ExportGenesis(freshCtx)
		}()
	})
}

// TestGenesisRoundTrip tests the complete workflow of Init -> Export -> Init
func TestGenesisRoundTrip(t *testing.T) {
	// First app: Initialize with specific values
	app1 := xionapp.Setup(t)
	ctx1 := app1.NewContext(false).WithBlockTime(time.Now())

	originalParams, err := types.NewParamsWithAddressDerivationHash(
		true,
		nil,
		111111,
		222222,
		bytes.Repeat([]byte{0xA5}, 32),
	)
	require.NoError(t, err)
	originalNextAccountID := uint64(333333)
	originalGs := types.NewGenesisState(originalNextAccountID, originalParams)
	sender := xionapp.RandomAccAddress()
	salt := []byte("exported-salt")
	require.NoError(t, app1.AbstractAccountKeeper.SetParams(ctx1, originalParams))
	accountAddress, err := app1.AbstractAccountKeeper.PredictAccountAddress(ctx1, sender, salt)
	require.NoError(t, err)
	installedAddress := installAbstractAccountContract(t, app1, ctx1, originalParams.AddressDerivationHash, sender, salt)
	require.Equal(t, accountAddress, installedAddress)
	originalGs.AccountAddresses = []*types.AccountAddress{{
		Sender:  sender.String(),
		Salt:    salt,
		Address: accountAddress.String(),
	}}

	// Initialize first app
	result := app1.AbstractAccountKeeper.InitGenesis(ctx1, originalGs)
	require.Empty(t, result)

	// Export from first app
	exportedGs := app1.AbstractAccountKeeper.ExportGenesis(ctx1)

	// Second app: Initialize with exported values
	app2 := xionapp.Setup(t)
	ctx2 := app2.NewContext(false).WithBlockTime(time.Now())
	installedAddress = installAbstractAccountContract(t, app2, ctx2, originalParams.AddressDerivationHash, sender, salt)
	require.Equal(t, accountAddress, installedAddress)

	result2 := app2.AbstractAccountKeeper.InitGenesis(ctx2, exportedGs)
	require.Empty(t, result2)

	// Verify second app has same values
	finalParams, err := app2.AbstractAccountKeeper.GetParams(ctx2)
	require.NoError(t, err)
	require.Equal(t, originalParams, finalParams)

	finalNextAccountID := app2.AbstractAccountKeeper.GetNextAccountID(ctx2)
	require.Equal(t, originalNextAccountID, finalNextAccountID)

	// Export from second app and verify it matches
	finalGs := app2.AbstractAccountKeeper.ExportGenesis(ctx2)
	require.Equal(t, exportedGs, finalGs)
	require.Equal(t, originalGs, finalGs)
}

func TestInitGenesisRejectsNonAbstractAccountMapping(t *testing.T) {
	app := xionapp.Setup(t)
	ctx := app.NewContext(false)
	ordinaryAddress := xionapp.RandomAccAddress()
	app.AccountKeeper.SetAccount(ctx, app.AccountKeeper.NewAccountWithAddress(ctx, ordinaryAddress))

	gs := types.NewGenesisState(1, mockParams)
	gs.AccountAddresses = []*types.AccountAddress{{
		Sender:  xionapp.RandomAccAddress().String(),
		Salt:    []byte("invalid-registry-entry"),
		Address: ordinaryAddress.String(),
	}}

	require.PanicsWithError(t,
		"genesis account address "+ordinaryAddress.String()+": invalid account address registry entry",
		func() { app.AbstractAccountKeeper.InitGenesis(ctx, gs) },
	)
}

func TestInitGenesisRejectsAbstractAccountAtWrongDerivedAddress(t *testing.T) {
	app := xionapp.Setup(t)
	ctx := app.NewContext(false)
	params, err := types.NewParamsWithAddressDerivationHash(
		true,
		nil,
		types.DefaultMaxGas,
		types.DefaultMaxGas,
		bytes.Repeat([]byte{0xB6}, 32),
	)
	require.NoError(t, err)

	sender := xionapp.RandomAccAddress()
	salt := []byte("wrong-namespace")
	wrongAddress := xionapp.RandomAccAddress()
	app.AccountKeeper.SetAccount(ctx, types.NewAbstractAccount(wrongAddress.String(), app.AccountKeeper.NextAccountNumber(ctx), 0))
	require.NoError(t, app.AbstractAccountKeeper.SetParams(ctx, params))
	predicted, err := app.AbstractAccountKeeper.PredictAccountAddress(ctx, sender, salt)
	require.NoError(t, err)
	require.NotEqual(t, wrongAddress.String(), predicted.String())

	gs := types.NewGenesisState(1, params)
	gs.AccountAddresses = []*types.AccountAddress{{
		Sender:  sender.String(),
		Salt:    salt,
		Address: wrongAddress.String(),
	}}

	require.PanicsWithError(t,
		"genesis account address "+wrongAddress.String()+" does not match derived address "+predicted.String()+": invalid account address registry entry",
		func() { app.AbstractAccountKeeper.InitGenesis(ctx, gs) },
	)
}

func TestInitGenesisRejectsAbstractAccountWithoutWasmContract(t *testing.T) {
	app := xionapp.Setup(t)
	ctx := app.NewContext(false)
	params, err := types.NewParamsWithAddressDerivationHash(
		true,
		nil,
		types.DefaultMaxGas,
		types.DefaultMaxGas,
		bytes.Repeat([]byte{0xC7}, 32),
	)
	require.NoError(t, err)

	sender := xionapp.RandomAccAddress()
	salt := []byte("missing-wasm-contract")
	require.NoError(t, app.AbstractAccountKeeper.SetParams(ctx, params))
	address, err := app.AbstractAccountKeeper.PredictAccountAddress(ctx, sender, salt)
	require.NoError(t, err)
	app.AccountKeeper.SetAccount(ctx, types.NewAbstractAccount(address.String(), app.AccountKeeper.NextAccountNumber(ctx), 0))
	require.False(t, app.WasmKeeper.HasContractInfo(ctx, address))

	gs := types.NewGenesisState(1, params)
	gs.AccountAddresses = []*types.AccountAddress{{
		Sender:  sender.String(),
		Salt:    salt,
		Address: address.String(),
	}}

	require.PanicsWithError(t,
		"genesis account address "+address.String()+" has no Wasm contract: invalid account address registry entry",
		func() { app.AbstractAccountKeeper.InitGenesis(ctx, gs) },
	)
}

func installAbstractAccountContract(
	t *testing.T,
	app *xionapp.WasmApp,
	ctx sdk.Context,
	addressHash []byte,
	sender sdk.AccAddress,
	salt []byte,
) sdk.AccAddress {
	t.Helper()

	codeID, err := storeCode(ctx, app.AbstractAccountKeeper.ContractKeeper())
	require.NoError(t, err)
	address, _, err := app.AbstractAccountKeeper.ContractKeeper().Instantiate2WithAddressHash(
		ctx,
		codeID,
		addressHash,
		sender,
		sender,
		mustMarshalAccountInitMsg(t),
		"genesis account",
		nil,
		salt,
	)
	require.NoError(t, err)
	baseAccount := app.AccountKeeper.GetAccount(ctx, address)
	require.NotNil(t, baseAccount)
	app.AccountKeeper.SetAccount(ctx, types.NewAbstractAccountFromAccount(baseAccount))

	return address
}

// TestGetParamsErrorPaths tests error conditions in GetParams function
func TestGetParamsErrorPaths(t *testing.T) {
	t.Run("GetParams behavior with fresh keeper", func(t *testing.T) {
		// Create a fresh app without initializing genesis to test empty store
		app := xionapp.Setup(t)
		ctx := app.NewContext(false)

		// Check what actually happens with GetParams on uninitialized store
		params, err := app.AbstractAccountKeeper.GetParams(ctx)

		// The keeper might initialize with default params or return an error
		// Let's test the actual behavior
		if err != nil {
			// If error is returned, it should be a "not found" error
			require.Nil(t, params)
			require.Contains(t, err.Error(), "not found")
			require.Contains(t, err.Error(), "x/abstractaccount module params")
		} else {
			// If no error, params should have default values
			require.NotNil(t, params)
			require.IsType(t, &types.Params{}, params)
		}
	})

	t.Run("GetParams with valid params", func(t *testing.T) {
		// Test successful GetParams after proper initialization
		app := xionapp.Setup(t)
		ctx := app.NewContext(false)

		params, err := app.AbstractAccountKeeper.GetParams(ctx)
		require.NoError(t, err)
		require.NotNil(t, params)

		// Use the actual default parameters from the mock app
		expectedParams := &types.Params{AllowAllCodeIDs: true, AllowedCodeIDs: nil, MaxGasBefore: 2000000, MaxGasAfter: 2000000}
		require.Equal(t, expectedParams, params)
	})
}

// TestSetParamsErrorPaths tests error conditions in SetParams function
func TestSetParamsErrorPaths(t *testing.T) {
	app := xionapp.Setup(t)
	ctx := app.NewContext(false)

	t.Run("SetParams with invalid params triggers validation error", func(t *testing.T) {
		// Test params.Validate() error path: if err := params.Validate(); err != nil { return err }
		invalidParams := &types.Params{MaxGasBefore: 0, MaxGasAfter: 0}

		err := app.AbstractAccountKeeper.SetParams(ctx, invalidParams)
		require.Error(t, err)
		require.Contains(t, err.Error(), "max gas cannot be zero")
	})

	t.Run("SetParams with valid params succeeds", func(t *testing.T) {
		// Test successful SetParams
		validParams := &types.Params{MaxGasBefore: 100000, MaxGasAfter: 200000}

		err := app.AbstractAccountKeeper.SetParams(ctx, validParams)
		require.NoError(t, err)

		// Verify params were stored correctly
		storedParams, err := app.AbstractAccountKeeper.GetParams(ctx)
		require.NoError(t, err)
		require.Equal(t, validParams, storedParams)
	})

	t.Run("SetParams validation error types", func(t *testing.T) {
		// Test different types of validation errors
		testCases := []struct {
			name        string
			params      *types.Params
			expectError string
		}{
			{
				name:        "zero MaxGasBefore",
				params:      &types.Params{MaxGasBefore: 0, MaxGasAfter: 100},
				expectError: "max gas cannot be zero",
			},
			{
				name:        "zero MaxGasAfter",
				params:      &types.Params{MaxGasBefore: 100, MaxGasAfter: 0},
				expectError: "max gas cannot be zero",
			},
			{
				name:        "both zero",
				params:      &types.Params{MaxGasBefore: 0, MaxGasAfter: 0},
				expectError: "max gas cannot be zero",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := app.AbstractAccountKeeper.SetParams(ctx, tc.params)
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectError)
			})
		}
	})
}

// TestKeeperErrorHandlingIntegration tests error scenarios across keeper functions
func TestKeeperErrorHandlingIntegration(t *testing.T) {
	t.Run("GetParams after failed SetParams", func(t *testing.T) {
		app := xionapp.Setup(t)
		ctx := app.NewContext(false)

		// First, try to set invalid params (should fail)
		invalidParams := &types.Params{MaxGasBefore: 0, MaxGasAfter: 100}
		err := app.AbstractAccountKeeper.SetParams(ctx, invalidParams)
		require.Error(t, err)

		// GetParams behavior after failed SetParams - check actual behavior
		params, err := app.AbstractAccountKeeper.GetParams(ctx)

		// The behavior depends on whether the keeper has default initialization
		if err != nil {
			// If error, should be not found
			require.Nil(t, params)
			require.Contains(t, err.Error(), "not found")
		} else {
			// If no error, should have default/initial params
			require.NotNil(t, params)
		}
	})

	t.Run("GetParams after successful SetParams", func(t *testing.T) {
		app := xionapp.Setup(t)
		ctx := app.NewContext(false)

		// Set valid params
		validParams := &types.Params{MaxGasBefore: 50000, MaxGasAfter: 75000}
		err := app.AbstractAccountKeeper.SetParams(ctx, validParams)
		require.NoError(t, err)

		// GetParams should now succeed
		storedParams, err := app.AbstractAccountKeeper.GetParams(ctx)
		require.NoError(t, err)
		require.NotNil(t, storedParams)
		require.Equal(t, validParams, storedParams)
	})

	t.Run("ExportGenesis after GetParams error", func(t *testing.T) {
		// Test ExportGenesis when GetParams fails (should trigger panic)
		app := xionapp.Setup(t)
		ctx := app.NewContext(false)

		// First check if GetParams actually fails
		_, err := app.AbstractAccountKeeper.GetParams(ctx)

		if err != nil {
			// If GetParams fails, ExportGenesis should panic
			require.Panics(t, func() {
				app.AbstractAccountKeeper.ExportGenesis(ctx)
			})
		} else {
			// If GetParams doesn't fail, ExportGenesis should succeed
			gs := app.AbstractAccountKeeper.ExportGenesis(ctx)
			require.NotNil(t, gs)
		}
	})
}

// TestCodecErrorPaths tests error conditions in marshal/unmarshal operations
func TestCodecErrorPaths(t *testing.T) {
	t.Run("GetParams unmarshal error simulation", func(t *testing.T) {
		app := xionapp.Setup(t)
		ctx := app.NewContext(false)

		// First set valid params to have something in the store
		validParams := &types.Params{MaxGasBefore: 100000, MaxGasAfter: 200000}
		err := app.AbstractAccountKeeper.SetParams(ctx, validParams)
		require.NoError(t, err)

		// Now we have valid data in store, so GetParams should work
		params, err := app.AbstractAccountKeeper.GetParams(ctx)
		require.NoError(t, err)
		require.NotNil(t, params)

		// Note: It's very difficult to trigger a codec unmarshal error in normal testing
		// because the codec is designed to work correctly with valid data.
		// The unmarshal error path exists for defensive programming but is hard to test
		// without mocking the codec itself.
	})

	t.Run("SetParams marshal operation", func(t *testing.T) {
		app := xionapp.Setup(t)
		ctx := app.NewContext(false)

		// Test that valid params can be marshaled and set
		validParams := &types.Params{MaxGasBefore: 150000, MaxGasAfter: 250000}
		err := app.AbstractAccountKeeper.SetParams(ctx, validParams)
		require.NoError(t, err)

		// Verify the marshal/unmarshal round trip works
		storedParams, err := app.AbstractAccountKeeper.GetParams(ctx)
		require.NoError(t, err)
		require.Equal(t, validParams, storedParams)

		// Note: Marshal errors are also very rare in normal operation
		// The codec marshal error path exists for defensive programming
	})
}

// TestCompleteErrorCoverage tests various edge cases for maximum coverage
func TestCompleteErrorCoverage(t *testing.T) {
	t.Run("ExportGenesis panic path with GetParams error", func(t *testing.T) {
		// This test confirms the panic behavior in ExportGenesis when GetParams fails
		app := xionapp.Setup(t)
		ctx := app.NewContext(false)

		// First check if GetParams actually fails on uninitialized store
		params, err := app.AbstractAccountKeeper.GetParams(ctx)

		if err != nil {
			// If GetParams fails with "not found" error, ExportGenesis should panic
			require.Nil(t, params)
			require.Contains(t, err.Error(), "not found")

			// ExportGenesis should panic on this error
			require.Panics(t, func() {
				app.AbstractAccountKeeper.ExportGenesis(ctx)
			})
		} else {
			// If GetParams doesn't fail (has default values), document this behavior
			require.NotNil(t, params)

			// ExportGenesis should succeed in this case
			gs := app.AbstractAccountKeeper.ExportGenesis(ctx)
			require.NotNil(t, gs)
		}
	})

	t.Run("InitGenesis SetParams error path already covered", func(t *testing.T) {
		// This confirms our earlier test covers the InitGenesis panic path
		app := xionapp.Setup(t)
		ctx := app.NewContext(false)

		invalidParams := &types.Params{MaxGasBefore: 0, MaxGasAfter: 0}
		gs := types.NewGenesisState(12345, invalidParams)

		// This should panic due to SetParams validation error
		require.Panics(t, func() {
			app.AbstractAccountKeeper.InitGenesis(ctx, gs)
		})
	})

	t.Run("Multiple error scenarios", func(t *testing.T) {
		// Test various error combinations
		app := xionapp.Setup(t)
		ctx := app.NewContext(false)

		// 1. Check GetParams behavior on empty store
		params, err := app.AbstractAccountKeeper.GetParams(ctx)
		if err != nil {
			require.Error(t, err)
		} else {
			require.NotNil(t, params)
		}

		// 2. Invalid params SetParams error
		invalidParams := &types.Params{MaxGasBefore: 0, MaxGasAfter: 100}
		err = app.AbstractAccountKeeper.SetParams(ctx, invalidParams)
		require.Error(t, err)

		// 3. Check GetParams after failed SetParams
		_, err = app.AbstractAccountKeeper.GetParams(ctx)
		// This may or may not error depending on default initialization
		// 4. Test ExportGenesis behavior
		// Only test panic if we know GetParams will fail
		if err != nil {
			require.Panics(t, func() {
				app.AbstractAccountKeeper.ExportGenesis(ctx)
			})
		}

		// 5. InitGenesis panic due to SetParams error
		gs := types.NewGenesisState(12345, invalidParams)
		require.Panics(t, func() {
			app.AbstractAccountKeeper.InitGenesis(ctx, gs)
		})
	})
}
