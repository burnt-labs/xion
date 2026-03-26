package module

import (
	"context"
	"encoding/json"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"

	abci "github.com/cometbft/cometbft/abci/types"

	"cosmossdk.io/client/v2/autocli"
	"cosmossdk.io/core/appmodule"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/burnt-labs/xion/x/tee/client/cli"
	"github.com/burnt-labs/xion/x/tee/keeper"
	"github.com/burnt-labs/xion/x/tee/types"
)

const (
	// ConsensusVersion defines the current x/tee module consensus version.
	ConsensusVersion = 1
)

var (
	_ module.AppModuleBasic    = AppModuleBasic{}
	_ module.AppModuleGenesis  = AppModule{}
	_ module.AppModule         = AppModule{}
	_ autocli.HasAutoCLIConfig = AppModule{}
	_ appmodule.AppModule      = AppModule{}
)

// AppModuleBasic defines the basic application module used by the tee module.
type AppModuleBasic struct {
	cdc codec.Codec
}

// AppModule implements an application module for the tee module.
type AppModule struct {
	AppModuleBasic

	keeper keeper.Keeper
}

// NewAppModule constructor
func NewAppModule(
	cdc codec.Codec,
	keeper keeper.Keeper,
) *AppModule {
	return &AppModule{
		AppModuleBasic: AppModuleBasic{cdc: cdc},
		keeper:         keeper,
	}
}

func (a AppModuleBasic) Name() string {
	return types.ModuleName
}

func (a AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(types.DefaultGenesisState())
}

func (a AppModuleBasic) ValidateGenesis(marshaler codec.JSONCodec, _ client.TxEncodingConfig, message json.RawMessage) error {
	var data types.GenesisState
	err := marshaler.UnmarshalJSON(message, &data)
	if err != nil {
		return err
	}
	return data.Validate()
}

func (a AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
	err := types.RegisterQueryHandlerClient(context.Background(), mux, types.NewQueryClient(clientCtx))
	if err != nil {
		panic(err)
	}
}

func (a AppModuleBasic) GetTxCmd() *cobra.Command {
	return nil
}

func (a AppModuleBasic) GetQueryCmd() *cobra.Command {
	return cli.GetQueryCmd()
}

func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterLegacyAminoCodec(cdc)
}

func (a AppModuleBasic) RegisterInterfaces(r codectypes.InterfaceRegistry) {
	types.RegisterInterfaces(r)
}

func (am AppModule) InitGenesis(_ sdk.Context, _ codec.JSONCodec, _ json.RawMessage) []abci.ValidatorUpdate {
	return nil
}

func (am AppModule) ExportGenesis(_ sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(types.DefaultGenesisState())
}

//nolint:staticcheck // SA1019: Deprecated but required for module.AppModule interface
func (am AppModule) RegisterInvariants(_ sdk.InvariantRegistry) {
	_ = am
}

func (am AppModule) QuerierRoute() string {
	return types.QuerierRoute
}

func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterQueryServer(cfg.QueryServer(), keeper.NewQuerier(am.keeper))
}

func (am AppModule) ConsensusVersion() uint64 {
	return ConsensusVersion
}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (am AppModule) IsOnePerModuleType() {
	_ = am
}

// IsAppModule implements the appmodule.AppModule interface.
func (am AppModule) IsAppModule() {
	_ = am
}
