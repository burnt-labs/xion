package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/globalfee/keeper"
	"github.com/burnt-labs/xion/x/globalfee/types"
)

func TestMigrate2to3(t *testing.T) {
	params := types.DefaultParams()
	var err error
	params.MinimumGasPrices, err = sdk.ParseDecCoins("0.005ibc/F082B65C88E4B6D5EF1DB243CDA1D331D002759E938A0F5CD3FFDC5D53B3E349,0.001uxion")
	require.NoError(t, err)
	params.BypassMinFeeMsgTypes = []string{"/cosmos.bank.v1beta1.MsgSend"}
	params.MaxTotalBypassMinFeeMsgGasUsage = 123456

	for _, malformed := range []bool{false, true} {
		name := "legacy values"
		if malformed {
			name = "malformed legacy value"
		}
		t.Run(name, func(t *testing.T) {
			key := storetypes.NewKVStoreKey(types.StoreKey)
			legacyKey := storetypes.NewKVStoreKey("params")
			ctx := testutil.DefaultContextWithKeys(map[string]*storetypes.KVStoreKey{types.StoreKey: key, "params": legacyKey}, nil, nil)
			k := keeper.NewKeeper(codec.NewProtoCodec(codectypes.NewInterfaceRegistry()), runtime.NewKVStoreService(key), "authority")
			legacy := runtime.NewKVStoreService(legacyKey)
			amino := codec.NewLegacyAmino()
			store := legacy.OpenKVStore(ctx)
			for _, field := range []struct {
				key   string
				value interface{}
			}{
				{"globalfee/MinimumGasPricesParam", params.MinimumGasPrices},
				{"globalfee/BypassMinFeeMsgTypes", params.BypassMinFeeMsgTypes},
				{"globalfee/MaxTotalBypassMinFeeMsgGasUsage", params.MaxTotalBypassMinFeeMsgGasUsage},
			} {
				bz, err := amino.MarshalJSON(field.value)
				require.NoError(t, err)
				require.NoError(t, store.Set([]byte(field.key), bz))
			}
			want := params
			if malformed {
				require.NoError(t, store.Set([]byte("globalfee/MinimumGasPricesParam"), []byte("not json")))
				want = types.DefaultParams()
				want.MaxTotalBypassMinFeeMsgGasUsage = 42
				require.NoError(t, k.SetParams(ctx, want))
			}

			err := keeper.NewMigrator(k, legacy, amino).Migrate2to3(ctx)
			if malformed {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			got, err := k.GetParams(ctx)
			require.NoError(t, err)
			require.True(t, want.MinimumGasPrices.Equal(got.MinimumGasPrices))
			require.Equal(t, want.BypassMinFeeMsgTypes, got.BypassMinFeeMsgTypes)
			require.Equal(t, want.MaxTotalBypassMinFeeMsgGasUsage, got.MaxTotalBypassMinFeeMsgGasUsage)
		})
	}
}
