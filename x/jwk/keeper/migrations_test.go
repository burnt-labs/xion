package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/burnt-labs/xion/x/jwk/keeper"
	v1migration "github.com/burnt-labs/xion/x/jwk/migrations/v1"
	"github.com/burnt-labs/xion/x/jwk/types"
)

func TestMigrations(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Test that we can get the current params (this verifies the keeper works)
	params := k.GetParams(ctx)
	require.NotNil(t, params)
	require.Equal(t, uint64(10_000), params.DeploymentGas)
	require.Equal(t, uint64(30_000), params.TimeOffset)

	// Test NewMigrator
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	tkey := storetypes.NewTransientStoreKey(paramstypes.TStoreKey)
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

	paramStore := paramstypes.NewSubspace(
		cdc,
		codec.NewLegacyAmino(),
		storeKey,
		tkey,
		types.ModuleName,
	).WithKeyTable(types.ParamKeyTable())

	migrator := keeper.NewMigrator(paramStore)
	require.NotNil(t, migrator)

	// Test Migrate1To2 with proper context
	dbCtx := testutil.DefaultContextWithDB(t, storeKey, tkey)
	err := migrator.Migrate1To2(dbCtx.Ctx)
	require.NoError(t, err)
}

func TestMigrateStore(t *testing.T) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	tkey := storetypes.NewTransientStoreKey(paramstypes.TStoreKey)
	dbCtx := testutil.DefaultContextWithDB(t, storeKey, tkey)

	// Create codec
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

	// Create param subspace
	paramStore := paramstypes.NewSubspace(
		cdc,
		codec.NewLegacyAmino(),
		storeKey,
		tkey,
		types.ModuleName,
	).WithKeyTable(types.ParamKeyTable())

	// Test the actual migration function from v1 package
	err := v1migration.MigrateStore(dbCtx.Ctx, paramStore)
	require.NoError(t, err)
}

func TestMigrationV1Functions(t *testing.T) {
	// Test the migration function exists and can be called
	require.NotNil(t, v1migration.MigrateStore)

	// Test the actual migration function with proper store setup
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	tkey := storetypes.NewTransientStoreKey(paramstypes.TStoreKey)
	ctx := testutil.DefaultContextWithDB(t, storeKey, tkey)
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

	// Test case 1: Create param subspace WITHOUT key table first
	paramStore := paramstypes.NewSubspace(
		cdc,
		codec.NewLegacyAmino(),
		storeKey,
		tkey,
		types.ModuleName,
	)
	// Don't call WithKeyTable here to test that branch in MigrateStore

	// Test MigrateStore - this should exercise the key table creation branch
	err := v1migration.MigrateStore(ctx.Ctx, paramStore)
	require.NoError(t, err)

	// Test case 2: Create a fresh param subspace with key table already set
	storeKey2 := storetypes.NewKVStoreKey(types.StoreKey + "2")
	tkey2 := storetypes.NewTransientStoreKey(paramstypes.TStoreKey + "2")
	ctx2 := testutil.DefaultContextWithDB(t, storeKey2, tkey2)

	paramStoreWithKeyTable := paramstypes.NewSubspace(
		cdc,
		codec.NewLegacyAmino(),
		storeKey2,
		tkey2,
		types.ModuleName,
	).WithKeyTable(types.ParamKeyTable())

	err = v1migration.MigrateStore(ctx2.Ctx, paramStoreWithKeyTable)
	require.NoError(t, err)
}
