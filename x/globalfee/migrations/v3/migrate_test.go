package v3_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/globalfee/keeper"
	v3 "github.com/burnt-labs/xion/x/globalfee/migrations/v3"
	"github.com/burnt-labs/xion/x/globalfee/types"
)

func TestMigrateStore(t *testing.T) {
	mainnet := types.DefaultParams()
	var err error
	mainnet.MinimumGasPrices, err = sdk.ParseDecCoins("0.005ibc/F082B65C88E4B6D5EF1DB243CDA1D331D002759E938A0F5CD3FFDC5D53B3E349,0.001uxion")
	require.NoError(t, err)
	for _, tc := range []struct {
		name string
		seed bool
		want types.Params
	}{
		{"mainnet", true, mainnet},
		{"missing keys", false, types.DefaultParams()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			key := storetypes.NewKVStoreKey(types.StoreKey)
			legacyKey := storetypes.NewKVStoreKey("params")
			ctx := testutil.DefaultContextWithKeys(map[string]*storetypes.KVStoreKey{types.StoreKey: key, "params": legacyKey}, nil, nil)
			k := keeper.NewKeeper(codec.NewProtoCodec(codectypes.NewInterfaceRegistry()), runtime.NewKVStoreService(key), "authority")
			legacy := runtime.NewKVStoreService(legacyKey)
			amino := codec.NewLegacyAmino()
			if tc.seed {
				store := legacy.OpenKVStore(ctx)
				for _, field := range []struct {
					key   string
					value interface{}
				}{
					{"globalfee/MinimumGasPricesParam", mainnet.MinimumGasPrices},
					{"globalfee/BypassMinFeeMsgTypes", mainnet.BypassMinFeeMsgTypes},
				} {
					bz, err := amino.MarshalJSON(field.value)
					require.NoError(t, err)
					require.NoError(t, store.Set([]byte(field.key), bz))
				}
				require.NoError(t, store.Set([]byte("globalfee/MaxTotalBypassMinFeeMsgGasUsage"), []byte(`"1000000"`)))
			}

			require.NoError(t, v3.MigrateStore(ctx, k, legacy, amino))
			got, err := k.GetParams(ctx)
			require.NoError(t, err)
			require.True(t, tc.want.MinimumGasPrices.Equal(got.MinimumGasPrices))
			require.Equal(t, tc.want.BypassMinFeeMsgTypes, got.BypassMinFeeMsgTypes)
			require.Equal(t, tc.want.MaxTotalBypassMinFeeMsgGasUsage, got.MaxTotalBypassMinFeeMsgGasUsage)
		})
	}
}

func TestMigrateStoreRejectsInvalidLegacyParams(t *testing.T) {
	for _, value := range []string{`not json`, `[""]`} {
		t.Run(value, func(t *testing.T) {
			key := storetypes.NewKVStoreKey(types.StoreKey)
			legacyKey := storetypes.NewKVStoreKey("params")
			ctx := testutil.DefaultContextWithKeys(map[string]*storetypes.KVStoreKey{types.StoreKey: key, "params": legacyKey}, nil, nil)
			k := keeper.NewKeeper(codec.NewProtoCodec(codectypes.NewInterfaceRegistry()), runtime.NewKVStoreService(key), "authority")
			legacy := runtime.NewKVStoreService(legacyKey)
			require.NoError(t, legacy.OpenKVStore(ctx).Set([]byte("globalfee/BypassMinFeeMsgTypes"), []byte(value)))

			require.Error(t, v3.MigrateStore(ctx, k, legacy, codec.NewLegacyAmino()))
			_, err := k.Params.Get(ctx)
			require.Error(t, err)
		})
	}
}
