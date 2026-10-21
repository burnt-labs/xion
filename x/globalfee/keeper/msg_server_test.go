package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	"github.com/cosmos/cosmos-sdk/testutil"

	"github.com/burnt-labs/xion/x/globalfee/keeper"
	"github.com/burnt-labs/xion/x/globalfee/types"
)

func TestUpdateParams(t *testing.T) {
	key := storetypes.NewKVStoreKey(types.StoreKey)
	tkey := storetypes.NewTransientStoreKey("transient")
	ctx := testutil.DefaultContextWithDB(t, key, tkey).Ctx
	k := keeper.NewKeeper(codec.NewProtoCodec(codectypes.NewInterfaceRegistry()), runtime.NewKVStoreService(key), "authority")
	server := keeper.NewMsgServerImpl(k)

	_, err := server.UpdateParams(ctx, &types.MsgUpdateParams{Authority: "other", Params: types.DefaultParams()})
	require.ErrorIs(t, err, types.ErrInvalidAuthority)

	invalid := types.DefaultParams()
	invalid.BypassMinFeeMsgTypes = []string{""}
	_, err = server.UpdateParams(ctx, &types.MsgUpdateParams{Authority: k.GetAuthority(), Params: invalid})
	require.Error(t, err)
	_, err = k.Params.Get(ctx)
	require.Error(t, err)

	params := types.DefaultParams()
	params.MaxTotalBypassMinFeeMsgGasUsage = 1234
	_, err = server.UpdateParams(ctx, &types.MsgUpdateParams{Authority: k.GetAuthority(), Params: params})
	require.NoError(t, err)
	stored, err := k.GetParams(ctx)
	require.NoError(t, err)
	require.True(t, params.MinimumGasPrices.Equal(stored.MinimumGasPrices))
	require.Equal(t, params.BypassMinFeeMsgTypes, stored.BypassMinFeeMsgTypes)
	require.Equal(t, params.MaxTotalBypassMinFeeMsgGasUsage, stored.MaxTotalBypassMinFeeMsgGasUsage)
}

func TestParamsDefaultsAndQuery(t *testing.T) {
	key := storetypes.NewKVStoreKey(types.StoreKey)
	tkey := storetypes.NewTransientStoreKey("transient")
	ctx := testutil.DefaultContextWithDB(t, key, tkey).Ctx
	k := keeper.NewKeeper(codec.NewProtoCodec(codectypes.NewInterfaceRegistry()), runtime.NewKVStoreService(key), "authority")
	params, err := k.GetParams(ctx)
	require.NoError(t, err)
	require.Equal(t, types.DefaultParams(), params)

	q := keeper.NewQuerier(k)
	_, err = q.Params(ctx, nil)
	require.EqualError(t, err, "invalid request")
	response, err := q.Params(ctx, &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, params, response.Params)
}

func TestParamsReadError(t *testing.T) {
	key := storetypes.NewKVStoreKey(types.StoreKey)
	tkey := storetypes.NewTransientStoreKey("transient")
	ctx := testutil.DefaultContextWithDB(t, key, tkey).Ctx
	service := runtime.NewKVStoreService(key)
	k := keeper.NewKeeper(codec.NewProtoCodec(codectypes.NewInterfaceRegistry()), service, "authority")
	require.NoError(t, service.OpenKVStore(ctx).Set(types.ParamsKey.Bytes(), []byte{0xff}))
	_, err := k.GetParams(ctx)
	require.Error(t, err)
	_, err = keeper.NewQuerier(k).Params(ctx, &types.QueryParamsRequest{})
	require.Error(t, err)
}
