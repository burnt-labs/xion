// nolint: staticcheck
package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"runtime/debug"

	"github.com/CosmWasm/wasmd/x/wasm"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	wasmvm "github.com/CosmWasm/wasmvm/v3"
	aa "github.com/burnt-labs/abstract-account/x/abstractaccount"
	aakeeper "github.com/burnt-labs/abstract-account/x/abstractaccount/keeper"
	aatypes "github.com/burnt-labs/abstract-account/x/abstractaccount/types"
	"github.com/spf13/cast"
	"github.com/strangelove-ventures/tokenfactory/x/tokenfactory"
	"github.com/strangelove-ventures/tokenfactory/x/tokenfactory/bindings"
	tokenfactorykeeper "github.com/strangelove-ventures/tokenfactory/x/tokenfactory/keeper"
	tokenfactorytypes "github.com/strangelove-ventures/tokenfactory/x/tokenfactory/types"

	abci "github.com/cometbft/cometbft/abci/types"
	tmos "github.com/cometbft/cometbft/libs/os"
	cmtstate "github.com/cometbft/cometbft/proto/tendermint/state"
	cmtstore "github.com/cometbft/cometbft/proto/tendermint/store"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/linxGnu/grocksdb"
	"github.com/cosmos/gogoproto/proto"
	packetforward "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v10/packetforward"
	packetforwardkeeper "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v10/packetforward/keeper"
	packetforwardtypes "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v10/packetforward/types"
	ibcwasm "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10"
	ibcwasmkeeper "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10/keeper"
	ibcwasmtypes "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/v10/types"
	ica "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts"
	icacontroller "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller"
	icacontrollerkeeper "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/keeper"
	icacontrollertypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/types"
	icahost "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host"
	icahostkeeper "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host/keeper"
	icahosttypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"
	ibccallbacks "github.com/cosmos/ibc-go/v10/modules/apps/callbacks"
	ibccallbacksv2 "github.com/cosmos/ibc-go/v10/modules/apps/callbacks/v2"
	"github.com/cosmos/ibc-go/v10/modules/apps/transfer"
	ibctransferkeeper "github.com/cosmos/ibc-go/v10/modules/apps/transfer/keeper"
	ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	transferv2 "github.com/cosmos/ibc-go/v10/modules/apps/transfer/v2"
	ibc "github.com/cosmos/ibc-go/v10/modules/core"
	ibcclienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	ibcconnectiontypes "github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	ibcapi "github.com/cosmos/ibc-go/v10/modules/core/api"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibckeeper "github.com/cosmos/ibc-go/v10/modules/core/keeper"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"

	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	reflectionv1 "cosmossdk.io/api/cosmos/reflection/v1"
	"cosmossdk.io/client/v2/autocli"
	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	"cosmossdk.io/x/circuit"
	circuitkeeper "cosmossdk.io/x/circuit/keeper"
	circuittypes "cosmossdk.io/x/circuit/types"
	"cosmossdk.io/x/evidence"
	evidencekeeper "cosmossdk.io/x/evidence/keeper"
	evidencetypes "cosmossdk.io/x/evidence/types"
	"cosmossdk.io/x/feegrant"
	feegrantkeeper "cosmossdk.io/x/feegrant/keeper"
	feegrantmodule "cosmossdk.io/x/feegrant/module"
	"cosmossdk.io/x/nft"
	nftkeeper "cosmossdk.io/x/nft/keeper"
	nftmodule "cosmossdk.io/x/nft/module"
	"cosmossdk.io/x/tx/signing"
	upgrademodule "cosmossdk.io/x/upgrade"
	upgradekeeper "cosmossdk.io/x/upgrade/keeper"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/grpc/cmtservice"
	nodeservice "github.com/cosmos/cosmos-sdk/client/grpc/node"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	runtimeservices "github.com/cosmos/cosmos-sdk/runtime/services"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/std"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	"github.com/cosmos/cosmos-sdk/x/auth/posthandler"
	authsims "github.com/cosmos/cosmos-sdk/x/auth/simulation"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/auth/vesting"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	authzmodule "github.com/cosmos/cosmos-sdk/x/authz/module"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/consensus"
	consensusparamkeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	consensusparamtypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	crisiskeeper "github.com/cosmos/cosmos-sdk/x/crisis/keeper"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/cosmos/cosmos-sdk/x/group"
	groupkeeper "github.com/cosmos/cosmos-sdk/x/group/keeper"
	groupmodule "github.com/cosmos/cosmos-sdk/x/group/module"
	"github.com/cosmos/cosmos-sdk/x/mint"
	mintkeeper "github.com/cosmos/cosmos-sdk/x/mint/keeper"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/cosmos/cosmos-sdk/x/params"
	paramsclient "github.com/cosmos/cosmos-sdk/x/params/client"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	paramproposal "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/burnt-labs/xion/indexer"
	owasm "github.com/burnt-labs/xion/wasmbindings"
	dkim "github.com/burnt-labs/xion/x/dkim"
	dkimkeeper "github.com/burnt-labs/xion/x/dkim/keeper"
	dkimtypes "github.com/burnt-labs/xion/x/dkim/types"
	"github.com/burnt-labs/xion/x/globalfee"
	"github.com/burnt-labs/xion/x/jwk"
	jwkkeeper "github.com/burnt-labs/xion/x/jwk/keeper"
	jwktypes "github.com/burnt-labs/xion/x/jwk/types"
	"github.com/burnt-labs/xion/x/xion"
	xionkeeper "github.com/burnt-labs/xion/x/xion/keeper"
	xiontypes "github.com/burnt-labs/xion/x/xion/types"
	zk "github.com/burnt-labs/xion/x/zk"
	zkkeeper "github.com/burnt-labs/xion/x/zk/keeper"
	zktypes "github.com/burnt-labs/xion/x/zk/types"
)

const (
	appName                 = "XionApp"
	WasmContractMemoryLimit = 32
)

// We pull these out, so we can set them with LDFLAGS in the Makefile
var (
	NodeDir      = ".xiond"
	Bech32Prefix = "xion"

	// If EnabledSpecificProposals is "", and this is "true", then enable all x/wasm proposals.
	// If EnabledSpecificProposals is "", and this is not "true", then disable all x/wasm proposals.
	ProposalsEnabled = "true"
	// If set to non-empty string it must be comma-separated list of values that are all a subset
	// of "EnableAllProposals" (takes precedence over ProposalsEnabled)
	// https://github.com/CosmWasm/wasmd/blob/02a54d33ff2c064f3539ae12d75d027d9c665f05/x/wasm/internal/types/proposal.go#L28-L34
	EnableSpecificProposals = ""
)

// These constants are derived from the above variables.
// These are the ones we will want to use in the code, based on
// any overrides above
var (
	// DefaultNodeHome default home directories for xiond
	DefaultNodeHome = os.ExpandEnv("$HOME/") + NodeDir

	// Bech32PrefixAccAddr defines the Bech32 prefix of an account's address
	Bech32PrefixAccAddr = Bech32Prefix
	// Bech32PrefixAccPub defines the Bech32 prefix of an account's public key
	Bech32PrefixAccPub = Bech32Prefix + sdk.PrefixPublic
	// Bech32PrefixValAddr defines the Bech32 prefix of a validator's operator address
	Bech32PrefixValAddr = Bech32Prefix + sdk.PrefixValidator + sdk.PrefixOperator
	// Bech32PrefixValPub defines the Bech32 prefix of a validator's operator public key
	Bech32PrefixValPub = Bech32Prefix + sdk.PrefixValidator + sdk.PrefixOperator + sdk.PrefixPublic
	// Bech32PrefixConsAddr defines the Bech32 prefix of a consensus node address
	Bech32PrefixConsAddr = Bech32Prefix + sdk.PrefixValidator + sdk.PrefixConsensus
	// Bech32PrefixConsPub defines the Bech32 prefix of a consensus node public key
	Bech32PrefixConsPub = Bech32Prefix + sdk.PrefixValidator + sdk.PrefixConsensus + sdk.PrefixPublic
)

var (
	// module account permissions
	maccPerms = map[string][]string{
		authtypes.FeeCollectorName:     {authtypes.Burner},
		distrtypes.ModuleName:          nil,
		minttypes.ModuleName:           {authtypes.Minter},
		stakingtypes.BondedPoolName:    {authtypes.Burner, authtypes.Staking},
		stakingtypes.NotBondedPoolName: {authtypes.Burner, authtypes.Staking},
		govtypes.ModuleName:            {authtypes.Burner},
		nft.ModuleName:                 nil,
		// non sdk modules
		ibctransfertypes.ModuleName:   {authtypes.Minter, authtypes.Burner},
		icatypes.ModuleName:           nil,
		wasmtypes.ModuleName:          {authtypes.Burner},
		tokenfactorytypes.ModuleName:  {authtypes.Minter, authtypes.Burner},
		globalfee.ModuleName:          nil,
		aatypes.ModuleName:            nil,
		xiontypes.ModuleName:          nil,
		jwktypes.ModuleName:           nil,
		packetforwardtypes.ModuleName: nil,
	}
	tokenFactoryCapabilities = []string{
		tokenfactorytypes.EnableBurnFrom,
		tokenfactorytypes.EnableForceTransfer,
		tokenfactorytypes.EnableSetMetadata,
	}
)

var (
	_ runtime.AppI            = (*WasmApp)(nil)
	_ servertypes.Application = (*WasmApp)(nil)
)

// WasmApp extended ABCI application
type WasmApp struct {
	*baseapp.BaseApp
	legacyAmino       *codec.LegacyAmino
	appCodec          codec.Codec
	txConfig          client.TxConfig
	interfaceRegistry types.InterfaceRegistry

	// keys to access the substores
	keys  map[string]*storetypes.KVStoreKey
	tkeys map[string]*storetypes.TransientStoreKey

	// keepers
	AccountKeeper         authkeeper.AccountKeeper
	BankKeeper            bankkeeper.BaseKeeper
	StakingKeeper         *stakingkeeper.Keeper
	SlashingKeeper        slashingkeeper.Keeper
	MintKeeper            mintkeeper.Keeper
	DistrKeeper           distrkeeper.Keeper
	GovKeeper             govkeeper.Keeper
	CrisisKeeper          *crisiskeeper.Keeper
	UpgradeKeeper         *upgradekeeper.Keeper
	ParamsKeeper          paramskeeper.Keeper
	AuthzKeeper           authzkeeper.Keeper
	EvidenceKeeper        evidencekeeper.Keeper
	FeeGrantKeeper        feegrantkeeper.Keeper
	GroupKeeper           groupkeeper.Keeper
	NFTKeeper             nftkeeper.Keeper
	ConsensusParamsKeeper consensusparamkeeper.Keeper
	CircuitKeeper         circuitkeeper.Keeper

	IBCKeeper             *ibckeeper.Keeper // IBC Keeper must be a pointer in the app, so we can SetRouter on it correctly
	ICAControllerKeeper   icacontrollerkeeper.Keeper
	ICAHostKeeper         icahostkeeper.Keeper
	TransferKeeper        ibctransferkeeper.Keeper
	WasmKeeper            wasmkeeper.Keeper
	WasmClientKeeper      ibcwasmkeeper.Keeper
	AbstractAccountKeeper aakeeper.Keeper
	ContractKeeper        *wasmkeeper.PermissionedKeeper
	PacketForwardKeeper   *packetforwardkeeper.Keeper

	XionKeeper         xionkeeper.Keeper
	JwkKeeper          jwkkeeper.Keeper
	TokenFactoryKeeper tokenfactorykeeper.Keeper
	DkimKeeper         dkimkeeper.Keeper
	ZkKeeper           zkkeeper.Keeper

	// the module manager
	ModuleManager      *module.Manager
	BasicModuleManager module.BasicManager

	// simulation manager
	sm *module.SimulationManager

	// module configurator
	configurator module.Configurator

	// indexer
	indexerService *indexer.StreamService
}

// NewWasmApp returns a reference to an initialized WasmApp.
func NewWasmApp(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	loadLatest bool,
	appOpts servertypes.AppOptions,
	wasmOpts []wasmkeeper.Option,
	baseAppOptions ...func(*baseapp.BaseApp),
) *WasmApp {
	overrideWasmVariables()

	interfaceRegistry, err := types.NewInterfaceRegistryWithOptions(types.InterfaceRegistryOptions{
		ProtoFiles: proto.HybridResolver,
		SigningOptions: signing.Options{
			AddressCodec: address.Bech32Codec{
				Bech32Prefix: sdk.GetConfig().GetBech32AccountAddrPrefix(),
			},
			ValidatorAddressCodec: address.Bech32Codec{
				Bech32Prefix: sdk.GetConfig().GetBech32ValidatorAddrPrefix(),
			},
		},
	})
	if err != nil {
		panic(err)
	}
	appCodec := codec.NewProtoCodec(interfaceRegistry)
	legacyAmino := codec.NewLegacyAmino()
	txConfig := authtx.NewTxConfig(appCodec, authtx.DefaultSignModes)

	std.RegisterLegacyAminoCodec(legacyAmino)
	std.RegisterInterfaces(interfaceRegistry)

	baseAppOptions = append(baseAppOptions, baseapp.SetOptimisticExecution())

	bApp := baseapp.NewBaseApp(appName, logger, db, txConfig.TxDecoder(), baseAppOptions...)
	bApp.SetCommitMultiStoreTracer(traceStore)
	bApp.SetVersion(version.Version)
	bApp.SetInterfaceRegistry(interfaceRegistry)
	bApp.SetTxEncoder(txConfig.TxEncoder())

	// TODO missing a key ?
	keys := storetypes.NewKVStoreKeys(
		authtypes.StoreKey,
		banktypes.StoreKey,
		stakingtypes.StoreKey,
		crisistypes.StoreKey,
		minttypes.StoreKey,
		distrtypes.StoreKey,
		slashingtypes.StoreKey,
		govtypes.StoreKey,
		paramstypes.StoreKey,
		consensusparamtypes.StoreKey,
		upgradetypes.StoreKey,
		feegrant.StoreKey,
		evidencetypes.StoreKey,
		circuittypes.StoreKey,
		authzkeeper.StoreKey,
		nftkeeper.StoreKey,
		group.StoreKey,
		// non sdk store keys
		ibcexported.StoreKey, ibctransfertypes.StoreKey,
		ibcwasmtypes.StoreKey, wasmtypes.StoreKey, icahosttypes.StoreKey,
		aatypes.StoreKey, icacontrollertypes.StoreKey, globalfee.StoreKey,
		xiontypes.StoreKey, packetforwardtypes.StoreKey,
		jwktypes.StoreKey, tokenfactorytypes.StoreKey, zktypes.StoreKey, dkimtypes.StoreKey,
	)

	tkeys := storetypes.NewTransientStoreKeys(paramstypes.TStoreKey)

	// register streaming services
	if err := bApp.RegisterStreamingServices(appOpts, keys); err != nil {
		panic(err)
	}

	app := &WasmApp{
		BaseApp:           bApp,
		legacyAmino:       legacyAmino,
		appCodec:          appCodec,
		txConfig:          txConfig,
		interfaceRegistry: interfaceRegistry,
		keys:              keys,
		tkeys:             tkeys,
	}

	app.ParamsKeeper = initParamsKeeper(
		appCodec,
		legacyAmino,
		keys[paramstypes.StoreKey],
		tkeys[paramstypes.TStoreKey],
	)

	// set the BaseApp's parameter store
	app.ConsensusParamsKeeper = consensusparamkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[consensusparamtypes.StoreKey]),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		runtime.EventService{},
	)
	bApp.SetParamStore(app.ConsensusParamsKeeper.ParamsStore)

	// add keepers
	app.AccountKeeper = authkeeper.NewAccountKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[authtypes.StoreKey]),
		authtypes.ProtoBaseAccount,
		maccPerms,
		authcodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix()),
		sdk.GetConfig().GetBech32AccountAddrPrefix(),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		authkeeper.WithUnorderedTransactions(true),
	)
	app.BankKeeper = bankkeeper.NewBaseKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[banktypes.StoreKey]),
		app.AccountKeeper,
		BlockedAddresses(),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		logger,
	)

	// optional: enable sign mode textual by overwriting the default tx config (after setting the bank keeper)
	// enabledSignModes := append(tx.DefaultSignModes, sigtypes.SignMode_SIGN_MODE_TEXTUAL)
	// txConfigOpts := tx.ConfigOptions{
	//	 EnabledSignModes:           enabledSignModes,
	//	 TextualCoinMetadataQueryFn: txmodule.NewBankKeeperCoinMetadataQueryFn(app.BankKeeper),
	// }
	// txConfig, err := tx.NewTxConfigWithOptions(
	// 	 appCodec,
	// 	 txConfigOpts,
	// )
	// if err != nil {
	//	 panic(err)
	// }
	// app.txConfig = txConfig

	app.StakingKeeper = stakingkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[stakingtypes.StoreKey]),
		app.AccountKeeper,
		app.BankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix()),
		authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ConsensusAddrPrefix()),
	)

	app.MintKeeper = mintkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[minttypes.StoreKey]),
		app.StakingKeeper,
		app.AccountKeeper,
		app.BankKeeper,
		authtypes.FeeCollectorName,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		mintkeeper.WithMintFn(xionkeeper.StakedInflationMintFn(
			authtypes.FeeCollectorName,
			minttypes.DefaultInflationCalculationFn,
			app.BankKeeper, app.AccountKeeper, app.StakingKeeper)))

	app.DistrKeeper = distrkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[distrtypes.StoreKey]),
		app.AccountKeeper,
		app.BankKeeper,
		app.StakingKeeper,
		authtypes.FeeCollectorName,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.SlashingKeeper = slashingkeeper.NewKeeper(
		appCodec,
		legacyAmino,
		runtime.NewKVStoreService(keys[slashingtypes.StoreKey]),
		app.StakingKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	invCheckPeriod := cast.ToUint(appOpts.Get(server.FlagInvCheckPeriod))
	app.CrisisKeeper = crisiskeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[crisistypes.StoreKey]),
		invCheckPeriod,
		app.BankKeeper,
		authtypes.FeeCollectorName,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		app.AccountKeeper.AddressCodec(),
	)

	app.FeeGrantKeeper = feegrantkeeper.NewKeeper(appCodec, runtime.NewKVStoreService(keys[feegrant.StoreKey]), app.AccountKeeper)

	// register the staking hooks
	// NOTE: stakingKeeper above is passed by reference, so that it will contain these hooks
	app.StakingKeeper.SetHooks(
		stakingtypes.NewMultiStakingHooks(app.DistrKeeper.Hooks(), app.SlashingKeeper.Hooks()),
	)

	app.CircuitKeeper = circuitkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[circuittypes.StoreKey]),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		app.AccountKeeper.AddressCodec(),
	)
	app.SetCircuitBreaker(&app.CircuitKeeper)

	app.AuthzKeeper = authzkeeper.NewKeeper(
		runtime.NewKVStoreService(keys[authzkeeper.StoreKey]),
		appCodec,
		app.MsgServiceRouter(),
		app.AccountKeeper,
	)
	app.AuthzKeeper = app.AuthzKeeper.SetBankKeeper(app.BankKeeper)

	groupConfig := group.DefaultConfig()
	/*
		Example of setting group params:
		groupConfig.MaxMetadataLen = 1000
	*/
	app.GroupKeeper = groupkeeper.NewKeeper(
		keys[group.StoreKey],
		appCodec,
		app.MsgServiceRouter(),
		app.AccountKeeper,
		groupConfig,
	)

	// get skipUpgradeHeights from the app options
	skipUpgradeHeights := map[int64]bool{}
	for _, h := range cast.ToIntSlice(appOpts.Get(server.FlagUnsafeSkipUpgrades)) {
		skipUpgradeHeights[int64(h)] = true
	}
	homePath := cast.ToString(appOpts.Get(flags.FlagHome))
	// set the governance module account as the authority for conducting upgrades
	app.UpgradeKeeper = upgradekeeper.NewKeeper(
		skipUpgradeHeights,
		runtime.NewKVStoreService(keys[upgradetypes.StoreKey]),
		appCodec,
		homePath,
		app.BaseApp,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.IBCKeeper = ibckeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[ibcexported.StoreKey]),
		app.GetSubspace(ibcexported.ModuleName),
		app.UpgradeKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	// Register the proposal types
	// Deprecated: Avoid adding new handlers, instead use the new proposal flow
	// by granting the governance module the right to execute the message.
	// See: https://docs.cosmos.network/main/modules/gov#proposal-messages
	govRouter := govv1beta1.NewRouter()
	govRouter.AddRoute(govtypes.RouterKey, govv1beta1.ProposalHandler).
		AddRoute(paramproposal.RouterKey, params.NewParamChangeProposalHandler(app.ParamsKeeper))

	govConfig := govtypes.DefaultConfig()
	/*
		Example of setting gov params:
		govConfig.MaxMetadataLen = 10000
	*/
	govKeeper := govkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[govtypes.StoreKey]),
		app.AccountKeeper,
		app.BankKeeper,
		app.StakingKeeper,
		app.DistrKeeper,
		app.MsgServiceRouter(),
		govConfig,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.GovKeeper = *govKeeper.SetHooks(
		govtypes.NewMultiGovHooks( // register the governance hooks
		),
	)

	app.NFTKeeper = nftkeeper.NewKeeper(
		runtime.NewKVStoreService(keys[nftkeeper.StoreKey]),
		appCodec,
		app.AccountKeeper,
		app.BankKeeper,
	)

	// create evidence keeper with router
	evidenceKeeper := evidencekeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[evidencetypes.StoreKey]),
		app.StakingKeeper,
		app.SlashingKeeper,
		app.AccountKeeper.AddressCodec(),
		runtime.ProvideCometInfoService(),
	)
	// If evidence needs to be handled for the app, set routes in router here and seal
	app.EvidenceKeeper = *evidenceKeeper

	app.ZkKeeper = zkkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[zktypes.StoreKey]),
		logger,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	app.DkimKeeper = dkimkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[dkimtypes.StoreKey]),
		logger,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		app.ZkKeeper,
	)

	app.JwkKeeper = jwkkeeper.NewKeeper(
		appCodec,
		keys[jwktypes.StoreKey],
		app.GetSubspace(jwktypes.ModuleName))

	app.TokenFactoryKeeper = tokenfactorykeeper.NewKeeper(
		appCodec,
		keys[tokenfactorytypes.StoreKey],
		maccPerms,
		app.AccountKeeper,
		app.BankKeeper,
		app.DistrKeeper,
		tokenFactoryCapabilities,
		func(_ context.Context, _ string) bool {
			return false
		},
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	// Configure the hooks keeper
	// hooksKeeper := ibchookskeeper.NewKeeper(
	// 	keys[ibchookstypes.StoreKey],
	// )
	// app.IBCHooksKeeper = &hooksKeeper

	// xionPrefix := sdk.GetConfig().GetBech32AccountAddrPrefix()
	// wasmHooks := ibchooks.NewWasmHooks(app.IBCHooksKeeper, nil, xionPrefix) // The contract keeper needs to be set later
	// app.Ics20WasmHooks = &wasmHooks
	// app.HooksICS4Wrapper = ibchooks.NewICS4Middleware(
	// 	app.IBCKeeper.ChannelKeeper,
	// 	app.Ics20WasmHooks,
	// )

	// Initialize packet forward middleware router
	app.PacketForwardKeeper = packetforwardkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[packetforwardtypes.StoreKey]),
		app.TransferKeeper, // Will be zero-value here. Reference is set later on with SetTransferKeeper.
		app.IBCKeeper.ChannelKeeper,
		app.BankKeeper,
		app.IBCKeeper.ChannelKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	// Create Transfer Keepers
	app.TransferKeeper = ibctransferkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[ibctransfertypes.StoreKey]),
		app.GetSubspace(ibctransfertypes.ModuleName),
		app.IBCKeeper.ChannelKeeper,
		app.IBCKeeper.ChannelKeeper,
		app.MsgServiceRouter(),
		app.AccountKeeper,
		app.BankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.PacketForwardKeeper.SetTransferKeeper(app.TransferKeeper)

	app.ICAHostKeeper = icahostkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[icahosttypes.StoreKey]),
		app.GetSubspace(icahosttypes.SubModuleName),
		app.IBCKeeper.ChannelKeeper,
		app.IBCKeeper.ChannelKeeper,
		app.AccountKeeper,
		app.MsgServiceRouter(),
		app.GRPCQueryRouter(), // set grpc router for ica host
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.ICAControllerKeeper = icacontrollerkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[icacontrollertypes.StoreKey]),
		app.GetSubspace(icacontrollertypes.SubModuleName),
		app.IBCKeeper.ChannelKeeper,
		app.IBCKeeper.ChannelKeeper,
		app.MsgServiceRouter(),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	wasmDir := filepath.Join(homePath, "wasm")
	nodeConfig, err := wasm.ReadNodeConfig(appOpts)
	if err != nil {
		panic(fmt.Sprintf("error while reading wasm config: %s", err))
	}

	// The last arguments can contain custom message handlers, and custom query handlers,
	// if we want to allow any custom callbacks
	// See https://github.com/CosmWasm/cosmwasm/blob/main/docs/CAPABILITIES-BUILT-IN.md

	availableCapabilities := append(wasmkeeper.BuiltInCapabilities(), "token_factory")

	tokenFactoryOpts := bindings.RegisterCustomPlugins(app.BankKeeper, &app.TokenFactoryKeeper)
	wasmOpts = append(owasm.RegisterStargateQueries(*app.GRPCQueryRouter(), appCodec), wasmOpts...)
	wasmOpts = append(wasmOpts, tokenFactoryOpts...)

	// instantiate the Wasm VM with the chosen parameters
	// we need to create this double wasm dir because the wasmd Keeper appends an extra `wasm/` to the value you give it
	doubleWasmDir := filepath.Join(wasmDir, "wasm")
	wasmVM, err := wasmvm.NewVM(
		doubleWasmDir,
		availableCapabilities,
		WasmContractMemoryLimit, // default of 32
		nodeConfig.ContractDebugMode,
		nodeConfig.MemoryCacheSize,
	)
	if err != nil {
		panic(err)
	}

	wasmOpts = append(wasmOpts, wasmkeeper.WithWasmEngine(wasmVM))

	app.WasmKeeper = wasmkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[wasmtypes.StoreKey]),
		app.AccountKeeper,
		app.BankKeeper,
		app.StakingKeeper,
		distrkeeper.NewQuerier(app.DistrKeeper),
		app.IBCKeeper.ChannelKeeper,
		app.IBCKeeper.ChannelKeeper,
		app.IBCKeeper.ChannelKeeperV2,
		app.TransferKeeper,
		app.MsgServiceRouter(),
		app.GRPCQueryRouter(),
		wasmDir,
		nodeConfig,
		wasmtypes.VMConfig{},
		availableCapabilities,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		wasmOpts...,
	)

	// Create fee enabled wasm ibc Stack
	// var wasmStackIBCHandler porttypes.IBCModule
	wasmStackIBCHandler := wasm.NewIBCHandler(
		app.WasmKeeper,              // IBCContractKeeper
		app.IBCKeeper.ChannelKeeper, // ChannelKeeper
		app.TransferKeeper,          // ICS20TransferPortSource
		app.IBCKeeper.ChannelKeeper, // appVersionGetter (or provide an implementation if needed)
	)

	app.WasmClientKeeper = ibcwasmkeeper.NewKeeperWithVM(
		appCodec,
		runtime.NewKVStoreService(keys[ibcwasmtypes.StoreKey]),
		app.IBCKeeper.ClientKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		wasmVM,
		app.GRPCQueryRouter(),
	)

	app.AbstractAccountKeeper = aakeeper.NewKeeper(
		appCodec,
		keys[aatypes.StoreKey],
		app.AccountKeeper,
		wasmkeeper.NewGovPermissionKeeper(app.WasmKeeper),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.ContractKeeper = wasmkeeper.NewDefaultPermissionKeeper(app.WasmKeeper)
	// app.Ics20WasmHooks.ContractKeeper = &app.WasmKeeper

	app.XionKeeper = xionkeeper.NewKeeper(
		appCodec,
		keys[xiontypes.StoreKey],
		app.GetSubspace(xiontypes.ModuleName),
		app.BankKeeper,
		app.AccountKeeper,
		app.ContractKeeper,
		app.WasmKeeper,
		app.AbstractAccountKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	// Set legacy router for backwards compatibility with gov v1beta1
	app.GovKeeper.SetLegacyRouter(govRouter)

	// Create Interchain Accounts Stack
	// SendPacket, since it is originating from the application to core IBC:
	// icaAuthModuleKeeper.SendTx -> icaController.SendPacket -> fee.SendPacket -> channel.SendPacket
	// var icaControllerStack porttypes.IBCModule
	// integration point for custom authentication modules
	// see https://medium.com/the-interchain-foundation/ibc-go-v6-changes-to-interchain-accounts-and-how-it-impacts-your-chain-806c185300d7
	icaControllerStack := icacontroller.NewIBCMiddleware(app.ICAControllerKeeper)

	// RecvPacket, message that originates from core IBC and goes down to app, the flow is:
	// channel.RecvPacket -> fee.OnRecvPacket -> icaHost.OnRecvPacket
	// var icaHostStack porttypes.IBCModule
	icaHostStack := icahost.NewIBCModule(app.ICAHostKeeper)

	// Create Transfer Stack
	var transferStack porttypes.IBCModule
	transferStack = transfer.NewIBCModule(app.TransferKeeper)
	// transferStack = ibchooks.NewIBCMiddleware(transferStack, &app.HooksICS4Wrapper)
	// callbacks wraps the transfer stack as its base app, and uses PacketForwardKeeper as the ICS4Wrapper
	// i.e. packet-forward-middleware is higher on the stack and sits between callbacks and the ibc channel keeper
	// Since this is the lowest level middleware of the transfer stack, it should be the first entrypoint for transfer keeper's
	// WriteAcknowledgement.
	cbStack := ibccallbacks.NewIBCMiddleware(transferStack, app.PacketForwardKeeper, wasmStackIBCHandler, wasm.DefaultMaxIBCCallbackGas)
	// Wire the transfer keeper's ICS4 wrapper through the callbacks middleware so that
	// SendPacket validation (e.g. src_callback schema checks) is consistent with the
	// timeout/ack path. Without this, the send path bypasses callbacks validation.
	app.TransferKeeper.WithICS4Wrapper(cbStack)
	transferStack = packetforward.NewIBCMiddleware(
		cbStack,
		app.PacketForwardKeeper,
		10,
		packetforwardkeeper.DefaultForwardTransferPacketTimeoutTimestamp,
	)

	// Create static IBC router, add app routes, then set and seal it
	ibcRouter := porttypes.NewRouter().
		AddRoute(ibctransfertypes.ModuleName, transferStack).
		AddRoute(wasmtypes.ModuleName, wasmStackIBCHandler).
		AddRoute(icacontrollertypes.SubModuleName, icaControllerStack).
		AddRoute(icahosttypes.SubModuleName, icaHostStack)
	app.IBCKeeper.SetRouter(ibcRouter)

	// IBC v2 transfer stack
	ibcv2TransferStack := ibccallbacksv2.NewIBCMiddleware(
		transferv2.NewIBCModule(app.TransferKeeper),
		app.IBCKeeper.ChannelKeeperV2,
		wasmStackIBCHandler,
		app.IBCKeeper.ChannelKeeperV2,
		wasm.DefaultMaxIBCCallbackGas,
	)

	ibcRouterV2 := ibcapi.NewRouter().
		AddRoute(ibctransfertypes.PortID, ibcv2TransferStack)
	app.IBCKeeper.SetRouterV2(ibcRouterV2)

	clientKeeper := app.IBCKeeper.ClientKeeper
	storeProvider := app.IBCKeeper.ClientKeeper.GetStoreProvider()

	tmLightClientModule := ibctm.NewLightClientModule(appCodec, storeProvider)
	clientKeeper.AddRoute(ibctm.ModuleName, &tmLightClientModule)

	wasmLightClientModule := ibcwasm.NewLightClientModule(app.WasmClientKeeper, storeProvider)
	clientKeeper.AddRoute(ibcwasmtypes.ModuleName, &wasmLightClientModule)

	/****  Module Options ****/

	// NOTE: we may consider parsing `appOpts` inside module constructors. For the moment
	// we prefer to be more strict in what arguments the modules expect.
	skipGenesisInvariants := cast.ToBool(appOpts.Get(crisis.FlagSkipGenesisInvariants))

	// NOTE: Any module instantiated in the module manager that is later modified
	// must be passed by reference here.
	app.ModuleManager = module.NewManager(
		genutil.NewAppModule(
			app.AccountKeeper,
			app.StakingKeeper,
			app,
			txConfig,
		),
		auth.NewAppModule(appCodec, app.AccountKeeper, authsims.RandomGenesisAccounts, app.GetSubspace(authtypes.ModuleName)),
		vesting.NewAppModule(app.AccountKeeper, app.BankKeeper),
		bank.NewAppModule(appCodec, app.BankKeeper, app.AccountKeeper, app.GetSubspace(banktypes.ModuleName)),
		feegrantmodule.NewAppModule(appCodec, app.AccountKeeper, app.BankKeeper, app.FeeGrantKeeper, app.interfaceRegistry),
		gov.NewAppModule(appCodec, &app.GovKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(govtypes.ModuleName)),
		mint.NewAppModule(appCodec, app.MintKeeper, app.AccountKeeper, nil, app.GetSubspace(minttypes.ModuleName)),
		slashing.NewAppModule(appCodec, app.SlashingKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper, app.GetSubspace(slashingtypes.ModuleName), app.interfaceRegistry),
		distr.NewAppModule(appCodec, app.DistrKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper, app.GetSubspace(distrtypes.ModuleName)),
		staking.NewAppModule(appCodec, app.StakingKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(stakingtypes.ModuleName)),
		upgrademodule.NewAppModule(app.UpgradeKeeper, app.AccountKeeper.AddressCodec()),
		evidence.NewAppModule(app.EvidenceKeeper),
		params.NewAppModule(app.ParamsKeeper),
		authzmodule.NewAppModule(appCodec, app.AuthzKeeper, app.AccountKeeper, app.BankKeeper, app.interfaceRegistry),
		groupmodule.NewAppModule(appCodec, app.GroupKeeper, app.AccountKeeper, app.BankKeeper, app.interfaceRegistry),
		nftmodule.NewAppModule(appCodec, app.NFTKeeper, app.AccountKeeper, app.BankKeeper, app.interfaceRegistry),
		consensus.NewAppModule(appCodec, app.ConsensusParamsKeeper),
		circuit.NewAppModule(appCodec, app.CircuitKeeper),
		// non sdk modules
		tokenfactory.NewAppModule(app.TokenFactoryKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(tokenfactorytypes.ModuleName)),
		jwk.NewAppModule(appCodec, app.JwkKeeper, app.GetSubspace(jwktypes.ModuleName)),
		globalfee.NewAppModule(app.GetSubspace(globalfee.ModuleName)),
		wasm.NewAppModule(appCodec, &app.WasmKeeper, app.StakingKeeper, app.AccountKeeper, app.BankKeeper, app.MsgServiceRouter(), app.GetSubspace(wasmtypes.ModuleName)),
		aa.NewAppModule(app.AbstractAccountKeeper),
		xion.NewAppModule(app.XionKeeper),
		ibc.NewAppModule(app.IBCKeeper),
		ibctm.NewAppModule(tmLightClientModule),
		ibcwasm.NewAppModule(app.WasmClientKeeper),
		transfer.NewAppModule(app.TransferKeeper),
		ica.NewAppModule(&app.ICAControllerKeeper, &app.ICAHostKeeper),
		// ibchooks.NewAppModule(app.AccountKeeper),
		packetforward.NewAppModule(app.PacketForwardKeeper, app.GetSubspace(packetforwardtypes.ModuleName)),
		crisis.NewAppModule(app.CrisisKeeper, skipGenesisInvariants, app.GetSubspace(crisistypes.ModuleName)), // always be last to make sure that it checks for all invariants and not only part of them
		zk.NewAppModule(appCodec, app.ZkKeeper),
		dkim.NewAppModule(appCodec, app.DkimKeeper),
	)

	// BasicModuleManager defines the module BasicManager is in charge of setting up basic,
	// non-dependant module elements, such as codec registration and genesis verification.
	// By default, it is composed of all the module from the module manager.
	// Additionally, app module basics can be overwritten by passing them as argument.
	app.BasicModuleManager = module.NewBasicManagerFromManager(
		app.ModuleManager,
		map[string]module.AppModuleBasic{
			genutiltypes.ModuleName: genutil.NewAppModuleBasic(genutiltypes.DefaultMessageValidator),
			govtypes.ModuleName: gov.NewAppModuleBasic(
				[]govclient.ProposalHandler{
					paramsclient.ProposalHandler,
				},
			),
		})
	app.BasicModuleManager.RegisterLegacyAminoCodec(legacyAmino)
	app.BasicModuleManager.RegisterInterfaces(interfaceRegistry)

	// NOTE: upgrade module is required to be prioritized
	app.ModuleManager.SetOrderPreBlockers(
		upgradetypes.ModuleName,
		authtypes.ModuleName,
	)
	// During begin block slashing happens after distr.BeginBlocker so that
	// there is nothing left over in the validator fee pool, to keep the
	// CanWithdrawInvariant invariant.
	// NOTE: staking module is required if HistoricalEntries param > 0
	// NOTE: capability module's beginblocker must come before any modules using capabilities (e.g. IBC)
	app.ModuleManager.SetOrderBeginBlockers(
		minttypes.ModuleName, distrtypes.ModuleName, slashingtypes.ModuleName,
		evidencetypes.ModuleName, stakingtypes.ModuleName, authtypes.ModuleName,
		banktypes.ModuleName, govtypes.ModuleName, crisistypes.ModuleName,
		genutiltypes.ModuleName, authz.ModuleName, feegrant.ModuleName,
		nft.ModuleName, group.ModuleName, paramstypes.ModuleName,
		vestingtypes.ModuleName, consensusparamtypes.ModuleName,
		tokenfactorytypes.ModuleName,
		globalfee.ModuleName,
		jwktypes.ModuleName,
		// additional non simd modules
		ibctransfertypes.ModuleName,
		ibcexported.ModuleName,
		icatypes.ModuleName,
		ibcwasmtypes.ModuleName,
		wasmtypes.ModuleName,
		aatypes.ModuleName,
		xiontypes.ModuleName,
		// ibchookstypes.ModuleName,
		packetforwardtypes.ModuleName,
		dkimtypes.ModuleName,
	)

	app.ModuleManager.SetOrderEndBlockers(
		crisistypes.ModuleName, govtypes.ModuleName, stakingtypes.ModuleName,
		authtypes.ModuleName, banktypes.ModuleName,
		distrtypes.ModuleName, slashingtypes.ModuleName, minttypes.ModuleName,
		genutiltypes.ModuleName, evidencetypes.ModuleName, authz.ModuleName,
		feegrant.ModuleName, nft.ModuleName, group.ModuleName,
		paramstypes.ModuleName, upgradetypes.ModuleName, vestingtypes.ModuleName,
		consensusparamtypes.ModuleName,
		tokenfactorytypes.ModuleName,
		globalfee.ModuleName,
		xiontypes.ModuleName,
		jwktypes.ModuleName,
		// additional non simd modules
		ibctransfertypes.ModuleName,
		ibcexported.ModuleName,
		icatypes.ModuleName,
		ibcwasmtypes.ModuleName,
		wasmtypes.ModuleName,
		aatypes.ModuleName,
		// ibchookstypes.ModuleName,
		packetforwardtypes.ModuleName,
		dkimtypes.ModuleName,
	)

	// NOTE: The genutils module must occur after staking so that pools are
	// properly initialized with tokens from genesis accounts.
	// NOTE: The genutils module must also occur after auth so that it can access the params from auth.
	// NOTE: Capability module must occur first so that it can initialize any capabilities
	// so that other modules that want to create or claim capabilities afterwards in InitChain
	// can do so safely.
	// NOTE: wasm module should be at the end as it can call other module functionality direct or via message dispatching during
	// genesis phase. For example bank transfer, auth account check, staking, ...
	genesisModuleOrder := []string{
		authtypes.ModuleName, banktypes.ModuleName,
		distrtypes.ModuleName, stakingtypes.ModuleName, slashingtypes.ModuleName,
		govtypes.ModuleName, minttypes.ModuleName, crisistypes.ModuleName,
		genutiltypes.ModuleName, evidencetypes.ModuleName, authz.ModuleName,
		feegrant.ModuleName, nft.ModuleName, group.ModuleName,
		paramstypes.ModuleName, upgradetypes.ModuleName, vestingtypes.ModuleName,
		consensusparamtypes.ModuleName,
		circuittypes.ModuleName,
		// additional non simd modules
		tokenfactorytypes.ModuleName,
		globalfee.ModuleName, xiontypes.ModuleName,
		jwktypes.ModuleName,
		ibctransfertypes.ModuleName,
		ibcexported.ModuleName,
		icatypes.ModuleName,
		ibcwasmtypes.ModuleName,
		// wasm after ibc transfer
		wasmtypes.ModuleName,
		aatypes.ModuleName,
		// ibchookstypes.ModuleName,
		packetforwardtypes.ModuleName,
		zktypes.ModuleName,
		dkimtypes.ModuleName,
	}
	app.ModuleManager.SetOrderInitGenesis(genesisModuleOrder...)
	app.ModuleManager.SetOrderExportGenesis(genesisModuleOrder...)

	// Uncomment if you want to set a custom migration order here.
	// app.ModuleManager.SetOrderMigrations(custom order)

	app.ModuleManager.RegisterInvariants(app.CrisisKeeper)
	app.configurator = module.NewConfigurator(app.appCodec, app.MsgServiceRouter(), app.GRPCQueryRouter())
	err = app.ModuleManager.RegisterServices(app.configurator)
	if err != nil {
		panic(err)
	}

	// Configure Indexer
	indexerConfig := indexer.NewConfigFromOptions(appOpts)
	services := []storetypes.ABCIListener{}
	if indexerConfig.Enabled {
		app.indexerService = indexer.New(homePath, app.appCodec, authcodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix()), app.Logger())
		if err = app.indexerService.RegisterServices(app.configurator); err != nil {
			// Log the error but don't panic - indexer is not consensus-critical
			app.Logger().Error("Failed to register indexer services", "error", err)
		}

		// Add listeners to commitmultistore
		// otherwise the ABCILister attached to the streammanager
		// will receive block information but empty []ChangeSet
		listenKeys := []storetypes.StoreKey{
			keys[feegrant.StoreKey],
			keys[authzkeeper.StoreKey],
		}
		app.CommitMultiStore().AddListeners(listenKeys)
		services = append(services, app.indexerService)
	}

	streamManager := storetypes.StreamingManager{
		ABCIListeners: services,
		StopNodeOnErr: false, // Changed from true to prevent indexer errors from halting the node
	}
	// attach stream manager
	app.SetStreamingManager(streamManager)

	// RegisterUpgradeHandlers is used for registering any on-chain upgrades.
	// Make sure it's called after `app.ModuleManager` and `app.configurator` are set.
	app.RegisterUpgradeHandlers()

	autocliv1.RegisterQueryServer(app.GRPCQueryRouter(), runtimeservices.NewAutoCLIQueryService(app.ModuleManager.Modules))

	reflectionSvc, err := runtimeservices.NewReflectionService()
	if err != nil {
		panic(err)
	}
	reflectionv1.RegisterReflectionServiceServer(app.GRPCQueryRouter(), reflectionSvc)

	// add test gRPC service for testing gRPC queries in isolation
	// testdata_pulsar.RegisterQueryServer(app.GRPCQueryRouter(), testdata_pulsar.QueryImpl{})

	// create the simulation manager and define the order of the modules for deterministic simulations
	//
	// NOTE: this is not required apps that don't use the simulator for fuzz testing
	// transactions
	overrideModules := map[string]module.AppModuleSimulation{
		authtypes.ModuleName: auth.NewAppModule(app.appCodec, app.AccountKeeper, authsims.RandomGenesisAccounts, app.GetSubspace(authtypes.ModuleName)),
	}
	app.sm = module.NewSimulationManagerFromAppModules(app.ModuleManager.Modules, overrideModules)

	app.sm.RegisterStoreDecoders()

	// initialize stores
	app.MountKVStores(keys)
	app.MountTransientStores(tkeys)

	// initialize BaseApp
	app.SetInitChainer(app.InitChainer)
	app.SetPreBlocker(app.PreBlocker)
	app.SetBeginBlocker(app.BeginBlocker)
	app.SetEndBlocker(app.EndBlocker)
	app.setAnteHandler(txConfig, nodeConfig, keys[wasmtypes.StoreKey])

	// Create rocksdb fork checkpoints every N blocks (controlled by XION_CHECKPOINT_INTERVAL).
	// If the env var is empty or unset, checkpointing is disabled.
	if cpInterval, _ := strconv.ParseInt(os.Getenv("XION_CHECKPOINT_INTERVAL"), 10, 64); cpInterval > 0 {
		if cp, ok := db.(dbm.Checkpointable); ok {
			logger.Info("fork checkpointing enabled", "interval", cpInterval)
			app.SetPrepareCheckStater(func(ctx sdk.Context) {
				height := ctx.BlockHeight()
				if height%cpInterval == 0 {
					cpDir := filepath.Join(homePath, "data", "checkpoints", fmt.Sprintf("block_%d", height))

					// Checkpoint application.db (app state — uses hardlinks, very cheap)
					appCpPath := filepath.Join(cpDir, "application.db")
					if err := cp.CreateCheckpoint(appCpPath); err != nil {
						logger.Error("failed to checkpoint application.db", "height", height, "error", err)
						return
					}

					// Build minimal state.db with only the keys needed to bootstrap from this height
					if err := buildMinimalStateDB(homePath, cpDir, height); err != nil {
						logger.Error("failed to build minimal state.db", "height", height, "error", err)
						return
					}

					// Build minimal blockstore.db with only the last block's data
					if err := buildMinimalBlockStoreDB(homePath, cpDir, height); err != nil {
						logger.Error("failed to build minimal blockstore.db", "height", height, "error", err)
						return
					}

					// Write a reset priv_validator_state.json so the fork can start signing
					pvStatePath := filepath.Join(cpDir, "priv_validator_state.json")
					if err := os.WriteFile(pvStatePath, []byte(`{"height":"0","round":0,"step":0}`), 0o644); err != nil {
						logger.Error("failed to write priv_validator_state.json", "height", height, "error", err)
						return
					}

					logger.Info("created fork checkpoint", "height", height, "path", cpDir)
				}
			})
		}
	}

	// must be before Loading version
	// requires the snapshot store to be created and registered as a BaseAppOption
	// see cmd/xiond/root.go: 206 - 214 approx
	if manager := app.SnapshotManager(); manager != nil {
		err := manager.RegisterExtensions(
			wasmkeeper.NewWasmSnapshotter(app.CommitMultiStore(), &app.WasmKeeper),
		)
		if err != nil {
			panic(fmt.Errorf("failed to register snapshot extension: %s", err))
		}
	}

	// set the contract keeper for the Ics20WasmHooks
	app.ContractKeeper = wasmkeeper.NewDefaultPermissionKeeper(app.WasmKeeper)
	// app.Ics20WasmHooks.ContractKeeper = &app.WasmKeeper

	// In v0.46, the SDK introduces _postHandlers_. PostHandlers are like
	// antehandlers, but are run _after_ the `runMsgs` execution. They are also
	// defined as a chain, and have the same signature as antehandlers.
	//
	// In baseapp, postHandlers are run in the same store branch as `runMsgs`,
	// meaning that both `runMsgs` and `postHandler` state will be committed if
	// both are successful, and both will be reverted if any of the two fails.
	//
	// The SDK exposes a default postHandlers chain, which comprises of only
	// one decorator: the Transaction Tips decorator. However, some chains do
	// not need it by default, so feel free to comment the next line if you do
	// not need tips.
	// To read more about tips:
	// https://docs.cosmos.network/main/core/tips.html
	//
	// Please note that changing any of the anteHandler or postHandler chain is
	// likely to be a state-machine breaking change, which needs a coordinated
	// upgrade.
	app.setPostHandler()

	if loadLatest {
		if err := app.LoadLatestVersion(); err != nil {
			logger.Error("error on loading last version", "err", err)
			os.Exit(1)
		}
		ctx := app.NewUncachedContext(true, cmtproto.Header{})

		// Initialize pinned codes in wasmvm as they are not persisted there
		if err := app.WasmKeeper.InitializePinnedCodes(ctx); err != nil {
			tmos.Exit(fmt.Sprintf("failed to initialize pinned codes %s", err))
		}
	}
	return app
}

func (app *WasmApp) setAnteHandler(txConfig client.TxConfig, nodeConfig wasmtypes.NodeConfig, txCounterStoreKey *storetypes.KVStoreKey) {
	anteHandler, err := NewAnteHandler(
		HandlerOptions{
			HandlerOptions: ante.HandlerOptions{
				AccountKeeper:   app.AccountKeeper,
				BankKeeper:      app.BankKeeper,
				SignModeHandler: txConfig.SignModeHandler(),
				FeegrantKeeper:  app.FeeGrantKeeper,
				SigGasConsumer:  aa.SigVerificationGasConsumer,
			},

			AbstractAccountKeeper: app.AbstractAccountKeeper,
			IBCKeeper:             app.IBCKeeper,
			NodeConfig:            &nodeConfig,
			TXCounterStoreService: runtime.NewKVStoreService(txCounterStoreKey),
			GlobalFeeSubspace:     app.GetSubspace(globalfee.ModuleName),
			StakingKeeper:         app.StakingKeeper,
			CircuitKeeper:         &app.CircuitKeeper,
		},
	)
	if err != nil {
		panic(fmt.Errorf("failed to create AnteHandler: %s", err))
	}
	app.SetAnteHandler(anteHandler)
}

func (app *WasmApp) setPostHandler() {
	postHandler, err := NewPostHandler(
		PostHandlerOptions{
			HandlerOptions:        posthandler.HandlerOptions{},
			AccountKeeper:         app.AccountKeeper,
			AbstractAccountKeeper: app.AbstractAccountKeeper,
		},
	)
	if err != nil {
		panic(err)
	}

	app.SetPostHandler(postHandler)
}

// Name returns the name of the App
func (app *WasmApp) Name() string { return app.BaseApp.Name() }

// PreBlocker application updates every pre block
func (app *WasmApp) PreBlocker(ctx sdk.Context, _ *abci.RequestFinalizeBlock) (*sdk.ResponsePreBlock, error) {
	return app.ModuleManager.PreBlock(ctx)
}

// BeginBlocker application updates every begin block
func (app *WasmApp) BeginBlocker(ctx sdk.Context) (sdk.BeginBlock, error) {
	// SECURITY: Add panic recovery to prevent network shutdown from malicious WASM contracts
	// that panic in their begin_block entry points (CVE-2025-WASM-PANIC)
	defer func() {
		if r := recover(); r != nil {
			ctx.Logger().Error(
				"Recovered from panic in BeginBlocker - potential malicious contract attack",
				"panic", r,
				"stack", string(debug.Stack()),
			)
			// Continue execution instead of crashing the validator
		}
	}()

	return app.ModuleManager.BeginBlock(ctx)
}

// EndBlocker application updates every end block
func (app *WasmApp) EndBlocker(ctx sdk.Context) (sdk.EndBlock, error) {
	return app.ModuleManager.EndBlock(ctx)
}

func (app *WasmApp) Configurator() module.Configurator {
	return app.configurator
}

// InitChainer application update at chain initialization
func (app *WasmApp) InitChainer(ctx sdk.Context, req *abci.RequestInitChain) (*abci.ResponseInitChain, error) {
	var genesisState GenesisState
	if err := json.Unmarshal(req.AppStateBytes, &genesisState); err != nil {
		panic(err)
	}
	err := app.UpgradeKeeper.SetModuleVersionMap(ctx, app.ModuleManager.GetVersionMap())
	if err != nil {
		panic(err)
	}
	response, err := app.ModuleManager.InitGenesis(ctx, app.appCodec, genesisState)
	return response, err
}

// LoadHeight loads a particular height
func (app *WasmApp) LoadHeight(height int64) error {
	return app.LoadVersion(height)
}

// LegacyAmino returns legacy amino codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *WasmApp) LegacyAmino() *codec.LegacyAmino {
	return app.legacyAmino
}

// AppCodec returns app codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *WasmApp) AppCodec() codec.Codec {
	return app.appCodec
}

// InterfaceRegistry returns WasmApp's InterfaceRegistry
func (app *WasmApp) InterfaceRegistry() types.InterfaceRegistry {
	return app.interfaceRegistry
}

// TxConfig returns WasmApp's TxConfig
func (app *WasmApp) TxConfig() client.TxConfig {
	return app.txConfig
}

// DefaultGenesis returns a default genesis from the registered AppModuleBasic's.
func (app *WasmApp) DefaultGenesis() map[string]json.RawMessage {
	return app.BasicModuleManager.DefaultGenesis(app.appCodec)
}

// GetKey returns the KVStoreKey for the provided store key.
//
// NOTE: This is solely to be used for testing purposes.
func (app *WasmApp) GetKey(storeKey string) *storetypes.KVStoreKey {
	return app.keys[storeKey]
}

// GetTKey returns the TransientStoreKey for the provided store key.
//
// NOTE: This is solely to be used for testing purposes.
func (app *WasmApp) GetTKey(storeKey string) *storetypes.TransientStoreKey {
	return app.tkeys[storeKey]
}

// GetSubspace returns a param subspace for a given module name.
//
// NOTE: This is solely to be used for testing purposes.
func (app *WasmApp) GetSubspace(moduleName string) paramstypes.Subspace {
	subspace, _ := app.ParamsKeeper.GetSubspace(moduleName)
	return subspace
}

// SimulationManager implements the SimulationApp interface
func (app *WasmApp) SimulationManager() *module.SimulationManager {
	return app.sm
}

// RegisterAPIRoutes registers all application module routes with the provided
// API server.
func (app *WasmApp) RegisterAPIRoutes(apiSvr *api.Server, apiConfig config.APIConfig) {
	clientCtx := apiSvr.ClientCtx
	// Register new tx routes from grpc-gateway.
	authtx.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// Register new tendermint queries routes from grpc-gateway.
	cmtservice.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// Register node gRPC service for grpc-gateway.
	nodeservice.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// Register grpc-gateway routes for all modules.
	app.BasicModuleManager.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// Register indexer service routes
	if app.indexerService != nil {
		app.indexerService.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)
	}

	// register swagger API from root so that other applications can override easily
	if err := RegisterSwaggerAPI(clientCtx, apiSvr.Router, apiConfig.Swagger); err != nil {
		panic(err)
	}
}

// RegisterTxService implements the Application.RegisterTxService method.
func (app *WasmApp) RegisterTxService(clientCtx client.Context) {
	authtx.RegisterTxService(app.GRPCQueryRouter(), clientCtx, app.Simulate, app.interfaceRegistry)
}

// RegisterTendermintService implements the Application.RegisterTendermintService method.
func (app *WasmApp) RegisterTendermintService(clientCtx client.Context) {
	cmtservice.RegisterTendermintService(
		clientCtx,
		app.GRPCQueryRouter(),
		app.interfaceRegistry,
		app.Query,
	)
}

func (app *WasmApp) RegisterNodeService(clientCtx client.Context, cfg config.Config) {
	nodeservice.RegisterNodeService(clientCtx, app.GRPCQueryRouter(), cfg)
}

// Close wraps BaseApp Close() to
// perform graceful shutdown of our own services.
func (app *WasmApp) Close() error {
	var errs []error

	err := app.BaseApp.Close()
	if err != nil {
		errs = append(errs, err)
	}

	if app.indexerService != nil {
		err = app.indexerService.Close()
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (app *WasmApp) IndexerService() *indexer.StreamService {
	return app.indexerService
}

// GetMaccPerms returns a copy of the module account permissions
//
// NOTE: This is solely to be used for testing purposes.
func GetMaccPerms() map[string][]string {
	dupMaccPerms := make(map[string][]string)
	for k, v := range maccPerms {
		dupMaccPerms[k] = v
	}

	return dupMaccPerms
}

// BlockedAddresses returns all the app's blocked account addresses.
func BlockedAddresses() map[string]bool {
	modAccAddrs := make(map[string]bool)
	for acc := range GetMaccPerms() {
		modAccAddrs[authtypes.NewModuleAddress(acc).String()] = true
	}

	// allow the following addresses to receive funds
	delete(modAccAddrs, authtypes.NewModuleAddress(govtypes.ModuleName).String())

	return modAccAddrs
}

// initParamsKeeper init params keeper and its subspaces
func initParamsKeeper(appCodec codec.BinaryCodec, legacyAmino *codec.LegacyAmino, key, tkey storetypes.StoreKey) paramskeeper.Keeper {
	paramsKeeper := paramskeeper.NewKeeper(appCodec, legacyAmino, key, tkey)

	paramsKeeper.Subspace(authtypes.ModuleName)
	paramsKeeper.Subspace(banktypes.ModuleName)
	paramsKeeper.Subspace(stakingtypes.ModuleName)
	paramsKeeper.Subspace(minttypes.ModuleName)
	paramsKeeper.Subspace(distrtypes.ModuleName)
	paramsKeeper.Subspace(slashingtypes.ModuleName)
	paramsKeeper.Subspace(govtypes.ModuleName)
	paramsKeeper.Subspace(crisistypes.ModuleName)
	paramsKeeper.Subspace(tokenfactorytypes.ModuleName)
	paramsKeeper.Subspace(globalfee.ModuleName)
	paramsKeeper.Subspace(xiontypes.ModuleName)
	paramsKeeper.Subspace(jwktypes.ModuleName)
	paramsKeeper.Subspace(wasmtypes.ModuleName)
	paramsKeeper.Subspace(aatypes.ModuleName)
	paramsKeeper.Subspace(packetforwardtypes.ModuleName)
	paramsKeeper.Subspace(ibcwasmtypes.ModuleName)
	paramsKeeper.Subspace(zktypes.ModuleName)
	paramsKeeper.Subspace(dkimtypes.ModuleName)

	// IBC params migration - legacySubspace to selfManaged
	// https://github.com/cosmos/ibc-go/blob/main/docs/docs/05-migrations/11-v7-to-v10.md#params-migration
	keyTable := ibcclienttypes.ParamKeyTable()
	keyTable.RegisterParamSet(&ibcconnectiontypes.Params{})
	paramsKeeper.Subspace(ibcexported.ModuleName).WithKeyTable(keyTable)
	paramsKeeper.Subspace(ibctransfertypes.ModuleName).WithKeyTable(ibctransfertypes.ParamKeyTable())
	paramsKeeper.Subspace(icacontrollertypes.SubModuleName).WithKeyTable(icacontrollertypes.ParamKeyTable())
	paramsKeeper.Subspace(icahosttypes.SubModuleName).WithKeyTable(icahosttypes.ParamKeyTable())

	return paramsKeeper
}

// AutoCliOpts returns the autocli options for the app.
func (app *WasmApp) AutoCliOpts() autocli.AppOptions {
	modules := make(map[string]appmodule.AppModule, 0)
	for _, m := range app.ModuleManager.Modules {
		if moduleWithName, ok := m.(module.HasName); ok {
			moduleName := moduleWithName.Name()
			if appModule, ok := moduleWithName.(appmodule.AppModule); ok {
				modules[moduleName] = appModule
			}
		}
	}

	return autocli.AppOptions{
		Modules:               modules,
		ModuleOptions:         runtimeservices.ExtractAutoCLIOptions(app.ModuleManager.Modules),
		AddressCodec:          authcodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix()),
		ValidatorAddressCodec: authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix()),
		ConsensusAddressCodec: authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ConsensusAddrPrefix()),
	}
}

// buildMinimalStateDB constructs a minimal CometBFT state.db for bootstrapping
// a forked chain from the given checkpoint height.
//
// CometBFT's state.db normally accumulates entries across all heights:
//
//	stateKey                    → main consensus state (validators, app hash, chain ID, etc.)
//	validatorsKey:{height}      → validator set or pointer to LastHeightChanged
//	consensusParamsKey:{height} → consensus params or pointer to LastHeightChanged
//	lastABCIResponseKey         → most recent FinalizeBlock response (crash recovery)
//
// A full checkpoint of state.db would include all historical entries. Instead,
// this function creates a new rocksdb containing only the keys required to
// continue consensus from the checkpoint height:
//
//  1. stateKey — patched so that LastHeightValidatorsChanged and
//     LastHeightConsensusParamsChanged point to the checkpoint height (not genesis).
//     This ensures new ValidatorsInfo entries written by CometBFT after the fork
//     reference our checkpoint height (where the full set is stored) instead of
//     looking back to the original chain's genesis.
//
//  2. validatorsKey for heights {H-1, H, H+1} — CometBFT needs:
//     - H-1: LastValidators (used to validate block H's LastCommit signatures)
//     - H:   Validators (current proposer selection)
//     - H+1: NextValidators (for the upcoming block)
//     Each entry is "resolved": if the source entry only contains a
//     LastHeightChanged pointer (no inline ValidatorSet), we follow the pointer,
//     read the full set, and write a self-contained entry with
//     LastHeightChanged = H. This breaks the dependency chain back to genesis.
//
//  3. consensusParamsKey for height H — resolved the same way as validators.
//
//  4. lastABCIResponseKey — copied verbatim for crash recovery.
func buildMinimalStateDB(homePath, cpDir string, height int64) error {
	srcPath := filepath.Join(homePath, "data", "state.db")
	dstPath := filepath.Join(cpDir, "state.db")

	if err := os.MkdirAll(dstPath, 0o755); err != nil {
		return fmt.Errorf("mkdir state.db: %w", err)
	}

	// Open source read-only
	roOpts := grocksdb.NewDefaultOptions()
	src, err := grocksdb.OpenDbForReadOnly(roOpts, srcPath, false)
	if err != nil {
		return fmt.Errorf("open source state.db: %w", err)
	}
	defer src.Close()

	// Open destination for writing
	wOpts := grocksdb.NewDefaultOptions()
	wOpts.SetCreateIfMissing(true)
	dst, err := grocksdb.OpenDb(wOpts, dstPath)
	if err != nil {
		return fmt.Errorf("open dest state.db: %w", err)
	}
	defer dst.Close()

	readOpts := grocksdb.NewDefaultReadOptions()
	writeOpts := grocksdb.NewDefaultWriteOptions()

	// Helper to copy a key directly from src to dst
	copyKey := func(key []byte) error {
		val, err := src.Get(readOpts, key)
		if err != nil {
			return fmt.Errorf("read key %s: %w", key, err)
		}
		defer val.Free()
		if val.Data() != nil {
			return dst.Put(writeOpts, key, val.Data())
		}
		return nil
	}

	// Helper to resolve a ValidatorsInfo: if it has no inline ValidatorSet,
	// follow LastHeightChanged to find the actual set, then write a
	// self-contained entry with LastHeightChanged = height itself.
	resolveAndWriteValidators := func(h int64) error {
		key := []byte(fmt.Sprintf("validatorsKey:%v", h))
		val, err := src.Get(readOpts, key)
		if err != nil {
			return fmt.Errorf("read %s: %w", key, err)
		}
		defer val.Free()
		if val.Data() == nil {
			return nil // no validators at this height, skip
		}

		var vi cmtstate.ValidatorsInfo
		if err := vi.Unmarshal(val.Data()); err != nil {
			return fmt.Errorf("unmarshal %s: %w", key, err)
		}

		// If ValidatorSet is nil, it means "same as LastHeightChanged" — resolve it
		if vi.ValidatorSet == nil {
			refKey := []byte(fmt.Sprintf("validatorsKey:%v", vi.LastHeightChanged))
			refVal, err := src.Get(readOpts, refKey)
			if err != nil {
				return fmt.Errorf("read ref %s: %w", refKey, err)
			}
			defer refVal.Free()
			if refVal.Data() == nil {
				return fmt.Errorf("validator set at ref height %d not found", vi.LastHeightChanged)
			}
			var refVi cmtstate.ValidatorsInfo
			if err := refVi.Unmarshal(refVal.Data()); err != nil {
				return fmt.Errorf("unmarshal ref %s: %w", refKey, err)
			}
			vi.ValidatorSet = refVi.ValidatorSet
		}

		// Write with LastHeightChanged = h so no further lookups are needed
		vi.LastHeightChanged = h
		bz, err := vi.Marshal()
		if err != nil {
			return fmt.Errorf("marshal validators for height %d: %w", h, err)
		}
		return dst.Put(writeOpts, key, bz)
	}

	// Same for ConsensusParamsInfo
	resolveAndWriteConsensusParams := func(h int64) error {
		key := []byte(fmt.Sprintf("consensusParamsKey:%v", h))
		val, err := src.Get(readOpts, key)
		if err != nil {
			return fmt.Errorf("read %s: %w", key, err)
		}
		defer val.Free()
		if val.Data() == nil {
			return nil
		}

		var cpi cmtstate.ConsensusParamsInfo
		if err := cpi.Unmarshal(val.Data()); err != nil {
			return fmt.Errorf("unmarshal %s: %w", key, err)
		}

		if cpi.ConsensusParams.Equal(cmtproto.ConsensusParams{}) && cpi.LastHeightChanged != h {
			refKey := []byte(fmt.Sprintf("consensusParamsKey:%v", cpi.LastHeightChanged))
			refVal, err := src.Get(readOpts, refKey)
			if err != nil {
				return fmt.Errorf("read ref %s: %w", refKey, err)
			}
			defer refVal.Free()
			if refVal.Data() != nil {
				var refCpi cmtstate.ConsensusParamsInfo
				if err := refCpi.Unmarshal(refVal.Data()); err != nil {
					return fmt.Errorf("unmarshal ref %s: %w", refKey, err)
				}
				cpi.ConsensusParams = refCpi.ConsensusParams
			}
		}

		cpi.LastHeightChanged = h
		bz, err := cpi.Marshal()
		if err != nil {
			return fmt.Errorf("marshal consensus params for height %d: %w", h, err)
		}
		return dst.Put(writeOpts, key, bz)
	}

	// Read, patch, and write stateKey — update LastHeightValidatorsChanged and
	// LastHeightConsensusParamsChanged to the checkpoint height so that new
	// validator entries written by CometBFT reference our checkpoint (where the
	// full set is stored) instead of the original genesis height.
	{
		stateVal, err := src.Get(readOpts, []byte("stateKey"))
		if err != nil {
			return fmt.Errorf("read stateKey: %w", err)
		}
		defer stateVal.Free()
		if stateVal.Data() == nil {
			return fmt.Errorf("stateKey not found in source state.db")
		}

		var state cmtstate.State
		if err := state.Unmarshal(stateVal.Data()); err != nil {
			return fmt.Errorf("unmarshal stateKey: %w", err)
		}

		// Point back-references to checkpoint height so future entries resolve here
		state.LastHeightValidatorsChanged = height
		state.LastHeightConsensusParamsChanged = height

		bz, err := state.Marshal()
		if err != nil {
			return fmt.Errorf("marshal patched stateKey: %w", err)
		}
		if err := dst.Put(writeOpts, []byte("stateKey"), bz); err != nil {
			return fmt.Errorf("write patched stateKey: %w", err)
		}
	}

	// Resolve and write self-contained validator sets for height-1, height, height+1
	for _, h := range []int64{height - 1, height, height + 1} {
		if err := resolveAndWriteValidators(h); err != nil {
			return fmt.Errorf("resolve validators at %d: %w", h, err)
		}
	}

	// Resolve and write consensus params
	if err := resolveAndWriteConsensusParams(height); err != nil {
		return fmt.Errorf("resolve consensus params: %w", err)
	}

	// Copy last ABCI response
	if err := copyKey([]byte("lastABCIResponseKey")); err != nil {
		return fmt.Errorf("copy lastABCIResponseKey: %w", err)
	}

	// Copy genesis doc — in-place-testnet reads it from state.db to avoid
	// parsing the genesis JSON file (which has integer encoding CometBFT rejects).
	// Note: in-place-testnet updates the chain ID in this doc after loading it.
	if err := copyKey([]byte("genesisDoc")); err != nil {
		return fmt.Errorf("copy genesisDoc: %w", err)
	}

	return nil
}

// buildMinimalBlockStoreDB constructs a minimal CometBFT blockstore.db for
// bootstrapping a forked chain from the given checkpoint height.
//
// CometBFT's blockstore.db normally stores every block ever produced:
//
//	blockStore       → BlockStoreState proto {Base, Height} — tracks the range of stored blocks
//	H:{height}       → BlockMeta proto (header, block ID, size, num txs)
//	P:{height}:{idx} → Part proto (64KB chunk of the serialized block)
//	C:{height}       → Commit proto (precommit signatures for this height's block)
//	SC:{height}      → SeenCommit proto (+2/3 precommits observed by this node)
//	BH:{hash}        → height (reverse index from block hash to height)
//
// A fork only needs the last committed block to continue consensus. This
// function creates a new rocksdb containing:
//
//  1. H:{H} — block meta at the checkpoint height. Parsed to determine the
//     number of block parts (via BlockID.PartSetHeader.Total).
//
//  2. P:{H}:{0..N} — all block parts for height H. CometBFT reconstructs the
//     full block by reassembling these 64KB chunks. All parts must be present.
//
//  3. C:{H-1} and C:{H} — commit (precommit signatures) for the previous and
//     current block. C:{H-1} is stored as block H's LastCommit and is needed
//     to validate the commit signatures. C:{H} is needed for the current height.
//
//  4. SC:{H} — the seen commit at the checkpoint height, tracking which +2/3
//     precommits this node observed.
//
//  5. blockStore — a BlockStoreState with Base=H and Height=H, telling CometBFT
//     that this store contains exactly one block at height H.
func buildMinimalBlockStoreDB(homePath, cpDir string, height int64) error {
	srcPath := filepath.Join(homePath, "data", "blockstore.db")
	dstPath := filepath.Join(cpDir, "blockstore.db")

	if err := os.MkdirAll(dstPath, 0o755); err != nil {
		return fmt.Errorf("mkdir blockstore.db: %w", err)
	}

	// Open source read-only
	roOpts := grocksdb.NewDefaultOptions()
	src, err := grocksdb.OpenDbForReadOnly(roOpts, srcPath, false)
	if err != nil {
		return fmt.Errorf("open source blockstore.db: %w", err)
	}
	defer src.Close()

	// Open destination for writing
	wOpts := grocksdb.NewDefaultOptions()
	wOpts.SetCreateIfMissing(true)
	dst, err := grocksdb.OpenDb(wOpts, dstPath)
	if err != nil {
		return fmt.Errorf("open dest blockstore.db: %w", err)
	}
	defer dst.Close()

	readOpts := grocksdb.NewDefaultReadOptions()
	writeOpts := grocksdb.NewDefaultWriteOptions()

	// First, read block meta to find how many parts
	metaKey := []byte(fmt.Sprintf("H:%v", height))
	metaVal, err := src.Get(readOpts, metaKey)
	if err != nil {
		return fmt.Errorf("read block meta: %w", err)
	}
	defer metaVal.Free()

	if metaVal.Data() == nil {
		return fmt.Errorf("block meta not found at height %d", height)
	}

	// Parse block meta to get total parts count
	var blockMeta cmtproto.BlockMeta
	if err := proto.Unmarshal(metaVal.Data(), &blockMeta); err != nil {
		return fmt.Errorf("unmarshal block meta: %w", err)
	}
	totalParts := int(blockMeta.BlockID.PartSetHeader.Total)

	// Copy block meta
	if err := dst.Put(writeOpts, metaKey, metaVal.Data()); err != nil {
		return fmt.Errorf("write block meta: %w", err)
	}

	// Copy all block parts
	for i := 0; i < totalParts; i++ {
		partKey := []byte(fmt.Sprintf("P:%v:%v", height, i))
		partVal, err := src.Get(readOpts, partKey)
		if err != nil {
			return fmt.Errorf("read block part %d: %w", i, err)
		}
		if partVal.Data() != nil {
			if err := dst.Put(writeOpts, partKey, partVal.Data()); err != nil {
				partVal.Free()
				return fmt.Errorf("write block part %d: %w", i, err)
			}
		}
		partVal.Free()
	}

	// Copy commit for height-1 (block's LastCommit) and height
	for _, h := range []int64{height - 1, height} {
		commitKey := []byte(fmt.Sprintf("C:%v", h))
		commitVal, err := src.Get(readOpts, commitKey)
		if err != nil {
			return fmt.Errorf("read commit %d: %w", h, err)
		}
		if commitVal.Data() != nil {
			if err := dst.Put(writeOpts, commitKey, commitVal.Data()); err != nil {
				commitVal.Free()
				return fmt.Errorf("write commit %d: %w", h, err)
			}
		}
		commitVal.Free()
	}

	// Copy seen commit
	scKey := []byte(fmt.Sprintf("SC:%v", height))
	scVal, err := src.Get(readOpts, scKey)
	if err != nil {
		return fmt.Errorf("read seen commit: %w", err)
	}
	if scVal.Data() != nil {
		if err := dst.Put(writeOpts, scKey, scVal.Data()); err != nil {
			scVal.Free()
			return fmt.Errorf("write seen commit: %w", err)
		}
	}
	scVal.Free()

	// Write BlockStoreState with base=height, height=height
	bss := &cmtstore.BlockStoreState{
		Base:   height,
		Height: height,
	}
	bssBytes, err := proto.Marshal(bss)
	if err != nil {
		return fmt.Errorf("marshal blockstore state: %w", err)
	}
	if err := dst.Put(writeOpts, []byte("blockStore"), bssBytes); err != nil {
		return fmt.Errorf("write blockstore state: %w", err)
	}

	return nil
}

// overrideWasmVariables overrides the wasm variables to:
//   - allow for larger wasm files
func overrideWasmVariables() {
	// Override Wasm size limitation from WASMD.
	wasmtypes.MaxWasmSize = 2 * 1024 * 1024
	wasmtypes.MaxProposalWasmSize = wasmtypes.MaxWasmSize
}
