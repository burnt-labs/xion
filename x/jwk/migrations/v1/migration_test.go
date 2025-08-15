package v1_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"

	v1migration "github.com/burnt-labs/xion/x/jwk/migrations/v1"
	"github.com/burnt-labs/xion/x/jwk/types"
)

func TestMigrateStore(t *testing.T) {
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
	// Don't call WithKeyTable here to test the HasKeyTable() branch

	// Test MigrateStore - this should exercise the key table creation branch
	err := v1migration.MigrateStore(ctx.Ctx, paramStore)
	require.NoError(t, err)

	// Verify params were set correctly
	// After MigrateStore, the paramStore now has a key table, so we can use it directly
	var params types.Params
	paramStore.GetParamSet(ctx.Ctx, &params)
	require.Equal(t, uint64(10_000), params.DeploymentGas)
	require.Equal(t, uint64(30_000), params.TimeOffset)

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

	// Test MigrateStore with key table already present
	err = v1migration.MigrateStore(ctx2.Ctx, paramStoreWithKeyTable)
	require.NoError(t, err)

	// Verify params were set correctly
	var params2 types.Params
	paramStoreWithKeyTable.GetParamSet(ctx2.Ctx, &params2)
	require.Equal(t, uint64(10_000), params2.DeploymentGas)
	require.Equal(t, uint64(30_000), params2.TimeOffset)
}
