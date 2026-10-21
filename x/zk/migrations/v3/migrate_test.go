package v3_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
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

	t.Run("returns nil when params not found", func(t *testing.T) {
		require.NoError(t, v3.MigrateStore(ctx, paramsItem))
		_, err := paramsItem.Get(ctx)
		require.ErrorIs(t, err, collections.ErrNotFound)
	})

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

	t.Run("is idempotent when all params already set", func(t *testing.T) {
		params := types.DefaultParams()
		params.MaxVkeySizeBytes = 2000
		params.UploadChunkSize = 40
		params.UploadChunkGas = 20_000
		params.MaxGroth16ProofSizeBytes = 111
		params.MaxGroth16PublicInputSizeBytes = 222
		params.MaxUltraHonkProofSizeBytes = 333
		params.MaxUltraHonkPublicInputSizeBytes = 444
		require.NoError(t, paramsItem.Set(ctx, params))

		require.NoError(t, v3.MigrateStore(ctx, paramsItem))
		afterFirst, err := paramsItem.Get(ctx)
		require.NoError(t, err)
		require.Equal(t, params, afterFirst)

		require.NoError(t, v3.MigrateStore(ctx, paramsItem))
		afterSecond, err := paramsItem.Get(ctx)
		require.NoError(t, err)
		require.Equal(t, afterFirst, afterSecond)
	})

	t.Run("fills only missing limit fields", func(t *testing.T) {
		params := types.DefaultParams()
		params.MaxGroth16ProofSizeBytes = 0
		params.MaxGroth16PublicInputSizeBytes = 800
		params.MaxUltraHonkProofSizeBytes = 0
		params.MaxUltraHonkPublicInputSizeBytes = 900
		require.NoError(t, paramsItem.Set(ctx, params))

		require.NoError(t, v3.MigrateStore(ctx, paramsItem))
		got, err := paramsItem.Get(ctx)
		require.NoError(t, err)

		require.Equal(t, types.DefaultMaxGroth16ProofSizeBytes, got.MaxGroth16ProofSizeBytes)
		require.Equal(t, uint64(800), got.MaxGroth16PublicInputSizeBytes)
		require.Equal(t, types.DefaultMaxUltraHonkProofSizeBytes, got.MaxUltraHonkProofSizeBytes)
		require.Equal(t, uint64(900), got.MaxUltraHonkPublicInputSizeBytes)
		require.Equal(t, params.MaxVkeySizeBytes, got.MaxVkeySizeBytes)
		require.Equal(t, params.UploadChunkSize, got.UploadChunkSize)
		require.Equal(t, params.UploadChunkGas, got.UploadChunkGas)
	})
}

var errStoreUnavailable = errors.New("store unavailable")

// faultyStoreService serves a store whose reads or writes fail on demand.
type faultyStoreService struct {
	store.KVStoreService

	failGet bool
	failSet bool
}

func (s faultyStoreService) OpenKVStore(ctx context.Context) store.KVStore {
	return faultyStore{KVStore: s.KVStoreService.OpenKVStore(ctx), failGet: s.failGet, failSet: s.failSet}
}

type faultyStore struct {
	store.KVStore

	failGet bool
	failSet bool
}

func (s faultyStore) Get(key []byte) ([]byte, error) {
	if s.failGet {
		return nil, errStoreUnavailable
	}
	return s.KVStore.Get(key)
}

func (s faultyStore) Set(key, value []byte) error {
	if s.failSet {
		return errStoreUnavailable
	}
	return s.KVStore.Set(key, value)
}

func newParamsItem(t *testing.T, svc store.KVStoreService, cdc codec.Codec) collections.Item[types.Params] {
	t.Helper()
	sb := collections.NewSchemaBuilder(svc)
	item := collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc))
	_, err := sb.Build()
	require.NoError(t, err)
	return item
}

func TestMigrateStoreFailures(t *testing.T) {
	encCfg := moduletestutil.MakeTestEncodingConfig()
	types.RegisterInterfaces(encCfg.InterfaceRegistry)

	key := storetypes.NewKVStoreKey(types.ModuleName)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx
	storeService := runtime.NewKVStoreService(key)

	t.Run("params read fails", func(t *testing.T) {
		item := newParamsItem(t, faultyStoreService{KVStoreService: storeService, failGet: true}, encCfg.Codec)
		err := v3.MigrateStore(ctx, item)
		require.ErrorContains(t, err, errStoreUnavailable.Error())
	})

	t.Run("params write fails", func(t *testing.T) {
		// Seed pre-v3 params through a working store, then migrate through one
		// that refuses writes.
		seed := newParamsItem(t, storeService, encCfg.Codec)
		require.NoError(t, seed.Set(ctx, types.Params{MaxVkeySizeBytes: 1000, UploadChunkSize: 20, UploadChunkGas: 10_000}))

		item := newParamsItem(t, faultyStoreService{KVStoreService: storeService, failSet: true}, encCfg.Codec)
		err := v3.MigrateStore(ctx, item)
		require.ErrorContains(t, err, errStoreUnavailable.Error())
	})
}
