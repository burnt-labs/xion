package mint

import (
	"context"
	"encoding/json"
	"fmt"

	gwruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"

	modulev1 "cosmossdk.io/api/cosmos/mint/module/v1"
	"cosmossdk.io/core/appmodule"
	store2 "cosmossdk.io/core/store"
	"cosmossdk.io/depinject"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/burnt-labs/xion/x/mint/client/cli"
	"github.com/burnt-labs/xion/x/mint/exported"
	"github.com/burnt-labs/xion/x/mint/keeper"
	"github.com/burnt-labs/xion/x/mint/simulation"
	"github.com/burnt-labs/xion/x/mint/types"
)

// ConsensusVersion defines the current x/mint module consensus version.
const ConsensusVersion = 1

var (
	_ module.AppModuleBasic      = AppModuleBasic{}
	_ module.AppModuleSimulation = AppModule{}
	_ module.HasGenesis          = AppModule{}
	_ module.HasServices         = AppModule{}

	_ appmodule.AppModule       = AppModule{}
	_ appmodule.HasBeginBlocker = AppModule{}
)

// AppModuleBasic defines the basic application module used by the mint module.
type AppModuleBasic struct {
	cdc codec.Codec
}

var _ module.AppModuleBasic = AppModuleBasic{}

// Name returns the mint module's name.
func (AppModuleBasic) Name() string {
	return types.ModuleName
}

// RegisterLegacyAminoCodec registers the mint module's types on the given LegacyAmino codec.
func (AppModuleBasic) RegisterLegacyAminoCodec(_ *codec.LegacyAmino) {
	// types.RegisterLegacyAminoCodec(cdc)
}

// RegisterInterfaces registers the module's interface types
func (b AppModuleBasic) RegisterInterfaces(_ cdctypes.InterfaceRegistry) {
	// types.RegisterInterfaces(r)
}

// DefaultGenesis returns default genesis state as raw bytes for the mint
// module.
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(types.DefaultGenesisState())
}

// ValidateGenesis performs genesis state validation for the mint module.
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	var data types.GenesisState
	if err := cdc.UnmarshalJSON(bz, &data); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", types.ModuleName, err)
	}

	return types.ValidateGenesis(data)
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the mint module.
func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *gwruntime.ServeMux) {
	if err := types.RegisterQueryHandlerClient(context.Background(), mux, types.NewQueryClient(clientCtx)); err != nil {
		panic(err)
	}
}

// GetTxCmd returns no root tx command for the mint module.
func (AppModuleBasic) GetTxCmd() *cobra.Command { return nil }

// GetQueryCmd returns the root query command for the mint module.
func (AppModuleBasic) GetQueryCmd() *cobra.Command {
	return cli.GetQueryCmd()
}

// AppModule implements an application module for the mint module.
type AppModule struct {
	AppModuleBasic

	keeper     keeper.Keeper
	authKeeper types.AccountKeeper

	// legacySubspace is used solely for migration of x/params managed parameters
	legacySubspace exported.Subspace

	// inflationCalculator is used to calculate the inflation rate during BeginBlock.
	// If inflationCalculator is nil, the default inflation calculation logic is used.
	inflationCalculator types.InflationCalculationFn
}

// NewAppModule creates a new AppModule object. If the InflationCalculationFn
// argument is nil, then the SDK's default inflation function will be used.
func NewAppModule(
	cdc codec.Codec,
	keeper keeper.Keeper,
	ak types.AccountKeeper,
	ic types.InflationCalculationFn,
	ss exported.Subspace,
) AppModule {
	if ic == nil {
		ic = types.DefaultInflationCalculationFn
	}

	return AppModule{
		AppModuleBasic:      AppModuleBasic{cdc: cdc},
		keeper:              keeper,
		authKeeper:          ak,
		inflationCalculator: ic,
		legacySubspace:      ss,
	}
}

var _ appmodule.AppModule = AppModule{}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (am AppModule) IsOnePerModuleType() {}

// IsAppModule implements the appmodule.AppModule interface.
func (am AppModule) IsAppModule() {}

// Name returns the mint module's name.
func (AppModule) Name() string {
	return types.ModuleName
}

// RegisterServices registers a gRPC query service to respond to the
// module-specific gRPC queries.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper))
	types.RegisterQueryServer(cfg.QueryServer(), am.keeper)
}

// InitGenesis performs genesis initialization for the mint module. It returns
// no validator updates.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) {
	var genesisState types.GenesisState
	cdc.MustUnmarshalJSON(data, &genesisState)

	am.keeper.InitGenesis(ctx, am.authKeeper, &genesisState)
}

// ExportGenesis returns the exported genesis state as raw bytes for the mint
// module.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	gs := am.keeper.ExportGenesis(ctx)
	return cdc.MustMarshalJSON(gs)
}

// ConsensusVersion implements AppModule/ConsensusVersion.
func (AppModule) ConsensusVersion() uint64 { return ConsensusVersion }

// BeginBlock returns the BeginBlocker for the mint module.
func (am AppModule) BeginBlock(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return BeginBlocker(sdkCtx, am.keeper, am.inflationCalculator)
}

// AppModuleSimulation functions
func (am AppModule) RegisterStoreDecoder(registry simtypes.StoreDecoderRegistry) {
	registry[types.StoreKey] = simulation.NewDecodeStore(am.cdc)
}

// GenerateGenesisState creates a randomized GenState of the mint module.
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	simulation.RandomizedGenState(simState)
}

// ProposalMsgs returns msgs used for governance proposals for simulations.
func (AppModule) ProposalMsgs(_ module.SimulationState) []simtypes.WeightedProposalMsg {
	return simulation.ProposalMsgs()
}

// RegisterStoreDecoder registers a decoder for mint module's types.
/*func (am AppModule) RegisterStoreDecoder(sdr sdk.StoreDecoderRegistry) {
	sdr[types.StoreKey] = simulation.NewDecodeStore(am.cdc)
}
*/

// WeightedOperations doesn't return any mint module operation.
func (AppModule) WeightedOperations(_ module.SimulationState) []simtypes.WeightedOperation {
	return nil
}

//
// App Wiring Setup
//

func init() {
	appmodule.Register(&modulev1.Module{},
		appmodule.Provide(ProvideModule),
	)
}

type Inputs struct {
	depinject.In

	ModuleKey              depinject.OwnModuleKey
	Config                 *modulev1.Module
	Key                    store2.KVStoreService
	Cdc                    codec.Codec
	InflationCalculationFn types.InflationCalculationFn `optional:"true"`

	// LegacySubspace is used solely for migration of x/params managed parameters
	LegacySubspace exported.Subspace

	AccountKeeper types.AccountKeeper
	BankKeeper    types.BankKeeper
	StakingKeeper types.StakingKeeper
}

type Outputs struct {
	depinject.Out

	MintKeeper keeper.Keeper
	Module     appmodule.AppModule
}

func ProvideModule(in Inputs) Outputs {
	feeCollectorName := in.Config.FeeCollectorName
	if feeCollectorName == "" {
		feeCollectorName = authtypes.FeeCollectorName
	}

	// default to governance authority if not provided
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)
	if in.Config.Authority != "" {
		authority = authtypes.NewModuleAddressOrBech32Address(in.Config.Authority)
	}

	k := keeper.NewKeeper(
		in.Cdc,
		in.Key,
		in.StakingKeeper,
		in.AccountKeeper,
		in.BankKeeper,
		feeCollectorName,
		authority.String(),
	)

	// when no inflation calculation function is provided it will use the default types.DefaultInflationCalculationFn
	m := NewAppModule(in.Cdc, k, in.AccountKeeper, in.InflationCalculationFn, in.LegacySubspace)

	return Outputs{MintKeeper: k, Module: m}
}
