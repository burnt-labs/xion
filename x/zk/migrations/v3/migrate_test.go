package v3_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/collections"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"

	v3 "github.com/burnt-labs/xion/x/zk/migrations/v3"
	"github.com/burnt-labs/xion/x/zk/types"
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

	paramsItem := collections.NewItem(
		sb,
		types.ParamsKey,
		"params",
		codec.CollValue[types.Params](encCfg.Codec),
	)

	_, err := sb.Build()
	require.NoError(t, err)

	t.Run("backfills groth16 param defaults", func(t *testing.T) {
		// Simulate pre-v3 persisted params (new groth16 fields unset -> zero values).
		oldParams := types.Params{
			MaxVkeySizeBytes: 1000,
			UploadChunkSize:  20,
			UploadChunkGas:   10_000,
		}
		require.NoError(t, paramsItem.Set(ctx, oldParams))

		// Perform migration.
		require.NoError(t, v3.MigrateStore(ctx, paramsItem))

		// Verify.
		got, err := paramsItem.Get(ctx)
		require.NoError(t, err)
		require.Equal(t, types.DefaultMaxGroth16ProofSizeBytes, got.MaxGroth16ProofSizeBytes)
		require.Equal(t, types.DefaultMaxGroth16PublicInputSizeBytes, got.MaxGroth16PublicInputSizeBytes)
		require.Equal(t, types.DefaultMaxUltraHonkProofSizeBytes, got.MaxUltraHonkProofSizeBytes)
		require.Equal(t, types.DefaultMaxUltraHonkPublicInputSizeBytes, got.MaxUltraHonkPublicInputSizeBytes)
		require.Equal(t, oldParams.MaxVkeySizeBytes, got.MaxVkeySizeBytes)
		require.Equal(t, oldParams.UploadChunkSize, got.UploadChunkSize)
		require.Equal(t, oldParams.UploadChunkGas, got.UploadChunkGas)
	})
}
