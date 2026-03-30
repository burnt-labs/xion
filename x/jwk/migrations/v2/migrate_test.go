package v2_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"

	v2migration "github.com/burnt-labs/xion/x/jwk/migrations/v2"
	"github.com/burnt-labs/xion/x/jwk/types"
)

func TestMigrateStore(t *testing.T) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	tkey := storetypes.NewTransientStoreKey(paramstypes.TStoreKey)
	ctx := testutil.DefaultContextWithDB(t, storeKey, tkey)
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

	paramStore := paramstypes.NewSubspace(
		cdc,
		codec.NewLegacyAmino(),
		storeKey,
		tkey,
		types.ModuleName,
	).WithKeyTable(types.ParamKeyTable())

	// Simulate chain that has DeploymentGas set but no TimeOffset.
	paramStore.Set(ctx.Ctx, types.ParamStoreKeyDeploymentGas, uint64(10_000))

	err := v2migration.MigrateStore(ctx.Ctx, paramStore)
	require.NoError(t, err)

	// TimeOffset should now be set to 30 seconds in nanoseconds.
	var timeOffset uint64
	paramStore.Get(ctx.Ctx, types.ParamStoreKeyTimeOffset, &timeOffset)
	require.Equal(t, uint64(30_000_000_000), timeOffset)
}

func TestMigrateStoreSkipsExistingTimeOffset(t *testing.T) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	tkey := storetypes.NewTransientStoreKey(paramstypes.TStoreKey)
	ctx := testutil.DefaultContextWithDB(t, storeKey, tkey)
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

	paramStore := paramstypes.NewSubspace(
		cdc,
		codec.NewLegacyAmino(),
		storeKey,
		tkey,
		types.ModuleName,
	).WithKeyTable(types.ParamKeyTable())

	// Set both params, including a non-zero TimeOffset.
	existingOffset := uint64(60_000_000_000)
	paramStore.Set(ctx.Ctx, types.ParamStoreKeyDeploymentGas, uint64(10_000))
	paramStore.Set(ctx.Ctx, types.ParamStoreKeyTimeOffset, existingOffset)

	err := v2migration.MigrateStore(ctx.Ctx, paramStore)
	require.NoError(t, err)

	// TimeOffset should remain unchanged.
	var timeOffset uint64
	paramStore.Get(ctx.Ctx, types.ParamStoreKeyTimeOffset, &timeOffset)
	require.Equal(t, existingOffset, timeOffset)
}
