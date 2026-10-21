package jwk_test

import (
	"fmt"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	grpc1 "github.com/cosmos/gogoproto/grpc"

	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdkruntime "github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/burnt-labs/xion/x/jwk"
)

// Test module functions that need coverage
func TestModuleFunctionsCoverage(t *testing.T) {
	// Test RegisterGRPCGatewayRoutes by calling it
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	appModuleBasic := jwk.NewAppModuleBasic(cdc)

	// Create minimal client context and mux for testing
	clientCtx := client.Context{}.WithCodec(cdc)
	mux := runtime.NewServeMux()

	// This should not panic and will exercise the RegisterGRPCGatewayRoutes code
	require.NotPanics(t, func() {
		appModuleBasic.RegisterGRPCGatewayRoutes(clientCtx, mux)
	})
}

func TestModuleInterfaceMethods(t *testing.T) {
	// Test the interface methods that are currently at 0% coverage
	appModule := jwk.AppModule{}

	// Test IsOnePerModuleType
	require.NotPanics(t, func() {
		appModule.IsOnePerModuleType()
	})

	// Test IsAppModule
	require.NotPanics(t, func() {
		appModule.IsAppModule()
	})
}

func TestRegisterServices(t *testing.T) {
	// Create a properly initialized AppModule and configurator for full coverage
	k, _ := setupKeeperForGenesis(t)
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

	storeKey := storetypes.NewKVStoreKey("params")
	appModule := jwk.NewAppModule(cdc, k, sdkruntime.NewKVStoreService(storeKey), codec.NewLegacyAmino())

	// Test success case - mock configurator that returns no error
	mockCfgSuccess := &testConfigurator{
		msgServer:   &mockGrpcServer{},
		queryServer: &mockGrpcServer{},
		shouldError: false,
	}

	// This should execute the full RegisterServices function without panic
	require.NotPanics(t, func() {
		appModule.RegisterServices(mockCfgSuccess)
	})

	// Test error case - mock configurator that returns error to trigger panic
	mockCfgError := &testConfigurator{
		msgServer:   &mockGrpcServer{},
		queryServer: &mockGrpcServer{},
		shouldError: true,
	}

	// This should execute RegisterServices and panic on migration error for 100% coverage
	require.Panics(t, func() {
		appModule.RegisterServices(mockCfgError)
	})

	// The v3->v4 registration names its versions in the panic
	mockCfgSecondError := &testConfigurator{
		msgServer:       &mockGrpcServer{},
		queryServer:     &mockGrpcServer{},
		failFromVersion: 3,
	}
	require.PanicsWithValue(t, "failed to migrate x/jwk from v3 to v4: mock migration error from version 3", func() {
		appModule.RegisterServices(mockCfgSecondError)
	})

	// Also test the panic case with nil configurator for extra coverage
	appModuleEmpty := jwk.AppModule{}
	require.Panics(t, func() {
		appModuleEmpty.RegisterServices(nil)
	})
}

// Mock grpc server that implements grpc1.Server
type mockGrpcServer struct{}

func (m *mockGrpcServer) RegisterService(sd *grpc.ServiceDesc, ss interface{}) {
	// no-op for testing
}

func (m *mockGrpcServer) GetServiceInfo() map[string]grpc.ServiceInfo {
	return nil
}

// Mock configurator that implements module.Configurator interface
type testConfigurator struct {
	msgServer   grpc1.Server
	queryServer grpc1.Server
	shouldError bool
	// failFromVersion rejects only the migration registered from this version
	failFromVersion uint64
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
	if tc.failFromVersion != 0 && fromVersion == tc.failFromVersion {
		return fmt.Errorf("mock migration error from version %d", fromVersion)
	}
	return nil // Success case to avoid panic
}

func (tc *testConfigurator) Error() error {
	return nil
}
