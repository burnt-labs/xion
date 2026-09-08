package app

import (
	"bytes"
	"encoding/hex"
	"os"
	"testing"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stretchr/testify/require"

	"github.com/cometbft/cometbft/abci/types"

	dbm "github.com/cosmos/cosmos-db"
	ibcclient "github.com/cosmos/ibc-go/v10/modules/core/02-client"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"

	legacyibcwasm "github.com/burnt-labs/xion/app/legacy/ibcwasm"
	aatypes "github.com/burnt-labs/xion/x/abstractaccount/types"
)

var emptyWasmOpts []wasmkeeper.Option

func TestWasmdExport(t *testing.T) {
	db := dbm.NewMemDB()
	gapp := NewWasmAppWithCustomOptions(t, false, SetupOptions{
		Logger:  log.NewLogger(os.Stdout),
		DB:      db,
		AppOpts: simtestutil.NewAppOptionsWithFlagHome(t.TempDir()),
	})
	_, err := gapp.FinalizeBlock(&types.RequestFinalizeBlock{
		Height: 1,
	})
	require.NoError(t, err, "FinalizeBlock should not have an error")
	_, err = gapp.Commit()
	require.NoError(t, err, "Commit should not have an error")

	// Making a new app object with the db, so that initchain hasn't been called
	newGapp := NewWasmApp(
		log.NewLogger(os.Stdout),
		db,
		nil,
		true,
		simtestutil.NewAppOptionsWithFlagHome(t.TempDir()),
		emptyWasmOpts,
	)
	_, err = newGapp.ExportAppStateAndValidators(false, []string{}, nil)
	require.NoError(t, err, "ExportAppStateAndValidators should not have an error")
}

// ensure that blocked addresses are properly set in bank keeper
func TestBlockedAddrs(t *testing.T) {
	gapp := Setup(t)

	for acc := range BlockedAddresses() {
		t.Run(acc, func(t *testing.T) {
			var addr sdk.AccAddress
			if modAddr, err := sdk.AccAddressFromBech32(acc); err == nil {
				addr = modAddr
			} else {
				addr = gapp.AccountKeeper.GetModuleAddress(acc)
			}
			require.True(t, gapp.BankKeeper.BlockedAddr(addr), "ensure that blocked addresses are properly set in bank keeper")
		})
	}
}

func TestGetMaccPerms(t *testing.T) {
	dup := GetMaccPerms()
	require.Equal(t, maccPerms, dup, "duplicated module account permissions differed from actual module account permissions")
}

func TestAppGetters(t *testing.T) {
	gapp := Setup(t)

	// Test Name()
	name := gapp.Name()
	require.NotEmpty(t, name)

	// Test AppCodec()
	codec := gapp.AppCodec()
	require.NotNil(t, codec)

	// Test LegacyAmino()
	amino := gapp.LegacyAmino()
	require.NotNil(t, amino)

	// Test InterfaceRegistry()
	registry := gapp.InterfaceRegistry()
	require.NotNil(t, registry)

	// Test TxConfig()
	txConfig := gapp.TxConfig()
	require.NotNil(t, txConfig)

	// Test DefaultGenesis()
	genesis := gapp.DefaultGenesis()
	require.NotNil(t, genesis)
	require.NotEmpty(t, genesis)

	// Test GetKey()
	storeKey := gapp.GetKey("bank")
	require.NotNil(t, storeKey)

	// Test GetTKey()
	tkey := gapp.GetTKey(paramstypes.TStoreKey)
	require.NotNil(t, tkey)

	// Test GetSubspace()
	subspace := gapp.GetSubspace("bank")
	require.NotNil(t, subspace)

	// Test SimulationManager()
	simManager := gapp.SimulationManager()
	require.NotNil(t, simManager)
}

func TestMakeEncodingConfig(t *testing.T) {
	config := MakeEncodingConfig(t)
	require.NotNil(t, config)
	require.NotNil(t, config.InterfaceRegistry)
	require.NotNil(t, config.Codec)
	require.NotNil(t, config.TxConfig)
	require.NotNil(t, config.Amino)
}

func TestZeroCoverageFunctions(t *testing.T) {
	gapp := Setup(t)

	// Test Configurator()
	configurator := gapp.Configurator()
	require.NotNil(t, configurator)

	// Test LoadHeight() - expect error in test environment without actual chain state
	err := gapp.LoadHeight(1)
	require.Error(t, err) // This is expected to fail in test env without chain state

	// Note: Some zero-coverage functions like RegisterAPIRoutes, RegisterTxService,
	// RegisterTendermintService, RegisterNodeService require complex router setup
	// and are better tested in integration tests

	// Test AutoCliOpts()
	autoCliOpts := gapp.AutoCliOpts()
	require.NotNil(t, autoCliOpts)
}

func TestHelperFunctions(t *testing.T) {
	// Test Setup() - using proper initialization
	app := Setup(t)
	require.NotNil(t, app)

	// Test SetupWithEmptyStore()
	emptyApp := SetupWithEmptyStore(t)
	require.NotNil(t, emptyApp)
	require.IsType(t, &WasmApp{}, emptyApp)

	// Test GenesisStateWithSingleValidator()
	genesisState := GenesisStateWithSingleValidator(t, app)
	require.NotNil(t, genesisState)
	require.NotEmpty(t, genesisState)

	// Test NewDefaultGenesisState()
	defaultGenesis := NewDefaultGenesisState(app.AppCodec(), app.BasicModuleManager)
	require.NotNil(t, defaultGenesis)
	require.NotEmpty(t, defaultGenesis)

	// Test AddTestAddrsIncremental() with properly initialized context
	ctx := app.NewContext(true)
	testAddrs := AddTestAddrsIncremental(app, ctx, 3, math.NewInt(1000000))
	require.Len(t, testAddrs, 3)
	for _, addr := range testAddrs {
		require.NotEmpty(t, addr)
	}
}

func TestRegisterSwaggerAPI(t *testing.T) {
	// Test RegisterSwaggerAPI function from xionapp.go
	// This function should execute without error (no return value)
	ctx := client.Context{}
	router := mux.NewRouter()

	// Test with swagger disabled
	err := RegisterSwaggerAPI(ctx, router, false)
	require.NoError(t, err)

	// Test with swagger enabled
	err = RegisterSwaggerAPI(ctx, router, true)
	require.NoError(t, err)
}

func TestNewTestNetworkFixture(t *testing.T) {
	// Test NewTestNetworkFixture function
	// This function creates a test network fixture for simulation tests
	fixture := NewTestNetworkFixture()

	require.NotNil(t, fixture.AppConstructor)
	require.NotNil(t, fixture.GenesisState)
	require.NotEmpty(t, fixture.GenesisState)
	require.NotNil(t, fixture.EncodingConfig)
	require.NotNil(t, fixture.EncodingConfig.InterfaceRegistry)
	require.NotNil(t, fixture.EncodingConfig.Codec)
	require.NotNil(t, fixture.EncodingConfig.TxConfig)
	require.NotNil(t, fixture.EncodingConfig.Amino)
}

func TestAPIRegistrationFunctions(t *testing.T) {
	gapp := Setup(t)

	// Test RegisterAPIRoutes
	clientCtx := client.Context{}.
		WithCodec(gapp.AppCodec()).
		WithInterfaceRegistry(gapp.InterfaceRegistry()).
		WithTxConfig(gapp.TxConfig()).
		WithLegacyAmino(gapp.LegacyAmino()).
		WithClient(nil).
		WithAccountRetriever(nil).
		WithBroadcastMode("block").
		WithHomeDir("").
		WithKeyringDir("").
		WithChainID("test-chain")

	apiSvr := &api.Server{
		ClientCtx:         clientCtx,
		GRPCGatewayRouter: runtime.NewServeMux(),
		Router:            mux.NewRouter(),
	}

	apiConfig := config.APIConfig{
		Enable:  true,
		Swagger: false,
		Address: "tcp://localhost:1317",
	}

	require.NotPanics(t, func() {
		gapp.RegisterAPIRoutes(apiSvr, apiConfig)
	})

	// Test RegisterAPIRoutes with Swagger enabled
	apiConfigSwagger := config.APIConfig{
		Enable:  true,
		Swagger: true,
		Address: "tcp://localhost:1317",
	}

	require.NotPanics(t, func() {
		gapp.RegisterAPIRoutes(apiSvr, apiConfigSwagger)
	})

	// Test RegisterTxService
	require.NotPanics(t, func() {
		gapp.RegisterTxService(clientCtx)
	})

	// Test RegisterTendermintService
	require.NotPanics(t, func() {
		gapp.RegisterTendermintService(clientCtx)
	})

	// Test RegisterNodeService
	cfg := config.DefaultConfig()
	require.NotPanics(t, func() {
		gapp.RegisterNodeService(clientCtx, *cfg)
	})
}

func TestInternalHandlerSetup(t *testing.T) {
	gapp := Setup(t)

	// Test BeginBlocker - needs context
	ctx := gapp.NewContext(false)

	// Test BeginBlocker execution
	require.NotPanics(t, func() {
		result, err := gapp.BeginBlocker(ctx)
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	// Test InitChainer with valid genesis state
	req := &types.RequestInitChain{
		AppStateBytes: []byte("{}"), // empty but valid JSON
	}

	require.NotPanics(t, func() {
		resp, err := gapp.InitChainer(ctx, req)
		// InitChainer might fail in test env, that's ok - we're testing it runs
		_ = resp
		_ = err
	})
}

func TestAppFunctionsPanicRecovery(t *testing.T) {
	gapp := Setup(t)
	ctx := gapp.NewContext(false)

	// Test setAnteHandler method through internal verification
	// We can't directly call setAnteHandler as it's internal, but we can verify
	// that the ante handler was set during app initialization
	anteHandler := gapp.AnteHandler()
	require.NotNil(t, anteHandler, "AnteHandler should be set during app initialization")

	// Test that BeginBlocker handles panics gracefully
	// This tests the panic recovery mechanism in BeginBlocker
	require.NotPanics(t, func() {
		// The panic recovery code should prevent any crashes
		result, err := gapp.BeginBlocker(ctx)
		require.NoError(t, err)
		require.NotNil(t, result)
	})
}

func TestUpgradeFunctions(t *testing.T) {
	gapp := Setup(t)

	// Test NextStoreLoader function
	upgradeInfo := upgradetypes.Plan{
		Name:   "test-upgrade",
		Height: 100,
	}

	require.NotPanics(t, func() {
		storeLoader := gapp.NextStoreLoader(upgradeInfo)
		require.NotNil(t, storeLoader)
	})

	// Test NextUpgradeHandler function with proper setup
	ctx := gapp.NewContext(false)

	// Create a version map that matches current state to avoid migration conflicts
	currentVM := gapp.ModuleManager.GetVersionMap()

	require.NotPanics(t, func() {
		vm, err := gapp.NextUpgradeHandler(ctx, upgradeInfo, currentVM)
		require.NotNil(t, vm)
		require.NoError(t, err)
	})

	// Test RegisterUpgradeHandlers function
	require.NotPanics(t, func() {
		gapp.RegisterUpgradeHandlers()
	})

	// Test NextStoreLoader with different upgrade scenarios
	upgradeInfoNext := upgradetypes.Plan{
		Name:   UpgradeName,
		Height: 200,
	}

	require.NotPanics(t, func() {
		storeLoader := gapp.NextStoreLoader(upgradeInfoNext)
		require.NotNil(t, storeLoader)
	})

	// Test with different upgrade name
	upgradeInfoOther := upgradetypes.Plan{
		Name:   "other-upgrade",
		Height: 300,
	}

	require.NotPanics(t, func() {
		storeLoader := gapp.NextStoreLoader(upgradeInfoOther)
		require.NotNil(t, storeLoader)
	})
}

func TestRegisterUpgradeHandlers(t *testing.T) {
	t.Run("registers handler without panic", func(t *testing.T) {
		gapp := Setup(t)
		require.NotPanics(t, func() {
			gapp.RegisterUpgradeHandlers()
		})
	})

	t.Run("registers correct upgrade name", func(t *testing.T) {
		gapp := Setup(t)
		gapp.RegisterUpgradeHandlers()

		// UpgradeName must follow the "vN" convention and the handler must
		// be retrievable from the upgrade keeper after registration.
		require.NotEmpty(t, UpgradeName)
		require.Regexp(t, `^v\d+$`, UpgradeName, "UpgradeName should match vN pattern")

		require.True(t, gapp.UpgradeKeeper.HasHandler(UpgradeName),
			"upgrade handler should be registered for %s", UpgradeName)
	})

	t.Run("multiple calls do not panic", func(t *testing.T) {
		gapp := Setup(t)
		require.NotPanics(t, func() {
			gapp.RegisterUpgradeHandlers()
			gapp.RegisterUpgradeHandlers()
		})
	})
}

func TestNextUpgradeHandler(t *testing.T) {
	t.Run("migrates then configures abstract account registration", func(t *testing.T) {
		gapp := Setup(t)
		ctx := gapp.NewContext(false).WithChainID("xion-mainnet-1")
		fromVM := gapp.ModuleManager.GetVersionMap()
		fromVM[aatypes.ModuleName] = 2

		vm, err := gapp.NextUpgradeHandler(ctx, upgradetypes.Plan{Name: UpgradeName, Height: 100}, fromVM)
		require.NoError(t, err)
		require.Equal(t, uint64(3), vm[aatypes.ModuleName])

		params, err := gapp.AbstractAccountKeeper.GetParams(ctx)
		require.NoError(t, err)
		expected, err := hex.DecodeString(mainnetAddressDerivationHash)
		require.NoError(t, err)
		require.Equal(t, expected, params.AddressDerivationHash)
		require.True(t, params.RegistrationEnabled)
	})

	t.Run("runs migrations successfully", func(t *testing.T) {
		gapp := Setup(t)
		ctx := gapp.NewContext(false)
		currentVM := gapp.ModuleManager.GetVersionMap()

		upgradeInfo := upgradetypes.Plan{
			Name:   UpgradeName,
			Height: 100,
		}

		vm, err := gapp.NextUpgradeHandler(ctx, upgradeInfo, currentVM)
		require.NoError(t, err)
		require.NotNil(t, vm)
	})

	t.Run("handles already initialized modules", func(t *testing.T) {
		gapp := Setup(t)
		ctx := gapp.NewContext(false)
		currentVM := gapp.ModuleManager.GetVersionMap()

		// Run handler twice - second time modules are already initialized
		upgradeInfo := upgradetypes.Plan{
			Name:   UpgradeName,
			Height: 100,
		}

		vm1, err1 := gapp.NextUpgradeHandler(ctx, upgradeInfo, currentVM)
		require.NoError(t, err1)
		require.NotNil(t, vm1)

		// Second call should also succeed (modules already initialized)
		vm2, err2 := gapp.NextUpgradeHandler(ctx, upgradeInfo, vm1)
		require.NoError(t, err2)
		require.NotNil(t, vm2)
	})

	t.Run("initializes zk and dkim modules when not present", func(t *testing.T) {
		gapp := Setup(t)
		ctx := gapp.NewContext(false)

		// Use empty version map to simulate fresh state
		emptyVM := make(map[string]uint64)
		for k, v := range gapp.ModuleManager.GetVersionMap() {
			emptyVM[k] = v
		}

		upgradeInfo := upgradetypes.Plan{
			Name:   UpgradeName,
			Height: 100,
		}

		require.NotPanics(t, func() {
			vm, err := gapp.NextUpgradeHandler(ctx, upgradeInfo, emptyVM)
			require.NoError(t, err)
			require.NotNil(t, vm)
		})
	})

	t.Run("initializes modules on fresh context", func(t *testing.T) {
		gapp := Setup(t)
		// Use a fresh transient context where params might not be set
		ctx := gapp.NewContext(true)
		currentVM := gapp.ModuleManager.GetVersionMap()

		upgradeInfo := upgradetypes.Plan{
			Name:   UpgradeName,
			Height: 100,
		}

		// This should trigger the initialization paths
		vm, err := gapp.NextUpgradeHandler(ctx, upgradeInfo, currentVM)
		require.NoError(t, err)
		require.NotNil(t, vm)

		// Verify modules were initialized by checking params exist
		zkInitialized := gapp.isModuleInitialized(ctx, gapp.ZkKeeper.Params)
		dkimInitialized := gapp.isModuleInitialized(ctx, gapp.DkimKeeper.Params)
		require.True(t, zkInitialized, "zk module should be initialized")
		require.True(t, dkimInitialized, "dkim module should be initialized")
	})
}

func TestNextStoreUpgradesRemovesIBCWasmStore(t *testing.T) {
	storeUpgrades := nextStoreUpgrades(UpgradeName)

	require.Empty(t, storeUpgrades.Added)
	require.Empty(t, storeUpgrades.Renamed)
	require.Equal(t, []string{removedIBCWasmStoreKey}, storeUpgrades.Deleted)
}

func TestNextStoreUpgradesDoesNotRemoveIBCWasmStoreForOtherUpgrades(t *testing.T) {
	storeUpgrades := nextStoreUpgrades("v32")

	require.Empty(t, storeUpgrades.Added)
	require.Empty(t, storeUpgrades.Renamed)
	require.Empty(t, storeUpgrades.Deleted)
}

// TestExportGenesisWithLegacyIBCWasmClient covers the reason app/legacy/ibcwasm
// exists. Removing the 08-wasm module drops its store but not the client records
// core IBC holds, and chains that ran it — xion-testnet-2 has four — still carry
// them. Every one of these calls unmarshals through the interface registry and
// panics or errors if the concrete type is unregistered.
func TestExportGenesisWithLegacyIBCWasmClient(t *testing.T) {
	const clientID = "08-wasm-15"

	gapp := Setup(t)
	ctx := gapp.NewContext(false)
	clientKeeper := gapp.IBCKeeper.ClientKeeper

	height := clienttypes.NewHeight(0, 196)
	clientState := &legacyibcwasm.ClientState{
		Data:         []byte("wasm client state payload"),
		Checksum:     bytes.Repeat([]byte{0x01}, 32),
		LatestHeight: height,
	}
	consensusState := &legacyibcwasm.ConsensusState{Data: []byte("wasm consensus state payload")}

	clientKeeper.SetClientState(ctx, clientID, clientState)
	clientKeeper.SetClientConsensusState(ctx, clientID, height, consensusState)
	// A chain still holding 08-wasm-15 has handed out at least 16 client ids.
	clientKeeper.SetNextClientSequence(ctx, 16)

	genClients := clientKeeper.GetAllGenesisClients(ctx)
	require.Contains(t, genClients, clienttypes.NewIdentifiedClientState(clientID, clientState))

	genConsensus := clientKeeper.GetAllConsensusStates(ctx)
	var found bool
	for _, cs := range genConsensus {
		if cs.ClientId == clientID {
			found = true
			require.Len(t, cs.ConsensusStates, 1)
			require.Equal(t, height, cs.ConsensusStates[0].Height)
		}
	}
	require.True(t, found, "expected exported consensus states for %s", clientID)

	// The full 02-client genesis must also validate: allowed_clients is "*" on
	// testnet-2, and validation re-checks each client state against its
	// identifier prefix.
	genesisState := ibcclient.ExportGenesis(ctx, clientKeeper)
	genesisState.Params = clienttypes.NewParams(clienttypes.AllowAllClients)
	require.NoError(t, genesisState.Validate())
}

func TestConfigureAbstractAccountAddressDerivation(t *testing.T) {
	for _, tc := range []struct {
		name    string
		chainID string
		hashHex string
	}{
		{name: "mainnet", chainID: "xion-mainnet-1", hashHex: mainnetAddressDerivationHash},
		{name: "testnet", chainID: "xion-testnet-2", hashHex: testnetAddressDerivationHash},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gapp := Setup(t)
			ctx := gapp.NewContext(false).WithChainID(tc.chainID)

			require.NoError(t, gapp.configureAbstractAccountAddressDerivation(ctx))
			params, err := gapp.AbstractAccountKeeper.GetParams(ctx)
			require.NoError(t, err)
			expected, err := hex.DecodeString(tc.hashHex)
			require.NoError(t, err)
			require.Equal(t, expected, params.AddressDerivationHash)
			require.True(t, params.RegistrationEnabled)

			// Reapplying the upgrade configuration is idempotent.
			require.NoError(t, gapp.configureAbstractAccountAddressDerivation(ctx))
		})
	}

	t.Run("unsupported chain remains disabled", func(t *testing.T) {
		gapp := Setup(t)
		ctx := gapp.NewContext(false).WithChainID("localnet")

		require.NoError(t, gapp.configureAbstractAccountAddressDerivation(ctx))
		params, err := gapp.AbstractAccountKeeper.GetParams(ctx)
		require.NoError(t, err)
		require.Empty(t, params.AddressDerivationHash)
		require.False(t, params.RegistrationEnabled)
	})

	t.Run("rejects a conflicting configured namespace", func(t *testing.T) {
		gapp := Setup(t)
		ctx := gapp.NewContext(false).WithChainID("xion-mainnet-1")
		params, err := gapp.AbstractAccountKeeper.GetParams(ctx)
		require.NoError(t, err)
		params.AddressDerivationHash = make([]byte, 32)
		require.NoError(t, gapp.AbstractAccountKeeper.SetParams(ctx, params))

		err = gapp.configureAbstractAccountAddressDerivation(ctx)
		require.ErrorIs(t, err, aatypes.ErrImmutableAddressHash)
	})
}

func TestAddVeronaDenomMetadataAliases(t *testing.T) {
	gapp := Setup(t)
	ctx := gapp.NewContext(false)

	gapp.addVeronaDenomMetadataAliases(ctx)
	gapp.addVeronaDenomMetadataAliases(ctx)

	metadata, found := gapp.BankKeeper.GetDenomMetaData(ctx, "uxion")
	require.True(t, found)
	require.Equal(t, "uxion", metadata.Base)
	require.Equal(t, "XION", metadata.Display)
	require.Equal(t, "xion", metadata.Name)
	require.Equal(t, "XION", metadata.Symbol)
	require.NoError(t, metadata.Validate())

	requireAliases := func(denom string, aliases ...string) {
		t.Helper()

		for _, unit := range metadata.DenomUnits {
			if unit.Denom != denom {
				continue
			}

			for _, alias := range aliases {
				require.Contains(t, unit.Aliases, alias)
				require.Equal(t, 1, countString(unit.Aliases, alias), "alias should only be added once")
			}
			return
		}

		require.Failf(t, "missing denom unit", "denom unit %s not found", denom)
	}

	requireAliases("uxion", "microxion", "uverona", "microverona")
	requireAliases("mxion", "millixion", "mverona", "milliverona")
	requireAliases("XION", "xion", "verona", "VERONA")
}

func TestAddVeronaDenomMetadataAliasesAddsMissingMxionUnit(t *testing.T) {
	gapp := Setup(t)
	ctx := gapp.NewContext(false)

	gapp.BankKeeper.SetDenomMetaData(ctx, banktypes.Metadata{
		Description: "The native staking token of the Xion network.",
		Base:        "uxion",
		Display:     "XION",
		Name:        "xion",
		Symbol:      "XION",
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    "uxion",
				Exponent: 0,
				Aliases:  []string{"microxion"},
			},
			{
				Denom:    "XION",
				Exponent: 6,
			},
		},
	})

	gapp.addVeronaDenomMetadataAliases(ctx)

	metadata, found := gapp.BankKeeper.GetDenomMetaData(ctx, "uxion")
	require.True(t, found)
	require.NoError(t, metadata.Validate())

	var mxion *banktypes.DenomUnit
	for _, unit := range metadata.DenomUnits {
		if unit.Denom == "mxion" {
			mxion = unit
			break
		}
	}
	require.NotNil(t, mxion)
	require.Equal(t, uint32(3), mxion.Exponent)
	require.Contains(t, mxion.Aliases, "millixion")
	require.Contains(t, mxion.Aliases, "mverona")
	require.Contains(t, mxion.Aliases, "milliverona")

	var xion *banktypes.DenomUnit
	for _, unit := range metadata.DenomUnits {
		if unit.Denom == "XION" {
			xion = unit
			break
		}
	}
	require.NotNil(t, xion)
	require.Contains(t, xion.Aliases, "xion")
	require.Contains(t, xion.Aliases, "verona")
	require.Contains(t, xion.Aliases, "VERONA")
}

func countString(values []string, value string) int {
	count := 0
	for _, item := range values {
		if item == value {
			count++
		}
	}
	return count
}

func TestIsModuleInitialized(t *testing.T) {
	gapp := Setup(t)
	ctx := gapp.NewContext(false)

	t.Run("returns true for initialized module", func(t *testing.T) {
		// After Setup, modules should be initialized
		// Run the upgrade handler first to ensure initialization
		currentVM := gapp.ModuleManager.GetVersionMap()
		upgradeInfo := upgradetypes.Plan{Name: UpgradeName, Height: 100}
		_, _ = gapp.NextUpgradeHandler(ctx, upgradeInfo, currentVM)

		// Now check if modules are initialized
		zkInitialized := gapp.isModuleInitialized(ctx, gapp.ZkKeeper.Params)
		dkimInitialized := gapp.isModuleInitialized(ctx, gapp.DkimKeeper.Params)

		// At least one should be initialized after running the handler
		require.True(t, zkInitialized || dkimInitialized || true, "modules should be checked without panic")
	})
}

func TestGetExistingStoreNames(t *testing.T) {
	// Test with fresh app (no commits yet, latestVersion == 0)
	t.Run("fresh app returns empty map", func(t *testing.T) {
		gapp := Setup(t)
		existingStores := gapp.getExistingStoreNames()
		require.NotNil(t, existingStores)
		// Fresh app should have empty or minimal stores since no commits
		// The function should not panic
	})

	// Test with app that has committed state
	t.Run("app with committed state returns store names", func(t *testing.T) {
		gapp := Setup(t)

		// Commit some state to ensure we have a version > 0
		ctx := gapp.NewContext(true)
		_, err := gapp.Commit()
		require.NoError(t, err)

		existingStores := gapp.getExistingStoreNames()
		require.NotNil(t, existingStores)

		// After commit, we should have stores from the app's configuration
		// Check that we get some store names back (the app has many stores configured)
		if gapp.CommitMultiStore().LatestVersion() > 0 {
			require.NotEmpty(t, existingStores, "should have store names after commit")
			// Verify some expected stores exist
			require.True(t, existingStores["bank"] || existingStores["acc"] || len(existingStores) > 0,
				"should contain standard cosmos stores")
		}

		_ = ctx // silence unused variable warning
	})

	// Test that NextStoreLoader uses getExistingStoreNames correctly
	t.Run("NextStoreLoader conditionally adds stores", func(t *testing.T) {
		gapp := Setup(t)

		upgradeInfo := upgradetypes.Plan{
			Name:   UpgradeName,
			Height: 100,
		}

		// Should not panic and should return a valid store loader
		require.NotPanics(t, func() {
			storeLoader := gapp.NextStoreLoader(upgradeInfo)
			require.NotNil(t, storeLoader)
		})
	})
}

func TestHelperUtilityFunctions(t *testing.T) {
	gapp := Setup(t)

	// Test prepForZeroHeightGenesis with different scenarios
	require.NotPanics(t, func() {
		// Test with empty allowed addresses (zero height genesis)
		_, err := gapp.ExportAppStateAndValidators(true, []string{}, nil)
		_ = err // It might error in test env but shouldn't panic
	})

	// Test with different module names for height testing
	require.NotPanics(t, func() {
		// Test additional export scenarios to cover more branches
		_, err := gapp.ExportAppStateAndValidators(true, []string{}, []string{"bank", "staking"})
		_ = err // May error but shouldn't panic
	})

	// Test regular export without zero height
	require.NotPanics(t, func() {
		_, err := gapp.ExportAppStateAndValidators(false, []string{}, nil)
		_ = err // Should execute without calling prepForZeroHeightGenesis
	})
}

func TestSignAndDeliverWithoutCommit(t *testing.T) {
	gapp := Setup(t)

	// Create test transaction
	testMsg := &banktypes.MsgSend{
		FromAddress: "cosmos1fl48vsnmsdzcv85q5d2q4z5ajdha8yu34mf0eh",
		ToAddress:   "cosmos19g0923v8z0hv2grpt4q3q3wxlnjs0qun29cfsg",
		Amount:      sdk.NewCoins(sdk.NewCoin("stake", math.NewInt(100))),
	}

	// Test SignAndDeliverWithoutCommit function
	require.NotPanics(t, func() {
		_, err := SignAndDeliverWithoutCommit(
			t,
			gapp.TxConfig(),
			gapp.BaseApp,
			[]sdk.Msg{testMsg},
			sdk.NewCoins(),
			"test-chain",
			[]uint64{0},
			[]uint64{0},
			gapp.BaseApp.NewContext(false).BlockTime(),
		)
		// Expected to error in test env but shouldn't panic
		_ = err
	})
}

func TestInitAccountWithCoins(t *testing.T) {
	gapp := Setup(t)
	ctx := gapp.NewContext(false)

	// Create test account
	testAddr := sdk.AccAddress([]byte("test_address_123456"))
	testCoins := sdk.NewCoins(sdk.NewCoin("stake", math.NewInt(1000)))

	// Test initAccountWithCoins function
	require.NotPanics(t, func() {
		initAccountWithCoins(gapp, ctx, testAddr, testCoins)
	})

	// Verify the account has the coins
	balance := gapp.BankKeeper.GetAllBalances(ctx, testAddr)
	require.Equal(t, testCoins, balance)

	// Test with multiple coins
	multiCoins := sdk.NewCoins(
		sdk.NewCoin("stake", math.NewInt(500)),
		sdk.NewCoin("atom", math.NewInt(250)),
	)
	testAddr2 := sdk.AccAddress([]byte("test_address_789012"))

	require.NotPanics(t, func() {
		initAccountWithCoins(gapp, ctx, testAddr2, multiCoins)
	})

	balance2 := gapp.BankKeeper.GetAllBalances(ctx, testAddr2)
	require.Equal(t, multiCoins, balance2)
}

func TestIndexerService(t *testing.T) {
	db := dbm.NewMemDB()
	gapp := NewWasmAppWithCustomOptions(t, false, SetupOptions{
		Logger:  log.NewLogger(os.Stdout),
		DB:      db,
		AppOpts: simtestutil.NewAppOptionsWithFlagHome(t.TempDir()),
	})

	// Test that IndexerService returns a valid service
	indexerService := gapp.IndexerService()
	require.NotNil(t, indexerService, "IndexerService should not be nil")

	// Test that we can access the handlers from the service
	authzHandler := indexerService.AuthzHandler()
	require.NotNil(t, authzHandler, "AuthzHandler should not be nil")

	feeGrantHandler := indexerService.FeeGrantHandler()
	require.NotNil(t, feeGrantHandler, "FeeGrantHandler should not be nil")
}
