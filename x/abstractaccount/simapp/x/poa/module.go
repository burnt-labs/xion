package poa

import (
	"encoding/json"
	"fmt"

	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"

	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/burnt-labs/xion/x/abstractaccount/simapp/x/poa/types"
)

var (
	//nolint:staticcheck // SA1019: module.AppModule is deprecated but migration to appmodule.AppModule requires larger refactor
	_ module.AppModule      = AppModule{}
	_ module.AppModuleBasic = AppModuleBasic{}
)

// ------------------------------ AppModuleBasic -------------------------------

type AppModuleBasic struct{}

func (AppModuleBasic) Name() string {
	return types.ModuleName
}

func (AppModuleBasic) RegisterInterfaces(_ codectypes.InterfaceRegistry) {
	// nothing to register
}

func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(types.DefaultGenesisState())
}

func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	var gs types.GenesisState
	if err := cdc.UnmarshalJSON(bz, &gs); err != nil {
		return fmt.Errorf("failed to unmarshal x/%s genesis state: %w", types.ModuleName, err)
	}

	return gs.Validate()
}

func (AppModuleBasic) RegisterGRPCGatewayRoutes(_ client.Context, _ *runtime.ServeMux) {
	// nothing to register
}

func (AppModuleBasic) GetTxCmd() *cobra.Command {
	return nil
}

func (AppModuleBasic) GetQueryCmd() *cobra.Command {
	return nil
}

// --------------------------------- AppModule ---------------------------------

type AppModule struct {
	AppModuleBasic
}

func NewAppModule() AppModule {
	return AppModule{AppModuleBasic{}}
}

func (am AppModule) IsOnePerModuleType() {
}

func (am AppModule) IsAppModule() {
}

func (am AppModule) RegisterServices(_ module.Configurator) {
	// nothing to register
}

func (AppModule) ConsensusVersion() uint64 {
	return 1
}

func (AppModule) EndBlock(_ sdk.Context, _ abci.RequestFinalizeBlock) []abci.ValidatorUpdate {
	return []abci.ValidatorUpdate{}
}

func (am AppModule) InitGenesis(_ sdk.Context, cdc codec.JSONCodec, data json.RawMessage) []abci.ValidatorUpdate {
	var gs types.GenesisState
	cdc.MustUnmarshalJSON(data, &gs)

	return gs.Validators
}

func (am AppModule) ExportGenesis(_ sdk.Context, _ codec.JSONCodec) json.RawMessage {
	panic("UNIMPLEMENTED")
}

// ----------------------------- Deprecated stuff ------------------------------

// deprecated
func (AppModuleBasic) RegisterLegacyAminoCodec(_ *codec.LegacyAmino) {
}

// deprecated
func (AppModuleBasic) RegisterRESTRoutes(_ client.Context, _ *mux.Router) {
}
