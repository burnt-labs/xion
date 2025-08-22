package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/burnt-labs/xion/x/jwk/keeper"
	"github.com/burnt-labs/xion/x/jwk/types"
)

func setupKeeper(t testing.TB) (keeper.Keeper, sdk.Context) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	tkey := storetypes.NewTransientStoreKey(paramstypes.TStoreKey)

	ctx := testutil.DefaultContextWithDB(t, storeKey, tkey)

	// Create codec
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

	// Create param subspace
	paramStore := paramstypes.NewSubspace(
		cdc,
		codec.NewLegacyAmino(),
		storeKey,
		tkey,
		types.ModuleName,
	)

	k := keeper.NewKeeper(
		cdc,
		storeKey,
		paramStore,
	)

	// Initialize with default params
	k.SetParams(ctx.Ctx, types.DefaultParams())

	return k, ctx.Ctx
}

func TestNewKeeper(t *testing.T) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	tkey := storetypes.NewTransientStoreKey(paramstypes.TStoreKey)
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

	// Test with param subspace that doesn't have key table
	paramStore := paramstypes.NewSubspace(
		cdc,
		codec.NewLegacyAmino(),
		storeKey,
		tkey,
		types.ModuleName,
	)

	k := keeper.NewKeeper(cdc, storeKey, paramStore)
	require.NotNil(t, k)

	// Test with fresh param subspace for key table test
	freshParamStore := paramstypes.NewSubspace(
		cdc,
		codec.NewLegacyAmino(),
		storeKey,
		tkey,
		"fresh_module",
	)
	paramStoreWithKeyTable := freshParamStore.WithKeyTable(types.ParamKeyTable())
	k2 := keeper.NewKeeper(cdc, storeKey, paramStoreWithKeyTable)
	require.NotNil(t, k2)
}

func TestKeeperLogger(t *testing.T) {
	k, ctx := setupKeeper(t)

	logger := k.Logger(ctx)
	require.NotNil(t, logger)

	// Logger should be of type log.Logger
	require.Implements(t, (*log.Logger)(nil), logger)
}

func TestKeeperParams(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Test GetParams
	params := k.GetParams(ctx)
	require.NotNil(t, params)
	require.Equal(t, types.DefaultParams(), params)

	// Test SetParams with custom values
	customParams := types.NewParams(1500, 1000000)
	k.SetParams(ctx, customParams)

	// Verify params were set
	retrievedParams := k.GetParams(ctx)
	require.Equal(t, customParams, retrievedParams)
	require.Equal(t, uint64(1000000), retrievedParams.DeploymentGas)
	require.Equal(t, uint64(1500), retrievedParams.TimeOffset)

	// Test GetTimeOffset
	timeOffset := k.GetTimeOffset(ctx)
	require.Equal(t, uint64(1500), timeOffset)

	// Test GetDeploymentGas
	deploymentGas := k.GetDeploymentGas(ctx)
	require.Equal(t, uint64(1000000), deploymentGas)
}
