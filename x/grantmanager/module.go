package grantmanager

import (
	"cosmossdk.io/core/appmodule"
	grantmanagerkeeper "github.com/burnt-labs/xion/x/grantmanager/keeper"
	grantmanagertypes "github.com/burnt-labs/xion/x/grantmanager/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
)

const ModuleName = "grantmanager"

var (
	_ module.AppModuleBasic = AppModuleBasic{}
)

// AppModuleBasic defines the basic application module.
type AppModuleBasic struct{}

// RegisterGRPCGatewayRoutes
func (a AppModuleBasic) RegisterGRPCGatewayRoutes(client.Context, *runtime.ServeMux) {

}
func (a AppModuleBasic) RegisterLegacyAminoCodec(*codec.LegacyAmino) {

}
func (a AppModuleBasic) RegisterInterfaces(codectypes.InterfaceRegistry) {}

func (a AppModuleBasic) Name() string {
	return grantmanagertypes.ModuleName
}

var _ appmodule.AppModule = AppModule{}

type AppModule struct {
	AppModuleBasic
	cdc    codec.Codec
	keeper grantmanagerkeeper.Keeper
}

// NewAppModule creates a new grant manager AppModule
func NewAppModule(cdc codec.Codec, keeper grantmanagerkeeper.Keeper) AppModule {
	return AppModule{
		cdc:    cdc,
		keeper: keeper,
	}
}

// RegisterServices wires new messages registration and routing
func (am AppModule) RegisterServices(cfg module.Configurator) {
	grantmanagertypes.RegisterMsgServer(cfg.MsgServer(), grantmanagerkeeper.NewMsgServer(am.keeper))

}

func (am AppModule) IsOnePerModuleType() {}

func (am AppModule) IsAppModule() {}

func (am AppModule) ConsensusVersion() uint64 { return 1 }

func (am AppModule) RegisterInvariants(_ sdk.InvariantRegistry) {}
