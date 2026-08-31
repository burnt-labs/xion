package keeper_test

import (
	"bytes"
	"testing"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/stretchr/testify/require"

	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	xionapp "github.com/burnt-labs/xion/app"
	"github.com/burnt-labs/xion/x/abstractaccount/keeper"
	"github.com/burnt-labs/xion/x/abstractaccount/types"
)

func TestNewKeeper(t *testing.T) {
	app := xionapp.Setup(t)

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
	app := xionapp.Setup(t)
	ctx := app.NewContext(false)

	id := app.AbstractAccountKeeper.GetAndIncrementNextAccountID(ctx)
	require.Equal(t, uint64(1), id)

	id = app.AbstractAccountKeeper.GetNextAccountID(ctx)
	require.Equal(t, uint64(2), id)
}

func TestSignerAddress(t *testing.T) {
	app := xionapp.Setup(t)
	ctx := app.NewContext(false)

	// Test getting signer address when not set (should return empty address)
	signerAddr := app.AbstractAccountKeeper.GetSignerAddress(ctx)
	require.Equal(t, 0, len(signerAddr))

	// Test setting and getting signer address
	testAddr := xionapp.RandomAccAddress()
	app.AbstractAccountKeeper.SetSignerAddress(ctx, testAddr)

	retrievedAddr := app.AbstractAccountKeeper.GetSignerAddress(ctx)
	require.Equal(t, testAddr, retrievedAddr)

	// Test deleting signer address
	app.AbstractAccountKeeper.DeleteSignerAddress(ctx)

	signerAddr = app.AbstractAccountKeeper.GetSignerAddress(ctx)
	require.Equal(t, 0, len(signerAddr))
}

func TestMigration(t *testing.T) {
	app := xionapp.Setup(t)
	ctx := app.NewContext(false)

	// Test migration
	migrator := app.AbstractAccountKeeper.Migrator()
	err := migrator.Migrate1to2(ctx)
	require.NoError(t, err)
}

func TestSetParamsError(t *testing.T) {
	app := xionapp.Setup(t)
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

// TestSignerAddressIsScopedToTx guards against a stale AA signer leaking from a
// transaction that passed the AnteHandler but failed message execution into a
// later transaction in the same block.
//
// This mirrors BaseApp's runTx sequencing. The transient store is block-scoped —
// Commit is what clears it — and ante writes are flushed as soon as the ante
// chain succeeds, whereas the PostHandler's cleanup lives in the runMsgs branch
// that is thrown away when message execution fails.
func TestSignerAddressIsScopedToTx(t *testing.T) {
	app := xionapp.Setup(t)
	blockCtx := app.NewContext(false)

	aaSigner := sdk.AccAddress([]byte("abstract_account_sig"))

	tx1 := blockCtx.WithTxBytes([]byte("tx-one"))
	tx2 := blockCtx.WithTxBytes([]byte("tx-two"))

	// tx1 is an AA tx: the AnteHandler records the signer, and BaseApp commits
	// that write to the block store once the ante chain succeeds.
	anteCtx, anteCache := cacheContext(tx1)
	app.AbstractAccountKeeper.SetSignerAddress(anteCtx, aaSigner)
	anteCache.Write()

	// tx1's messages fail. The PostHandler still runs, but against the runMsgs
	// branch, which is discarded — so its DeleteSignerAddress never lands.
	msgCtx, msgCache := cacheContext(tx1)
	require.Equal(t, aaSigner, app.AbstractAccountKeeper.GetSignerAddress(msgCtx),
		"the failing tx's own post handling should still see its signer")
	app.AbstractAccountKeeper.DeleteSignerAddress(msgCtx)
	_ = msgCache // messages failed, so the branch is never written back

	// tx2 is an unrelated, non-AA tx later in the same block. It must not observe
	// tx1's signer, or its PostHandler would sudo after_tx on tx1's account.
	require.Nil(t, app.AbstractAccountKeeper.GetSignerAddress(tx2),
		"a stale AA signer must not leak into a later tx in the same block")

	// The signer also must not leak back through the shared block context.
	require.Nil(t, app.AbstractAccountKeeper.GetSignerAddress(blockCtx),
		"the block context must not expose a per-tx signer")
}

// cacheContext branches ctx the way BaseApp.cacheTxContext does.
func cacheContext(ctx sdk.Context) (sdk.Context, storetypes.CacheMultiStore) {
	msCache := ctx.MultiStore().CacheMultiStore()

	return ctx.WithMultiStore(msCache), msCache
}

// TestSignerAddressGasIsIndependentOfTxSize pins the gas cost of recording the
// AA signer to a constant.
//
// gaskv charges TransientGasConfig.WriteCostPerByte over the length of the key,
// so a key that embeds the raw transaction would make every abstract account
// transaction pay gas proportional to its own size — worst exactly for the large
// JWT and ZK proof payloads this module serves.
func TestSignerAddressGasIsIndependentOfTxSize(t *testing.T) {
	app := xionapp.Setup(t)
	signer := sdk.AccAddress([]byte("abstract_account_sig"))

	gasFor := func(txBytes []byte) storetypes.Gas {
		ctx := app.NewContext(false).
			WithTxBytes(txBytes).
			WithGasMeter(storetypes.NewInfiniteGasMeter())

		before := ctx.GasMeter().GasConsumed()
		app.AbstractAccountKeeper.SetSignerAddress(ctx, signer)

		return ctx.GasMeter().GasConsumed() - before
	}

	small := gasFor([]byte("tx"))
	require.Greater(t, small, storetypes.Gas(0),
		"sanity check: recording the AA signer should consume gas")
	large := gasFor(bytes.Repeat([]byte{0xAB}, 16*1024))

	require.Equal(t, small, large,
		"recording the AA signer must cost the same for a 2-byte and a 16KiB tx")
}
