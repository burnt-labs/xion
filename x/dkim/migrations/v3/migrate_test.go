package v3_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/collections"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"

	v3 "github.com/burnt-labs/xion/x/dkim/migrations/v3"
	"github.com/burnt-labs/xion/x/dkim/types"
)

func TestMigrateStore(t *testing.T) {
	encCfg := moduletestutil.MakeTestEncodingConfig()
	types.RegisterInterfaces(encCfg.InterfaceRegistry)

	key := storetypes.NewKVStoreKey(types.ModuleName)
	tkey := storetypes.NewTransientStoreKey("transient_test")
	testCtx := testutil.DefaultContextWithDB(t, key, tkey)
	ctx := testCtx.Ctx

	storeService := runtime.NewKVStoreService(key)
	sb := collections.NewSchemaBuilder(storeService)

	paramsCollection := collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](encCfg.Codec))

	_, err := sb.Build()
	require.NoError(t, err)

	t.Run("existing params with MinRsaKeyBits=0 migrated to default", func(t *testing.T) {
		// Simulate v2 state where MinRsaKeyBits was not yet a field (proto zero value).
		oldParams := types.Params{
			VkeyIdentifier:     1,
			MaxPubkeySizeBytes: 512,
			PublicInputIndices: types.DefaultPublicInputIndices(),
			MinRsaKeyBits:      0,
		}
		err := paramsCollection.Set(ctx, oldParams)
		require.NoError(t, err)

		// Run migration
		err = v3.MigrateStore(ctx.WithLogger(log.NewNopLogger()), paramsCollection)
		require.NoError(t, err)

		// Verify MinRsaKeyBits was set to default
		newParams, err := paramsCollection.Get(ctx)
		require.NoError(t, err)
		require.Equal(t, types.DefaultMinRSAKeyBits, newParams.MinRsaKeyBits)
		// Other fields should be preserved
		require.Equal(t, uint64(1), newParams.VkeyIdentifier)
		require.Equal(t, uint64(512), newParams.MaxPubkeySizeBytes)
	})

	t.Run("existing params with MinRsaKeyBits already set unchanged", func(t *testing.T) {
		customMinBits := uint64(2048)
		existingParams := types.Params{
			VkeyIdentifier:     2,
			MaxPubkeySizeBytes: 1024,
			PublicInputIndices: types.DefaultPublicInputIndices(),
			MinRsaKeyBits:      customMinBits,
		}
		err := paramsCollection.Set(ctx, existingParams)
		require.NoError(t, err)

		// Run migration
		err = v3.MigrateStore(ctx.WithLogger(log.NewNopLogger()), paramsCollection)
		require.NoError(t, err)

		// Verify MinRsaKeyBits was NOT changed
		newParams, err := paramsCollection.Get(ctx)
		require.NoError(t, err)
		require.Equal(t, customMinBits, newParams.MinRsaKeyBits)
		// Other fields should be preserved
		require.Equal(t, uint64(2), newParams.VkeyIdentifier)
		require.Equal(t, uint64(1024), newParams.MaxPubkeySizeBytes)
	})

	t.Run("no existing params sets defaults", func(t *testing.T) {
		// Clear params
		err := paramsCollection.Remove(ctx)
		require.NoError(t, err)

		// Run migration
		err = v3.MigrateStore(ctx.WithLogger(log.NewNopLogger()), paramsCollection)
		require.NoError(t, err)

		// Verify default params are set
		newParams, err := paramsCollection.Get(ctx)
		require.NoError(t, err)
		require.Equal(t, types.DefaultParams(), newParams)
	})
}
