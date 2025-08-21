package globalfee_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/burnt-labs/xion/x/globalfee"
	"github.com/burnt-labs/xion/x/globalfee/types"
)

func TestNewGrpcQuerier(t *testing.T) {
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

	querier := globalfee.NewGrpcQuerier(subspace)
	require.NotNil(t, querier)
}

func TestGrpcQuerierParams(t *testing.T) {
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

	// Set default params
	params := types.DefaultParams()
	subspace.SetParamSet(ctx.Ctx, &params)

	querier := globalfee.NewGrpcQuerier(subspace)

	// Test Params query
	req := &types.QueryParamsRequest{}
	resp, err := querier.Params(ctx.Ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Compare individual fields - MinimumGasPrices can be either empty slice or nil
	require.True(t, (len(params.MinimumGasPrices) == 0 && len(resp.Params.MinimumGasPrices) == 0),
		"MinimumGasPrices should both be empty")
	require.Equal(t, params.BypassMinFeeMsgTypes, resp.Params.BypassMinFeeMsgTypes)
	require.Equal(t, params.MaxTotalBypassMinFeeMsgGasUsage, resp.Params.MaxTotalBypassMinFeeMsgGasUsage)

	// Test Params query with nil request
	resp, err = querier.Params(ctx.Ctx, nil)
	require.Error(t, err)
	require.Nil(t, resp)
}
