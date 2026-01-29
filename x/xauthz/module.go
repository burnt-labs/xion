package xauthz

import (
	gwruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/burnt-labs/xion/x/xauthz/client/cli"
	"github.com/burnt-labs/xion/x/xauthz/types"
)

const ConsensusVersion = 1

var (
	_ module.AppModuleBasic = AppModuleBasic{}
	_ module.AppModule      = AppModule{} //nolint:staticcheck // deprecated but still required
)

// AppModuleBasic defines the basic application module used by the xauthz(xion authz)  module.
type AppModuleBasic struct{}

func (AppModuleBasic) Name() string {
	return types.ModuleName
}

// AppModule implements the AppModule interface for the xauthz(xion authz) module.
type AppModule struct {
	AppModuleBasic
}

func NewAppModule() AppModule {
	return AppModule{}
}

func (AppModule) IsOnePerModuleType() {
	_ = "IsOnePerModuleType"
}

func (AppModule) IsAppModule() {
	_ = "IsAppModule"
}

func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterLegacyAminoCodec(cdc)
}

func (AppModuleBasic) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	types.RegisterInterfaces(registry)
}

// ConsensusVersion returns the consensus version of the xauthz module.
func (AppModuleBasic) ConsensusVersion() uint64 { return ConsensusVersion }

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the xauthz module.
func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *gwruntime.ServeMux) {
	_ = "RegisterGRPCGatewayRoutes"
}

// GetTxCmd returns the root tx command for the xauthz module.
func (AppModuleBasic) GetTxCmd() *cobra.Command {
	return cli.GetTxCmd()
}
