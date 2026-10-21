package globalfee_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/gorilla/mux"
	gwruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/gogoproto/proto"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	"github.com/cosmos/cosmos-sdk/testutil"

	"github.com/burnt-labs/xion/x/globalfee"
	"github.com/burnt-labs/xion/x/globalfee/keeper"
	"github.com/burnt-labs/xion/x/globalfee/types"
)

func TestAppModuleBasic(t *testing.T) {
	appModule := globalfee.AppModuleBasic{}

	// Test Name
	require.Equal(t, types.ModuleName, appModule.Name())

	// Test DefaultGenesis
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	genesis := appModule.DefaultGenesis(cdc)
	require.NotNil(t, genesis)

	var genesisState types.GenesisState
	err := cdc.UnmarshalJSON(genesis, &genesisState)
	require.NoError(t, err)
	require.Equal(t, types.DefaultParams(), genesisState.Params)

	// Test ValidateGenesis with valid data
	var txConfig client.TxEncodingConfig
	err = appModule.ValidateGenesis(cdc, txConfig, genesis)
	require.NoError(t, err)

	// Test ValidateGenesis with invalid JSON
	invalidJSON := []byte(`{"invalid": "json"}`)
	err = appModule.ValidateGenesis(cdc, txConfig, invalidJSON)
	require.Error(t, err)

	// Test ValidateGenesis with invalid params
	// Test with invalid JSON structure to trigger validation error
	invalidGenesis := []byte(`{"params":{"minimum_gas_prices":[{"denom":"","amount":"1"}]}}`)
	err = appModule.ValidateGenesis(cdc, txConfig, invalidGenesis)
	require.Error(t, err)

	// Test RegisterInterfaces (should not panic)
	registry := codectypes.NewInterfaceRegistry()
	require.NotPanics(t, func() {
		appModule.RegisterInterfaces(registry)
	})

	// Test RegisterRESTRoutes (should not panic)
	ctx := client.Context{}
	router := mux.NewRouter()
	require.NotPanics(t, func() {
		appModule.RegisterRESTRoutes(ctx, router)
	})

	// Test GetTxCmd
	txCmd := appModule.GetTxCmd()
	require.Nil(t, txCmd)

	// Test GetQueryCmd
	queryCmd := appModule.GetQueryCmd()
	require.NotNil(t, queryCmd)
	require.Equal(t, "globalfee", queryCmd.Use)

	// Test RegisterLegacyAminoCodec (should not panic)
	amino := codec.NewLegacyAmino()
	require.NotPanics(t, func() {
		appModule.RegisterLegacyAminoCodec(amino)
	})
}

func TestAppModuleBasicRegisterGRPCGatewayRoutes(t *testing.T) {
	appModule := globalfee.AppModuleBasic{}

	// Create a test client context
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(interfaceRegistry)
	clientCtx := client.Context{}.WithCodec(cdc)

	// Create a new ServeMux
	mux := gwruntime.NewServeMux()

	// This should not panic with proper client context
	require.NotPanics(t, func() {
		appModule.RegisterGRPCGatewayRoutes(clientCtx, mux)
	})

	// Test the panic path with nil ServeMux which should cause a panic
	require.Panics(t, func() {
		appModule.RegisterGRPCGatewayRoutes(clientCtx, nil)
	})

	// Force error path by overriding RegisterQueryHandlerClientFn
	orig := globalfee.RegisterQueryHandlerClientFn
	defer func() { globalfee.RegisterQueryHandlerClientFn = orig }()
	globalfee.RegisterQueryHandlerClientFn = func(_ context.Context, _ *gwruntime.ServeMux, _ types.QueryClient) error {
		return fmt.Errorf("forced error")
	}
	require.Panics(t, func() {
		appModule.RegisterGRPCGatewayRoutes(clientCtx, mux)
	})

	// Note: The panic-on-error path is now tested by injecting a failing implementation
	// via the exported RegisterQueryHandlerClientFn hook.
}

func TestAppModule(t *testing.T) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	tkey := storetypes.NewTransientStoreKey("transient")
	ctx := testutil.DefaultContextWithDB(t, storeKey, tkey)
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	k := keeper.NewKeeper(cdc, runtime.NewKVStoreService(storeKey), "authority")
	appModule := globalfee.NewAppModule(cdc, k, runtime.NewKVStoreService(storeKey), codec.NewLegacyAmino())

	// Test IsOnePerModuleType (should not panic)
	require.NotPanics(t, func() {
		appModule.IsOnePerModuleType()
	})

	// Test IsAppModule (should not panic)
	require.NotPanics(t, func() {
		appModule.IsAppModule()
	})

	// Test ConsensusVersion
	version := appModule.ConsensusVersion()
	require.Equal(t, uint64(3), version)

	// Test InitGenesis
	defaultGenesis := types.DefaultGenesisState()
	genesisData, err := cdc.MarshalJSON(defaultGenesis)
	require.NoError(t, err)

	validators := appModule.InitGenesis(ctx.Ctx, cdc, genesisData)
	require.Nil(t, validators)

	// Test ExportGenesis
	exportedGenesis := appModule.ExportGenesis(ctx.Ctx, cdc)
	require.NotNil(t, exportedGenesis)

	var exportedState types.GenesisState
	err = cdc.UnmarshalJSON(exportedGenesis, &exportedState)
	require.NoError(t, err)
	require.True(t, proto.Equal(defaultGenesis, &exportedState))
}
