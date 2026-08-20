package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	storetypes "cosmossdk.io/store/types"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	simapptesting "github.com/burnt-labs/xion/x/abstractaccount/simapp/testing"
	"github.com/burnt-labs/xion/x/abstractaccount/keeper"
	"github.com/burnt-labs/xion/x/abstractaccount/types"
)

func TestNewKeeper(t *testing.T) {
	app := simapptesting.MakeSimpleMockApp(t)

	// Test normal creation (already works from the app)
	require.NotNil(t, app.AbstractAccountKeeper)

	// Test panic conditions
	cdc := app.AppCodec()
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	transientStoreKey := storetypes.NewTransientStoreKey(types.TransientStoreKey)
	contractKeeper := wasmkeeper.NewGovPermissionKeeperWithAddressHash(app.WasmKeeper)

	t.Run("panic when AccountKeeper is nil", func(t *testing.T) {
		require.Panics(t, func() {
			keeper.NewKeeper(cdc, storeKey, transientStoreKey, nil, contractKeeper, &app.WasmKeeper, "authority")
		})
	})

	t.Run("panic when ContractKeeper is nil", func(t *testing.T) {
		require.Panics(t, func() {
			keeper.NewKeeper(cdc, storeKey, transientStoreKey, app.AccountKeeper, nil, &app.WasmKeeper, "authority")
		})
	})

	t.Run("panic when ViewKeeper is nil", func(t *testing.T) {
		require.Panics(t, func() {
			keeper.NewKeeper(cdc, storeKey, transientStoreKey, app.AccountKeeper, contractKeeper, nil, "authority")
		})
	})
}

func TestGetAndIncrementNextAccountID(t *testing.T) {
	app := simapptesting.MakeSimpleMockApp(t)
	ctx := app.NewContext(false)

	id := app.AbstractAccountKeeper.GetAndIncrementNextAccountID(ctx)
	require.Equal(t, uint64(1), id)

	id = app.AbstractAccountKeeper.GetNextAccountID(ctx)
	require.Equal(t, uint64(2), id)
}

func TestSignerAddress(t *testing.T) {
	app := simapptesting.MakeSimpleMockApp(t)
	ctx := app.NewContext(false)

	// Test getting signer address when not set (should return empty address)
	signerAddr := app.AbstractAccountKeeper.GetSignerAddress(ctx)
	require.Equal(t, 0, len(signerAddr))

	// Test setting and getting signer address
	testAddr := simapptesting.MakeRandomAddress()
	app.AbstractAccountKeeper.SetSignerAddress(ctx, testAddr)

	retrievedAddr := app.AbstractAccountKeeper.GetSignerAddress(ctx)
	require.Equal(t, testAddr, retrievedAddr)

	// Test deleting signer address
	app.AbstractAccountKeeper.DeleteSignerAddress(ctx)

	signerAddr = app.AbstractAccountKeeper.GetSignerAddress(ctx)
	require.Equal(t, 0, len(signerAddr))
}

func TestMigration(t *testing.T) {
	app := simapptesting.MakeSimpleMockApp(t)
	ctx := app.NewContext(false)

	// Test migration
	migrator := app.AbstractAccountKeeper.Migrator()
	err := migrator.Migrate1to2(ctx)
	require.NoError(t, err)
}

func TestSetParamsError(t *testing.T) {
	app := simapptesting.MakeSimpleMockApp(t)
	ctx := app.NewContext(false)

	// Test with invalid params (MaxGasAfter = 0 while MaxGasBefore > 0)
	invalidParams := &types.Params{
		MaxGasBefore:    1000,
		MaxGasAfter:     0, // This should trigger validation error
		AllowAllCodeIDs: true,
	}

	err := app.AbstractAccountKeeper.SetParams(ctx, invalidParams)
	require.Error(t, err)
}
