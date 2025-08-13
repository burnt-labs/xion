package v2

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

	"github.com/burnt-labs/xion/x/globalfee/types"
)

func TestMigrateStore(t *testing.T) {
	// Create test context
	storeKey := storetypes.NewKVStoreKey(paramstypes.StoreKey)
	tkey := storetypes.NewTransientStoreKey(paramstypes.TStoreKey)
	ctx := testutil.DefaultContextWithDB(t, storeKey, tkey)

	// Create subspace
	subspace := paramstypes.NewSubspace(
		codec.NewProtoCodec(codectypes.NewInterfaceRegistry()),
		codec.NewLegacyAmino(),
		storeKey,
		tkey,
		types.ModuleName,
	)

	// Test with subspace that doesn't have key table
	oldGlobalMinGasPrices := sdk.DecCoins{
		sdk.NewDecCoinFromDec("uxion", math.LegacyNewDecWithPrec(1, 3)),
	}

	// Set old params without key table
	subspaceWithKeyTable := subspace.WithKeyTable(types.ParamKeyTable())
	subspaceWithKeyTable.Set(ctx.Ctx, types.ParamStoreKeyMinGasPrices, oldGlobalMinGasPrices)

	// Remove key table to test migration with no key table
	subspaceNoKeyTable := paramstypes.NewSubspace(
		codec.NewProtoCodec(codectypes.NewInterfaceRegistry()),
		codec.NewLegacyAmino(),
		storeKey,
		tkey,
		types.ModuleName,
	)
	subspaceNoKeyTable.Set(ctx.Ctx, types.ParamStoreKeyMinGasPrices, oldGlobalMinGasPrices)

	// Test migration
	err := MigrateStore(ctx.Ctx, subspaceNoKeyTable)
	require.NoError(t, err)

	// Verify the migration worked
	var newParams types.Params
	subspaceWithKeyTable.GetParamSet(ctx.Ctx, &newParams)

	// Check that old minimum gas prices are preserved
	require.Equal(t, oldGlobalMinGasPrices, newParams.MinimumGasPrices)

	// Check that default values are set for new fields
	defaultParams := types.DefaultParams()
	require.Equal(t, defaultParams.BypassMinFeeMsgTypes, newParams.BypassMinFeeMsgTypes)
	require.Equal(t, defaultParams.MaxTotalBypassMinFeeMsgGasUsage, newParams.MaxTotalBypassMinFeeMsgGasUsage)
}

func TestMigrateStoreWithKeyTable(t *testing.T) {
	// Create test context
	storeKey := storetypes.NewKVStoreKey(paramstypes.StoreKey)
	tkey := storetypes.NewTransientStoreKey(paramstypes.TStoreKey)
	ctx := testutil.DefaultContextWithDB(t, storeKey, tkey)

	// Create subspace with key table
	subspace := paramstypes.NewSubspace(
		codec.NewProtoCodec(codectypes.NewInterfaceRegistry()),
		codec.NewLegacyAmino(),
		storeKey,
		tkey,
		types.ModuleName,
	).WithKeyTable(types.ParamKeyTable())

	oldGlobalMinGasPrices := sdk.DecCoins{
		sdk.NewDecCoinFromDec("stake", math.LegacyNewDecWithPrec(5, 4)),
	}

	// Set old params
	subspace.Set(ctx.Ctx, types.ParamStoreKeyMinGasPrices, oldGlobalMinGasPrices)

	// Test migration with existing key table
	err := MigrateStore(ctx.Ctx, subspace)
	require.NoError(t, err)

	// Verify the migration worked
	var newParams types.Params
	subspace.GetParamSet(ctx.Ctx, &newParams)

	// Check that old minimum gas prices are preserved
	require.Equal(t, oldGlobalMinGasPrices, newParams.MinimumGasPrices)

	// Check that default values are set for new fields
	defaultParams := types.DefaultParams()
	require.Equal(t, defaultParams.BypassMinFeeMsgTypes, newParams.BypassMinFeeMsgTypes)
	require.Equal(t, defaultParams.MaxTotalBypassMinFeeMsgGasUsage, newParams.MaxTotalBypassMinFeeMsgGasUsage)
}
