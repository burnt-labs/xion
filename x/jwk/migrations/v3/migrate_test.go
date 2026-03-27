package v3_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"

	v3migration "github.com/burnt-labs/xion/x/jwk/migrations/v3"
	"github.com/burnt-labs/xion/x/jwk/types"
)

func TestMigrateStore(t *testing.T) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	tkey := storetypes.NewTransientStoreKey(paramstypes.TStoreKey)
	ctx := testutil.DefaultContextWithDB(t, storeKey, tkey)
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

	// Simulate the broken state: subspace with key table, TimeOffset set to the
	// wrong value that Migrate1To2 wrote (30_000 instead of 30_000_000_000).
	paramStore := paramstypes.NewSubspace(
		cdc,
		codec.NewLegacyAmino(),
		storeKey,
		tkey,
		types.ModuleName,
	).WithKeyTable(types.ParamKeyTable())

	brokenTimeOffset := uint64(30_000)
	paramStore.Set(ctx.Ctx, types.ParamStoreKeyTimeOffset, brokenTimeOffset)

	var beforeParams types.Params
	paramStore.GetParamSet(ctx.Ctx, &beforeParams)
	require.Equal(t, brokenTimeOffset, beforeParams.TimeOffset, "pre-condition: TimeOffset should be the broken value")

	// Run the migration.
	err := v3migration.MigrateStore(ctx.Ctx, paramStore)
	require.NoError(t, err)

	// Verify TimeOffset was corrected.
	var afterParams types.Params
	paramStore.GetParamSet(ctx.Ctx, &afterParams)
	require.Equal(t, uint64(30_000_000_000), afterParams.TimeOffset, "TimeOffset should be 30 seconds in nanoseconds after migration")
}

func TestMigrateStoreWithoutKeyTable(t *testing.T) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	tkey := storetypes.NewTransientStoreKey(paramstypes.TStoreKey)
	ctx := testutil.DefaultContextWithDB(t, storeKey, tkey)
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

	// Create subspace WITHOUT key table to exercise that branch.
	paramStore := paramstypes.NewSubspace(
		cdc,
		codec.NewLegacyAmino(),
		storeKey,
		tkey,
		types.ModuleName,
	)

	err := v3migration.MigrateStore(ctx.Ctx, paramStore)
	require.NoError(t, err)
}
