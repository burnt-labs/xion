package globalfee

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"

	abci "github.com/cometbft/cometbft/abci/types"

	corestore "cosmossdk.io/core/store"
	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/burnt-labs/xion/x/globalfee/client/cli"
	"github.com/burnt-labs/xion/x/globalfee/keeper"
	"github.com/burnt-labs/xion/x/globalfee/types"
)

var (
	_ module.AppModuleBasic   = AppModuleBasic{}
	_ module.AppModuleGenesis = AppModule{}
	_ module.AppModule        = AppModule{}
)

// AppModuleBasic defines the basic application module used by the wasm module.
type AppModuleBasic struct{}

// reused literal to avoid triggering goconst / allow inline explanatory comments
const noOp = "no-op"

func (a AppModuleBasic) Name() string {
	return types.ModuleName
}

func (a AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(&types.GenesisState{
		Params: types.DefaultParams(),
	})
}

func (a AppModuleBasic) ValidateGenesis(marshaler codec.JSONCodec, _ client.TxEncodingConfig, message json.RawMessage) error {
	var data types.GenesisState
	err := marshaler.UnmarshalJSON(message, &data)
	if err != nil {
		return err
	}
	if err := data.Params.ValidateBasic(); err != nil {
		return errorsmod.Wrap(err, "params")
	}
	return nil
}

func (a AppModuleBasic) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	types.RegisterInterfaces(registry)
}

func (a AppModuleBasic) RegisterRESTRoutes(_ client.Context, _ *mux.Router) {
	// No REST routes to register for globalfee module
	_ = noOp
}

// RegisterQueryHandlerClientFn is an overrideable hook (exported for tests) used to register
// gRPC gateway query handlers. Tests can substitute this to force error paths and achieve
// full coverage of panic handling without altering consensus logic.
var RegisterQueryHandlerClientFn = types.RegisterQueryHandlerClient

func (a AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
	err := RegisterQueryHandlerClientFn(context.Background(), mux, types.NewQueryClient(clientCtx))
	if err != nil {
		// same behavior as in cosmos-sdk
		panic(err)
	}
}

func (a AppModuleBasic) GetTxCmd() *cobra.Command {
	return nil
}

func (a AppModuleBasic) GetQueryCmd() *cobra.Command {
	return cli.GetQueryCmd()
}

func (a AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterLegacyAminoCodec(cdc)
}

type AppModule struct {
	AppModuleBasic
	keeper       keeper.Keeper
	legacyParams corestore.KVStoreService
	legacyAmino  *codec.LegacyAmino
}

func (a AppModule) IsOnePerModuleType() {
	// Interface marker method - indicates this module should only be instantiated once per chain
	_ = noOp
}

func (a AppModule) IsAppModule() {
	// Interface marker method - indicates this implements the AppModule interface
	_ = noOp
}

func NewAppModule(_ codec.Codec, k keeper.Keeper, legacyParams corestore.KVStoreService, legacyAmino *codec.LegacyAmino) *AppModule {
	return &AppModule{keeper: k, legacyParams: legacyParams, legacyAmino: legacyAmino}
}

func (a AppModule) InitGenesis(ctx sdk.Context, marshaler codec.JSONCodec, message json.RawMessage) []abci.ValidatorUpdate {
	var genesisState types.GenesisState
	marshaler.MustUnmarshalJSON(message, &genesisState)

	if err := a.keeper.SetParams(ctx, genesisState.Params); err != nil {
		panic(err)
	}
	return nil
}

func (a AppModule) ExportGenesis(ctx sdk.Context, marshaler codec.JSONCodec) json.RawMessage {
	params, err := a.keeper.GetParams(ctx)
	if err != nil {
		panic(err)
	}
	return marshaler.MustMarshalJSON(&types.GenesisState{Params: params})
}

func (a AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(a.keeper))
	types.RegisterQueryServer(cfg.QueryServer(), keeper.NewQuerier(a.keeper))

	m := keeper.NewMigrator(a.keeper, a.legacyParams, a.legacyAmino)
	if err := cfg.RegisterMigration(types.ModuleName, 2, m.Migrate2to3); err != nil {
		panic(fmt.Sprintf("failed to migrate x/globalfee from version 2 to 3: %v", err))
	}
}

// ConsensusVersion is a sequence number for state-breaking change of the
// module. It should be incremented on each consensus-breaking change
// introduced by the module. To avoid wrong/empty versions, the initial version
// should be set to 1.
func (a AppModule) ConsensusVersion() uint64 {
	return 3
}
