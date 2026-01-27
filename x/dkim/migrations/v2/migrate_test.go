package v2_test

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

	v2 "github.com/burnt-labs/xion/x/dkim/migrations/v2"
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
	dkimPubKeys := collections.NewMap(
		sb,
		types.DkimPrefix,
		"dkim_pubkeys",
		collections.PairKeyCodec(collections.StringKey, collections.StringKey),
		codec.CollValue[types.DkimPubKey](encCfg.Codec),
	)

	_, err := sb.Build()
	require.NoError(t, err)

	t.Run("migrate params with default PublicInputIndices", func(t *testing.T) {
		// Set params without PublicInputIndices (simulating v1 state)
		oldParams := types.Params{
			VkeyIdentifier:     1,
			MaxPubkeySizeBytes: 2048,
			// PublicInputIndices is nil (v1 state)
		}
		err := paramsCollection.Set(ctx, oldParams)
		require.NoError(t, err)

		// Run migration
		err = v2.MigrateStore(ctx.WithLogger(log.NewNopLogger()), paramsCollection, dkimPubKeys)
		require.NoError(t, err)

		// Verify params have PublicInputIndices
		newParams, err := paramsCollection.Get(ctx)
		require.NoError(t, err)
		require.NotNil(t, newParams.PublicInputIndices)
		require.Equal(t, types.DefaultPublicInputIndices().MinLength, newParams.PublicInputIndices.MinLength)
	})

	t.Run("migrate with no existing params sets defaults", func(t *testing.T) {
		// Clear params
		err := paramsCollection.Remove(ctx)
		require.NoError(t, err)

		// Run migration
		err = v2.MigrateStore(ctx.WithLogger(log.NewNopLogger()), paramsCollection, dkimPubKeys)
		require.NoError(t, err)

		// Verify default params are set
		newParams, err := paramsCollection.Get(ctx)
		require.NoError(t, err)
		require.NotNil(t, newParams.PublicInputIndices)
	})
}
