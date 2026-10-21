package globalfee_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/burnt-labs/xion/x/globalfee"
	"github.com/burnt-labs/xion/x/globalfee/keeper"
	"github.com/burnt-labs/xion/x/globalfee/types"
)

func TestRegisterServices(t *testing.T) {
	for _, tc := range []struct {
		name               string
		duplicateMigration bool
	}{
		{name: "services and migration"},
		{name: "duplicate migration", duplicateMigration: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			key := storetypes.NewKVStoreKey(types.StoreKey)
			legacyKey := storetypes.NewKVStoreKey("params")
			ctx := testutil.DefaultContextWithKeys(map[string]*storetypes.KVStoreKey{types.StoreKey: key, "params": legacyKey}, nil, nil)
			registry := codectypes.NewInterfaceRegistry()
			globalfee.AppModuleBasic{}.RegisterInterfaces(registry)
			cdc := codec.NewProtoCodec(registry)
			k := keeper.NewKeeper(cdc, runtime.NewKVStoreService(key), "authority")
			legacy := runtime.NewKVStoreService(legacyKey)
			amino := codec.NewLegacyAmino()
			appModule := globalfee.NewAppModule(cdc, k, legacy, amino)
			msgRouter := baseapp.NewMsgServiceRouter()
			msgRouter.SetInterfaceRegistry(registry)
			queryRouter := baseapp.NewGRPCQueryRouter()
			queryRouter.SetInterfaceRegistry(registry)
			cfg := module.NewConfigurator(cdc, msgRouter, queryRouter)
			if tc.duplicateMigration {
				err := cfg.RegisterMigration(types.ModuleName, 2, keeper.NewMigrator(k, legacy, amino).Migrate2to3)
				require.NoError(t, err)
				require.Panics(t, func() { appModule.RegisterServices(cfg) })
				return
			}

			appModule.RegisterServices(cfg)
			require.NoError(t, cfg.Error())
			msgHandler := msgRouter.HandlerByTypeURL(sdk.MsgTypeURL(&types.MsgUpdateParams{}))
			require.NotNil(t, msgHandler)
			queryHandler := queryRouter.Route("/xion.globalfee.v1.Query/Params")
			require.NotNil(t, queryHandler)

			params := types.DefaultParams()
			var err error
			params.MinimumGasPrices, err = sdk.ParseDecCoins("0.002uxion")
			require.NoError(t, err)
			params.BypassMinFeeMsgTypes = []string{"/cosmos.bank.v1beta1.MsgSend"}
			params.MaxTotalBypassMinFeeMsgGasUsage = 123456
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
				require.NoError(t, legacy.OpenKVStore(ctx).Set([]byte(field.key), bz))
			}
			versions, err := module.NewManager(appModule).RunMigrations(ctx, cfg, module.VersionMap{types.ModuleName: 2})
			require.NoError(t, err)
			require.Equal(t, module.VersionMap{types.ModuleName: 3}, versions)
			migrated, err := k.GetParams(ctx)
			require.NoError(t, err)
			require.Equal(t, params, migrated)

			_, err = msgHandler(ctx, &types.MsgUpdateParams{Authority: "other", Params: params})
			require.ErrorIs(t, err, types.ErrInvalidAuthority)
			params.MaxTotalBypassMinFeeMsgGasUsage = 654321
			_, err = msgHandler(ctx, &types.MsgUpdateParams{Authority: k.GetAuthority(), Params: params})
			require.NoError(t, err)

			response, err := queryHandler(ctx, &abci.RequestQuery{Data: cdc.MustMarshal(&types.QueryParamsRequest{})})
			require.NoError(t, err)
			var decoded types.QueryParamsResponse
			require.NoError(t, cdc.Unmarshal(response.Value, &decoded))
			require.Equal(t, params, decoded.Params)
		})
	}
}
