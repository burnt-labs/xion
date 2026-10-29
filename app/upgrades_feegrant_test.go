package app

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/appmodule"

	"github.com/cosmos/cosmos-sdk/x/feegrant"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

// TestNextUpgradeHandlerMigratesFeeAllowanceQueue covers the queue entries
// pre-v32 feegrant stored with an empty value: once one expires the v0.55
// EndBlocker fails, which halted xion-testnet-2 at height 19286361. After the
// upgrade the entry holds the BoolValue encoding and the expired grant prunes.
func TestNextUpgradeHandlerMigratesFeeAllowanceQueue(t *testing.T) {
	gapp := Setup(t)
	ctx := gapp.NewContext(false).WithBlockTime(time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC))
	granter, grantee := RandomAccAddress(), RandomAccAddress()
	exp := ctx.BlockTime().Add(time.Hour)
	gapp.AccountKeeper.SetAccount(ctx, gapp.AccountKeeper.NewAccountWithAddress(ctx, grantee))
	require.NoError(t, gapp.FeeGrantKeeper.GrantAllowance(ctx, granter, grantee, &feegrant.BasicAllowance{Expiration: &exp}))

	queueKey, err := collections.EncodeKeyWithPrefix(
		feegrant.FeeAllowanceQueueKeyPrefix,
		gapp.FeeGrantKeeper.FeeAllowanceQueue.KeyCodec(),
		collections.Join3(exp, grantee, granter),
	)
	require.NoError(t, err)
	store := ctx.KVStore(gapp.GetKey(feegrant.StoreKey))
	require.Equal(t, []byte{1}, store.Get(queueKey))
	store.Set(queueKey, []byte{})

	endBlocker, ok := gapp.ModuleManager.Modules[feegrant.ModuleName].(appmodule.HasEndBlocker)
	require.True(t, ok)
	expired := ctx.WithBlockTime(exp.Add(time.Second))
	before, _ := expired.CacheContext()
	require.ErrorContains(t, endBlocker.EndBlock(before), "wanted size to be at least 1")

	_, err = gapp.NextUpgradeHandler(ctx, upgradetypes.Plan{Name: UpgradeName, Height: 100}, gapp.ModuleManager.GetVersionMap())
	require.NoError(t, err)
	require.Equal(t, []byte{1}, store.Get(queueKey))

	require.NoError(t, endBlocker.EndBlock(expired))
	_, err = gapp.FeeGrantKeeper.GetAllowance(expired, granter, grantee)
	require.ErrorIs(t, err, collections.ErrNotFound)
	require.False(t, store.Has(queueKey))
}
