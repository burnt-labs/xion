package xauthz_test

import (
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/burnt-labs/xion/x/xauthz"
	"github.com/burnt-labs/xion/x/xauthz/types"
)

func TestAppModuleBasic_Name(t *testing.T) {
	b := xauthz.AppModuleBasic{}
	require.Equal(t, types.ModuleName, b.Name())
}

func TestNewAppModule(t *testing.T) {
	am := xauthz.NewAppModule()
	require.NotNil(t, am)

	// Verify the module implements the correct interfaces
	var _ module.AppModule = am //nolint:staticcheck // deprecated but still required
}

func TestAppModule_IsOnePerModuleType(t *testing.T) {
	am := xauthz.NewAppModule()
	require.NotPanics(t, func() {
		am.IsOnePerModuleType()
	})
}

func TestAppModule_IsAppModule(t *testing.T) {
	am := xauthz.NewAppModule()
	require.NotPanics(t, func() {
		am.IsAppModule()
	})
}

func TestAppModuleBasic_RegisterLegacyAminoCodec(t *testing.T) {
	b := xauthz.AppModuleBasic{}
	cdc := codec.NewLegacyAmino()
	require.NotPanics(t, func() {
		b.RegisterLegacyAminoCodec(cdc)
	})
}

func TestAppModuleBasic_RegisterInterfaces(t *testing.T) {
	b := xauthz.AppModuleBasic{}
	registry := codectypes.NewInterfaceRegistry()
	require.NotPanics(t, func() {
		b.RegisterInterfaces(registry)
	})
}

func TestAppModule_ConsensusVersion(t *testing.T) {
	b := xauthz.AppModuleBasic{}
	require.Equal(t, uint64(1), b.ConsensusVersion())
}

func TestAppModuleBasic_RegisterGRPCGatewayRoutes(t *testing.T) {
	b := xauthz.AppModuleBasic{}
	mux := runtime.NewServeMux()
	// no-op, should not panic
	require.NotPanics(t, func() {
		b.RegisterGRPCGatewayRoutes(client.Context{}, mux)
	})
}

func TestAppModuleBasic_GetTxCmd(t *testing.T) {
	b := xauthz.AppModuleBasic{}
	txCmd := b.GetTxCmd()
	require.NotNil(t, txCmd)
	require.Equal(t, types.ModuleName, txCmd.Use)
}
