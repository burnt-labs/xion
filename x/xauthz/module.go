package xauthz

import (
	"github.com/burnt-labs/xion/x/xauthz/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/module"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	gwruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"
)

const ConsensusVersion = 1

var (
	_ module.AppModuleBasic = AppModuleBasic{}
	_ module.AppModule      = AppModule{}
)

// AppModuleBasic defines the basic application module used by the xauthz(xion authz)  module.
type AppModuleBasic struct {
}

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
	_ = AppModule{}
}

func (AppModule) IsAppModule() {
	_ = AppModule{}
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
}
