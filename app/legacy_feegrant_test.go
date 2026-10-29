package app

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/appmodule"

	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

var (
	queueGranter        = sdk.AccAddress([]byte("legacy_granter______"))
	queueLegacyGrantee  = sdk.AccAddress([]byte("legacy_grantee______"))
	queueCurrentGrantee = sdk.AccAddress([]byte("current_grantee_____"))
	queueLaterGrantee   = sdk.AccAddress([]byte("later_grantee_______"))
)

// setupFeeAllowanceQueue grants three allowances and returns their expiries: an
// expired one stored the pre-v32 way (empty queue value), an expired one stored
// the v0.55 way, and an unexpired one stored the pre-v32 way.
func setupFeeAllowanceQueue(t *testing.T, gapp *WasmApp, ctx sdk.Context) (expired, later time.Time) {
	t.Helper()
	expired = ctx.BlockTime().Add(45 * time.Minute)
	later = ctx.BlockTime().Add(4 * time.Hour)
	for grantee, exp := range map[string]time.Time{
		string(queueLegacyGrantee):  expired,
		string(queueCurrentGrantee): expired,
		string(queueLaterGrantee):   later,
	} {
		addr := sdk.AccAddress(grantee)
		gapp.AccountKeeper.SetAccount(ctx, gapp.AccountKeeper.NewAccountWithAddress(ctx, addr))
		require.NoError(t, gapp.FeeGrantKeeper.GrantAllowance(ctx, queueGranter, addr, &feegrant.BasicAllowance{Expiration: &exp}))
	}

	store := ctx.KVStore(gapp.GetKey(feegrant.StoreKey))
	var legacy [][]byte
	it := storetypes.KVStorePrefixIterator(store, feegrant.FeeAllowanceQueueKeyPrefix)
	for ; it.Valid(); it.Next() {
		_, key, err := gapp.FeeGrantKeeper.FeeAllowanceQueue.KeyCodec().Decode(it.Key()[len(feegrant.FeeAllowanceQueueKeyPrefix):])
		require.NoError(t, err)
		if !key.K2().Equals(queueCurrentGrantee) {
			legacy = append(legacy, it.Key())
		}
	}
	require.NoError(t, it.Close())
	require.Len(t, legacy, 2)
	for _, key := range legacy {
		store.Set(key, []byte{})
	}
	return expired, later
}

// requireLegacyQueueValues asserts the two entries setupFeeAllowanceQueue
// stored the pre-v32 way hold want, and the one stored the v0.55 way holds 0x01.
func requireLegacyQueueValues(t *testing.T, gapp *WasmApp, ctx sdk.Context, want []byte) {
	t.Helper()
	got := map[string][]byte{}
	it := storetypes.KVStorePrefixIterator(ctx.KVStore(gapp.GetKey(feegrant.StoreKey)), feegrant.FeeAllowanceQueueKeyPrefix)
	for ; it.Valid(); it.Next() {
		_, key, err := gapp.FeeGrantKeeper.FeeAllowanceQueue.KeyCodec().Decode(it.Key()[len(feegrant.FeeAllowanceQueueKeyPrefix):])
		require.NoError(t, err)
		got[key.K2().String()] = it.Value()
	}
	require.NoError(t, it.Close())
	require.Equal(t, map[string][]byte{
		queueLegacyGrantee.String():  want,
		queueCurrentGrantee.String(): {1},
		queueLaterGrantee.String():   want,
	}, got)
}

// TestFeeAllowanceQueueFixAtTestnetHaltHeight reproduces the xion-testnet-2
// halt at height 19286361, where the feegrant EndBlocker failed on an expired
// queue entry written before v32, and checks that the PreBlocker rewrite at
// exactly that height lets the block apply.
func TestFeeAllowanceQueueFixAtTestnetHaltHeight(t *testing.T) {
	const haltHeight = 19286361

	gapp := Setup(t)
	ctx := gapp.NewContext(false).
		WithChainID("xion-testnet-2").
		WithBlockTime(time.Date(2026, 10, 29, 15, 0, 0, 0, time.UTC))
	expired, later := setupFeeAllowanceQueue(t, gapp, ctx)
	endBlocker, ok := gapp.ModuleManager.Modules[feegrant.ModuleName].(appmodule.HasEndBlocker)
	require.True(t, ok)

	// One block early, and on another chain at the halt height, nothing changes,
	// and the halt reproduces. Rewriting at any other block would diverge from
	// the state the chain committed.
	for _, c := range []sdk.Context{
		ctx.WithBlockHeight(haltHeight - 1),
		ctx.WithChainID("xion-mainnet-1").WithBlockHeight(haltHeight),
	} {
		c, _ = c.CacheContext()
		_, err := gapp.PreBlocker(c, nil)
		require.NoError(t, err)
		requireLegacyQueueValues(t, gapp, c, []byte{})
		require.ErrorContains(t, endBlocker.EndBlock(c.WithBlockTime(expired.Add(time.Second))), "wanted size to be at least 1")
	}

	ctx = ctx.WithBlockHeight(haltHeight)
	_, err := gapp.PreBlocker(ctx, nil)
	require.NoError(t, err)
	requireLegacyQueueValues(t, gapp, ctx, []byte{1})

	ctx = ctx.WithBlockTime(expired.Add(time.Second))
	require.NoError(t, endBlocker.EndBlock(ctx))
	for _, grantee := range []sdk.AccAddress{queueLegacyGrantee, queueCurrentGrantee} {
		_, err := gapp.FeeGrantKeeper.GetAllowance(ctx, queueGranter, grantee)
		require.ErrorIs(t, err, collections.ErrNotFound, grantee.String())
		has, err := gapp.FeeGrantKeeper.FeeAllowanceQueue.Has(ctx, collections.Join3(expired, grantee, queueGranter))
		require.NoError(t, err)
		require.False(t, has, "queue entry for %s survived pruning", grantee)
	}

	// The rewritten entry that had not expired stays usable, then is pruned.
	_, err = gapp.FeeGrantKeeper.GetAllowance(ctx, queueGranter, queueLaterGrantee)
	require.NoError(t, err)
	ctx = ctx.WithBlockTime(later.Add(time.Second))
	require.NoError(t, endBlocker.EndBlock(ctx))
	_, err = gapp.FeeGrantKeeper.GetAllowance(ctx, queueGranter, queueLaterGrantee)
	require.ErrorIs(t, err, collections.ErrNotFound)
}

// TestNextUpgradeHandlerMigratesFeeAllowanceQueue covers chains that have not
// yet run v32: the upgrade rewrites the legacy entries, so they never reach the
// EndBlocker in the old encoding.
func TestNextUpgradeHandlerMigratesFeeAllowanceQueue(t *testing.T) {
	gapp := Setup(t)
	ctx := gapp.NewContext(false).
		WithChainID("xion-mainnet-1").
		WithBlockTime(time.Date(2026, 10, 29, 15, 0, 0, 0, time.UTC))
	expired, _ := setupFeeAllowanceQueue(t, gapp, ctx)

	_, err := gapp.NextUpgradeHandler(ctx, upgradetypes.Plan{Name: UpgradeName, Height: 100}, gapp.ModuleManager.GetVersionMap())
	require.NoError(t, err)
	requireLegacyQueueValues(t, gapp, ctx, []byte{1})

	endBlocker := gapp.ModuleManager.Modules[feegrant.ModuleName].(appmodule.HasEndBlocker)
	require.NoError(t, endBlocker.EndBlock(ctx.WithBlockTime(expired.Add(time.Second))))
}
