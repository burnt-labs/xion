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
	ctx := testCtx.Ctx.WithLogger(log.NewNopLogger())

	storeService := runtime.NewKVStoreService(key)
	sb := collections.NewSchemaBuilder(storeService)
	paramsCollection := collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](encCfg.Codec))
	_, err := sb.Build()
	require.NoError(t, err)

	t.Run("backfills MinRsaKeyBits when zero", func(t *testing.T) {
		// Simulate v28 state: params exist but MinRsaKeyBits was never set.
		oldParams := types.Params{
			VkeyIdentifier:     1,
			MaxPubkeySizeBytes: 512,
			PublicInputIndices: types.DefaultPublicInputIndices(),
			MinRsaKeyBits:      0,
		}
		require.NoError(t, paramsCollection.Set(ctx, oldParams))

		require.NoError(t, v3.MigrateStore(ctx, paramsCollection))

		newParams, err := paramsCollection.Get(ctx)
		require.NoError(t, err)
		require.Equal(t, types.DefaultMinRSAKeyBits, newParams.MinRsaKeyBits)
		// Other fields must be preserved.
		require.Equal(t, uint64(1), newParams.VkeyIdentifier)
		require.Equal(t, uint64(512), newParams.MaxPubkeySizeBytes)
	})

	t.Run("does not overwrite MinRsaKeyBits when already set", func(t *testing.T) {
		existingParams := types.Params{
			VkeyIdentifier:     1,
			MaxPubkeySizeBytes: 512,
			PublicInputIndices: types.DefaultPublicInputIndices(),
			MinRsaKeyBits:      2048,
		}
		require.NoError(t, paramsCollection.Set(ctx, existingParams))

		require.NoError(t, v3.MigrateStore(ctx, paramsCollection))

		newParams, err := paramsCollection.Get(ctx)
		require.NoError(t, err)
		require.Equal(t, uint64(2048), newParams.MinRsaKeyBits)
	})

	t.Run("no-ops when params not found", func(t *testing.T) {
		require.NoError(t, paramsCollection.Remove(ctx))
		require.NoError(t, v3.MigrateStore(ctx, paramsCollection))
		// Params should still not exist.
		_, err := paramsCollection.Get(ctx)
		require.Error(t, err)
	})
}
