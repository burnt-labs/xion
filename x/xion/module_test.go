package xion_test

import (
	"testing"

	"github.com/burnt-labs/xion/x/xion"
	"github.com/burnt-labs/xion/x/xion/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stretchr/testify/require"
)

// minimal mocks for required keeper interfaces

// No external keeper behavior needed; we rely on zero-value keeper in AppModule.

func TestAppModuleBasicCoverage(t *testing.T) {
	b := xion.AppModuleBasic{}
	// Name
	require.Equal(t, types.ModuleName, b.Name())
	// Legacy amino
	amino := codec.NewLegacyAmino()
	require.NotPanics(t, func() { b.RegisterLegacyAminoCodec(amino) })
	// Interfaces
	reg := codectypes.NewInterfaceRegistry()
	require.NotPanics(t, func() { b.RegisterInterfaces(reg) })
	// Default genesis & ValidateGenesis
	cdc := codec.NewProtoCodec(reg)
	gen := b.DefaultGenesis(cdc)
	require.NotNil(t, gen)
	var gs types.GenesisState
	require.NoError(t, cdc.UnmarshalJSON(gen, &gs))
	// valid
	var txCfg client.TxEncodingConfig
	require.NoError(t, b.ValidateGenesis(cdc, txCfg, gen))
	// invalid json
	require.Error(t, b.ValidateGenesis(cdc, txCfg, []byte(`{"bad":1}`)))
	// grpc gateway routes (no-op)
	mux := runtime.NewServeMux()
	clientCtx := client.Context{}
	require.NotPanics(t, func() { b.RegisterGRPCGatewayRoutes(clientCtx, mux) })
	// tx/query cmds
	txCmd := b.GetTxCmd()
	require.NotNil(t, txCmd)
	queryCmd := b.GetQueryCmd()
	require.NotNil(t, queryCmd)
}
