package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/burnt-labs/xion/x/globalfee/keeper"
	"github.com/burnt-labs/xion/x/globalfee/types"
)

func TestNewMigrator(t *testing.T) {
	// Create a test subspace
	storeKey := storetypes.NewKVStoreKey(paramstypes.StoreKey)
	tkey := storetypes.NewTransientStoreKey(paramstypes.TStoreKey)

	subspace := paramstypes.NewSubspace(
		codec.NewProtoCodec(codectypes.NewInterfaceRegistry()),
		codec.NewLegacyAmino(),
		storeKey,
		tkey,
		types.ModuleName,
	).WithKeyTable(types.ParamKeyTable())

	migrator := keeper.NewMigrator(subspace)
	require.NotNil(t, migrator)
}

func TestMigrate1to2(t *testing.T) {
	// Create a test subspace
	storeKey := storetypes.NewKVStoreKey(paramstypes.StoreKey)
	tkey := storetypes.NewTransientStoreKey(paramstypes.TStoreKey)
	ctx := testutil.DefaultContextWithDB(t, storeKey, tkey)

	subspace := paramstypes.NewSubspace(
		codec.NewProtoCodec(codectypes.NewInterfaceRegistry()),
		codec.NewLegacyAmino(),
		storeKey,
		tkey,
		types.ModuleName,
	).WithKeyTable(types.ParamKeyTable())

	// Set up initial params for migration
	initialParams := sdk.DecCoins{sdk.NewDecCoin("stake", math.NewInt(1000))}
	subspace.Set(ctx.Ctx, types.ParamStoreKeyMinGasPrices, &initialParams)

	migrator := keeper.NewMigrator(subspace)

	// Test that the migration function can be called
	err := migrator.Migrate1to2(ctx.Ctx)
	require.NoError(t, err)
}
