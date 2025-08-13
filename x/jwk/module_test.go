package jwk_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"

	"github.com/burnt-labs/xion/x/jwk"
	"github.com/burnt-labs/xion/x/jwk/types"
)

func TestJWKModule(t *testing.T) {
	// Create codec
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

	// Test NewAppModuleBasic
	appModuleBasic := jwk.NewAppModuleBasic(cdc)
	require.NotNil(t, appModuleBasic)

	// Test Name
	require.Equal(t, types.ModuleName, appModuleBasic.Name())

	// Test RegisterLegacyAminoCodec
	amino := codec.NewLegacyAmino()
	require.NotPanics(t, func() {
		appModuleBasic.RegisterLegacyAminoCodec(amino)
	})

	// Test RegisterInterfaces
	registry := codectypes.NewInterfaceRegistry()
	require.NotPanics(t, func() {
		appModuleBasic.RegisterInterfaces(registry)
	})

	// Test DefaultGenesis
	genesis := appModuleBasic.DefaultGenesis(cdc)
	require.NotNil(t, genesis)

	// Test GetTxCmd
	txCmd := appModuleBasic.GetTxCmd()
	require.NotNil(t, txCmd)

	// Test GetQueryCmd
	queryCmd := appModuleBasic.GetQueryCmd()
	require.NotNil(t, queryCmd)
}

func TestJWKAppModule(t *testing.T) {
	// Create a basic app module for testing
	appModule := jwk.AppModule{}

	// Test IsOnePerModuleType
	require.NotPanics(t, func() {
		appModule.IsOnePerModuleType()
	})

	// Test IsAppModule
	require.NotPanics(t, func() {
		appModule.IsAppModule()
	})

	// Test ConsensusVersion
	version := appModule.ConsensusVersion()
	require.Equal(t, uint64(2), version)
}
