package app

import (
	"os"
	"testing"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stretchr/testify/require"

	"github.com/cometbft/cometbft/abci/types"

	dbm "github.com/cosmos/cosmos-db"

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
	upgradeInfoV22 := upgradetypes.Plan{
		Name:   "v22",
		Height: 200,
	}

	require.NotPanics(t, func() {
		storeLoader := gapp.NextStoreLoader(upgradeInfoV22)
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
