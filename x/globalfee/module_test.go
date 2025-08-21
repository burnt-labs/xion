package globalfee_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	grpc1 "github.com/cosmos/gogoproto/grpc"

	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	"github.com/cosmos/cosmos-sdk/types/module"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/burnt-labs/xion/x/globalfee"
	"github.com/burnt-labs/xion/x/globalfee/types"
)

// Mock configurator that implements module.Configurator interface
type testConfigurator struct {
	msgServer   grpc1.Server
	queryServer grpc1.Server
	shouldError bool
}

func (tc *testConfigurator) RegisterService(sd *grpc.ServiceDesc, ss interface{}) {
	// no-op for testing
}

func (tc *testConfigurator) GetServiceInfo() map[string]grpc.ServiceInfo {
	return nil
}

func (tc *testConfigurator) MsgServer() grpc1.Server {
	return tc.msgServer
}

func (tc *testConfigurator) QueryServer() grpc1.Server {
	return tc.queryServer
}

func (tc *testConfigurator) RegisterMigration(moduleName string, fromVersion uint64, handler module.MigrationHandler) error {
	if tc.shouldError {
		return fmt.Errorf("mock migration error for testing")
	}
	return nil // Success case to avoid panic
}

func (tc *testConfigurator) Error() error {
	return nil
}

// mockGrpcServer implements grpc1.Server interface for testing
type mockGrpcServer struct{}

func (m *mockGrpcServer) RegisterService(sd *grpc.ServiceDesc, ss interface{}) {}
func (m *mockGrpcServer) GetServiceInfo() map[string]grpc.ServiceInfo          { return nil }

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
	mux := runtime.NewServeMux()

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
	globalfee.RegisterQueryHandlerClientFn = func(_ context.Context, _ *runtime.ServeMux, _ types.QueryClient) error {
		return fmt.Errorf("forced error")
	}
	require.Panics(t, func() {
		appModule.RegisterGRPCGatewayRoutes(clientCtx, mux)
	})

	// Note: The panic-on-error path is now tested by injecting a failing implementation
	// via the exported RegisterQueryHandlerClientFn hook.
}

func TestAppModule(t *testing.T) {
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
	)

	// Test NewAppModule with subspace that has key table (should NOT trigger WithKeyTable path)
	subspaceWithKeyTable := subspace.WithKeyTable(types.ParamKeyTable())
	appModule := globalfee.NewAppModule(subspaceWithKeyTable)
	require.NotNil(t, appModule)

	// Test NewAppModule with subspace that doesn't have key table (should trigger WithKeyTable path)
	appModule2 := globalfee.NewAppModule(subspace)
	require.NotNil(t, appModule2)

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
	require.Equal(t, uint64(2), version)

	// Test InitGenesis
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
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
}

func TestNewAppModuleBothPaths(t *testing.T) {
	// Create test store keys
	storeKey := storetypes.NewKVStoreKey(paramstypes.StoreKey)
	tkey := storetypes.NewTransientStoreKey(paramstypes.TStoreKey)

	// Test path 1: subspace without key table (should call WithKeyTable)
	subspace1 := paramstypes.NewSubspace(
		codec.NewProtoCodec(codectypes.NewInterfaceRegistry()),
		codec.NewLegacyAmino(),
		storeKey,
		tkey,
		types.ModuleName,
	)
	require.False(t, subspace1.HasKeyTable(), "Test setup: subspace should not have key table initially")
	appModule1 := globalfee.NewAppModule(subspace1)
	require.NotNil(t, appModule1)

	// Test path 2: subspace with key table (should NOT call WithKeyTable)
	subspace2 := paramstypes.NewSubspace(
		codec.NewProtoCodec(codectypes.NewInterfaceRegistry()),
		codec.NewLegacyAmino(),
		storeKey,
		tkey,
		types.ModuleName+"2", // Different module name to avoid conflicts
	)
	subspaceWithKeyTable := subspace2.WithKeyTable(types.ParamKeyTable())
	require.True(t, subspaceWithKeyTable.HasKeyTable(), "Test setup: subspace should have key table")
	appModule2 := globalfee.NewAppModule(subspaceWithKeyTable)
	require.NotNil(t, appModule2)
}

func TestAppModuleRegisterServices(t *testing.T) {
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

	appModule := globalfee.NewAppModule(subspace)

	// Test that RegisterServices doesn't panic when called
	// We can't easily test the actual registration without more complex mocking
	// but we can test that the function exists and doesn't panic with nil input
	// Expect a panic when passing nil configurator; verify panic occurs (previously empty branch)
	require.Panics(t, func() {
		appModule.RegisterServices(nil)
	}, "expected panic when calling RegisterServices with nil configurator")

	// Test panic path in RegisterServices when migration registration fails
	mockCfgError := &testConfigurator{
		msgServer:   &mockGrpcServer{},
		queryServer: &mockGrpcServer{},
		shouldError: true,
	}
	require.Panics(t, func() {
		appModule.RegisterServices(mockCfgError)
	})
}

func TestAppModuleBasicNoOpMethods(t *testing.T) {
	appModuleBasic := globalfee.AppModuleBasic{}

	// Test RegisterInterfaces - should not panic
	require.NotPanics(t, func() {
		appModuleBasic.RegisterInterfaces(nil)
	})

	// Test RegisterRESTRoutes - should not panic
	require.NotPanics(t, func() {
		appModuleBasic.RegisterRESTRoutes(client.Context{}, nil)
	})

	// Test RegisterLegacyAminoCodec - should not panic
	require.NotPanics(t, func() {
		appModuleBasic.RegisterLegacyAminoCodec(nil)
	})

	// Test RegisterInterfaces - should not panic
	require.NotPanics(t, func() {
		appModuleBasic.RegisterInterfaces(nil)
	})

	// Test RegisterRESTRoutes - should not panic
	require.NotPanics(t, func() {
		appModuleBasic.RegisterRESTRoutes(client.Context{}, nil)
	})
}

func TestAppModuleNoOpMethods(t *testing.T) {
	appModule := globalfee.AppModule{}

	// Test IsOnePerModuleType - should not panic
	require.NotPanics(t, func() {
		appModule.IsOnePerModuleType()
	})

	// Test IsAppModule - should not panic
	require.NotPanics(t, func() {
		appModule.IsAppModule()
	})
}
