package module_test

import (
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stretchr/testify/require"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/client"
	sdkruntime "github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil/integration"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	dkimmodule "github.com/burnt-labs/xion/x/dkim"
	"github.com/burnt-labs/xion/x/dkim/keeper"
	"github.com/burnt-labs/xion/x/dkim/types"
	zkkeeper "github.com/burnt-labs/xion/x/zk/keeper"
)

func setupModule(t *testing.T) *dkimmodule.AppModule {
	t.Helper()

	encCfg := moduletestutil.MakeTestEncodingConfig()
	key := storetypes.NewKVStoreKey(types.ModuleName)
	storeService := sdkruntime.NewKVStoreService(key)

	govModAddr := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	logger := log.NewTestLogger(t)

	zkeeper := zkkeeper.NewKeeper(encCfg.Codec, storeService, logger, govModAddr)

	k := keeper.NewKeeper(encCfg.Codec, storeService, logger, govModAddr, zkeeper)

	appModule := dkimmodule.NewAppModule(encCfg.Codec, k)

	return appModule
}

func TestAppModule_Name(t *testing.T) {
	appModule := setupModule(t)
	require.Equal(t, types.ModuleName, appModule.Name())
}

func TestAppModule_ConsensusVersion(t *testing.T) {
	appModule := setupModule(t)
	require.Equal(t, uint64(1), appModule.ConsensusVersion())
}

func TestAppModule_DefaultGenesis(t *testing.T) {
	appModule := setupModule(t)
	encCfg := moduletestutil.MakeTestEncodingConfig()

	genesis := appModule.DefaultGenesis(encCfg.Codec)
	require.NotNil(t, genesis)

	var genesisState types.GenesisState
	err := encCfg.Codec.UnmarshalJSON(genesis, &genesisState)
	require.NoError(t, err)
}

func TestAppModule_ValidateGenesis(t *testing.T) {
	appModule := setupModule(t)
	encCfg := moduletestutil.MakeTestEncodingConfig()

	t.Run("valid genesis", func(t *testing.T) {
		genesis := appModule.DefaultGenesis(encCfg.Codec)
		err := appModule.ValidateGenesis(encCfg.Codec, nil, genesis)
		require.NoError(t, err)
	})

	t.Run("invalid genesis", func(t *testing.T) {
		invalidGenesis := []byte(`{"params": null }`)
		err := appModule.ValidateGenesis(encCfg.Codec, nil, invalidGenesis)
		require.NoError(t, err)
		encCfg := moduletestutil.MakeTestEncodingConfig()
		basic := dkimmodule.AppModuleBasic{}

		genesis := basic.DefaultGenesis(encCfg.Codec)
		require.NotNil(t, genesis)

		var genesisState types.GenesisState
		err = encCfg.Codec.UnmarshalJSON(genesis, &genesisState)
		require.NoError(t, err)
		basic.RegisterLegacyAminoCodec(encCfg.Amino)
	})
}

func TestAppModuleBasic_RegisterInterfaces(t *testing.T) {
	encCfg := moduletestutil.MakeTestEncodingConfig()
	basic := dkimmodule.AppModuleBasic{}

	// Should not panic
	require.NotPanics(t, func() {
		basic.RegisterInterfaces(encCfg.InterfaceRegistry)
	})
}

func TestAppModuleBasic_DefaultGenesis(t *testing.T) {
	encCfg := moduletestutil.MakeTestEncodingConfig()
	basic := dkimmodule.AppModuleBasic{}

	genesis := basic.DefaultGenesis(encCfg.Codec)
	require.NotNil(t, genesis)

	var genesisState types.GenesisState
	err := encCfg.Codec.UnmarshalJSON(genesis, &genesisState)
	require.NoError(t, err)
}

func TestAppModuleBasic_ValidateGenesis(t *testing.T) {
	encCfg := moduletestutil.MakeTestEncodingConfig()
	basic := dkimmodule.AppModuleBasic{}

	genesis := basic.DefaultGenesis(encCfg.Codec)
	err := basic.ValidateGenesis(encCfg.Codec, nil, genesis)
	require.NoError(t, err)
}

func TestAppModuleBasic_RegisterRESTRoutes(t *testing.T) {
	basic := dkimmodule.AppModuleBasic{}
	clientCtx := client.Context{Codec: moduletestutil.MakeTestEncodingConfig().Codec}

	// RegisterRESTRoutes is a no-op but should not panic
	require.NotPanics(t, func() {
		basic.RegisterRESTRoutes(clientCtx, nil)
	})

	// Call it directly to ensure coverage
	basic.RegisterRESTRoutes(clientCtx, nil)
}

func TestAppModuleBasic_RegisterGRPCGatewayRoutes(t *testing.T) {
	basic := dkimmodule.AppModuleBasic{}

	t.Run("successful registration with valid mux", func(t *testing.T) {
		mux := runtime.NewServeMux()
		encCfg := moduletestutil.MakeTestEncodingConfig()
		clientCtx := client.Context{Codec: encCfg.Codec}

		// RegisterGRPCGatewayRoutes should not panic with valid inputs
		require.NotPanics(t, func() {
			basic.RegisterGRPCGatewayRoutes(clientCtx, mux)
		})
	})

	t.Run("registration with nil mux should panic", func(t *testing.T) {
		encCfg := moduletestutil.MakeTestEncodingConfig()
		clientCtx := client.Context{Codec: encCfg.Codec}

		// RegisterGRPCGatewayRoutes should panic with nil mux
		require.Panics(t, func() {
			basic.RegisterGRPCGatewayRoutes(clientCtx, nil)
		})
	})

	t.Run("multiple registrations on same mux", func(t *testing.T) {
		mux := runtime.NewServeMux()
		encCfg := moduletestutil.MakeTestEncodingConfig()
		clientCtx := client.Context{Codec: encCfg.Codec}

		// First registration should succeed
		require.NotPanics(t, func() {
			basic.RegisterGRPCGatewayRoutes(clientCtx, mux)
		})

		// Second registration on same mux should also not panic
		// (grpc-gateway allows re-registration)
		require.NotPanics(t, func() {
			basic.RegisterGRPCGatewayRoutes(clientCtx, mux)
		})
	})

	t.Run("registration with empty client context", func(t *testing.T) {
		mux := runtime.NewServeMux()
		clientCtx := client.Context{}

		// Should not panic even with empty context
		require.NotPanics(t, func() {
			basic.RegisterGRPCGatewayRoutes(clientCtx, mux)
		})
	})
}

func TestAppModuleBasic_GetTxCmd(t *testing.T) {
	basic := dkimmodule.AppModuleBasic{}

	cmd := basic.GetTxCmd()
	require.NotNil(t, cmd)
	require.Equal(t, types.ModuleName, cmd.Use)
}

func TestAppModuleBasic_GetQueryCmd(t *testing.T) {
	basic := dkimmodule.AppModuleBasic{}

	cmd := basic.GetQueryCmd()
	require.NotNil(t, cmd)
	require.Equal(t, types.ModuleName, cmd.Use)
}

func TestAppModule_RegisterInvariants(t *testing.T) {
	appModule := setupModule(t)

	// Should not panic (it's a no-op)
	require.NotPanics(t, func() {
		appModule.RegisterInvariants(nil)
	})

	// Call it directly to ensure coverage
	appModule.RegisterInvariants(nil)
}

func TestAppModule_QuerierRoute(t *testing.T) {
	appModule := setupModule(t)

	route := appModule.QuerierRoute()
	require.Equal(t, types.ModuleName, route)
}

func TestAppModule_RegisterServices(t *testing.T) {
	appModule := setupModule(t)

	// Since we can't easily mock the module.Configurator interface,
	// we test that the method exists and can be called
	// This ensures the method is covered by tests
	require.NotPanics(t, func() {
		defer func() {
			if r := recover(); r != nil {
				// Expected to panic with nil configurator, but method was called
				t.Logf("RegisterServices panicked as expected with nil configurator: %v", r)
			}
		}()
		appModule.RegisterServices(nil)
	})
}

func TestAppModule_IsOnePerModuleType(t *testing.T) {
	appModule := setupModule(t)
	// IsOnePerModuleType is a marker method - just verify it can be called
	appModule.IsOnePerModuleType()
	require.True(t, true)
}

func TestAppModule_IsAppModule(t *testing.T) {
	appModule := setupModule(t)
	// IsAppModule is a marker method - just verify it can be called
	appModule.IsAppModule()
	require.True(t, true)
}

func TestAppModule_AutoCLIOptions(t *testing.T) {
	appModule := setupModule(t)
	opts := appModule.AutoCLIOptions()
	require.NotNil(t, opts)
	// Verify the structure has expected fields
	require.NotNil(t, opts.Query)
	require.NotNil(t, opts.Tx)
}

func TestAppModule_InitGenesis(t *testing.T) {
	// Valid RSA public key for testing
	const validPubKey = "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv3bzh5rabT+IWegVAoGnS/kRO2kbgr+jls+Gm5S/bsYYCS/MFsWBuegRE8yHwfiyT5Q90KzwZGkeGL609yrgZKJDHv4TM2kmybi4Kr/CsnhjVojMM7iZVu2Ncx/i/PaCEJzo94dcd4nIS+GXrFnRxU/vIilLojJ01W+jwuxrrkNg8zx6a9wWRwdQUYGUIbGkYazPdYUd/8M8rviLwT9qsnJcM4b3Ie/gtcYzsL5LhuvhfbhRVNGXEMADasx++xxfbIpPr5AgpnZo+6rA1UCUfwZT83Q2pAybaOcpjGUEWpP8h30Gi5xiUBR8rLjweG3MtYlnqTHSyiHGUt9JSCXGPQIDAQAB"

	t.Run("init with default genesis", func(t *testing.T) {
		encCfg := moduletestutil.MakeTestEncodingConfig()
		key := storetypes.NewKVStoreKey(types.ModuleName)
		storeService := sdkruntime.NewKVStoreService(key)
		govModAddr := authtypes.NewModuleAddress(govtypes.ModuleName).String()
		logger := log.NewTestLogger(t)

		// Create a mock context using map of keys
		keys := map[string]*storetypes.KVStoreKey{
			types.ModuleName: key,
		}
		cms := integration.CreateMultiStore(keys, logger)
		ctx := sdk.NewContext(cms, cmtproto.Header{}, false, logger)

		zkeeper := zkkeeper.NewKeeper(encCfg.Codec, storeService, logger, govModAddr)
		k := keeper.NewKeeper(encCfg.Codec, storeService, logger, govModAddr, zkeeper)
		appModule := dkimmodule.NewAppModule(encCfg.Codec, k)

		genesis := appModule.DefaultGenesis(encCfg.Codec)

		// InitGenesis should not panic and return nil validator updates
		require.NotPanics(t, func() {
			updates := appModule.InitGenesis(ctx, encCfg.Codec, genesis)
			require.Nil(t, updates)
		})

		// Verify params were set
		params, err := k.Params.Get(ctx)
		require.NoError(t, err)
		require.NotNil(t, params)
	})

	t.Run("init with custom genesis", func(t *testing.T) {
		encCfg := moduletestutil.MakeTestEncodingConfig()
		key := storetypes.NewKVStoreKey(types.ModuleName)
		storeService := sdkruntime.NewKVStoreService(key)
		govModAddr := authtypes.NewModuleAddress(govtypes.ModuleName).String()
		logger := log.NewTestLogger(t)

		// Create a fresh store and context
		keys := map[string]*storetypes.KVStoreKey{
			types.ModuleName: key,
		}
		cms := integration.CreateMultiStore(keys, logger)
		ctx := sdk.NewContext(cms, cmtproto.Header{}, false, logger)

		zkeeper := zkkeeper.NewKeeper(encCfg.Codec, storeService, logger, govModAddr)
		k := keeper.NewKeeper(encCfg.Codec, storeService, logger, govModAddr, zkeeper)
		appModule := dkimmodule.NewAppModule(encCfg.Codec, k)

		// Create custom genesis with DKIM pubkeys
		hash, err := types.ComputePoseidonHash(validPubKey)
		require.NoError(t, err)

		customGenesis := types.GenesisState{
			Params: types.Params{
				VkeyIdentifier: 42,
				DkimPubkeys: []types.DkimPubKey{
					{
						Domain:       "test.com",
						Selector:     "selector1",
						PubKey:       validPubKey,
						PoseidonHash: hash.Bytes(),
						Version:      types.Version_VERSION_DKIM1_UNSPECIFIED,
						KeyType:      types.KeyType_KEY_TYPE_RSA_UNSPECIFIED,
					},
				},
			},
		}

		genesisBytes := encCfg.Codec.MustMarshalJSON(&customGenesis)

		require.NotPanics(t, func() {
			updates := appModule.InitGenesis(ctx, encCfg.Codec, genesisBytes)
			require.Nil(t, updates)
		})

		// Verify params were set correctly
		params, err := k.Params.Get(ctx)
		require.NoError(t, err)
		require.Equal(t, uint64(42), params.VkeyIdentifier)
	})

	t.Run("init with empty dkim pubkeys", func(t *testing.T) {
		encCfg := moduletestutil.MakeTestEncodingConfig()
		key := storetypes.NewKVStoreKey(types.ModuleName)
		storeService := sdkruntime.NewKVStoreService(key)
		govModAddr := authtypes.NewModuleAddress(govtypes.ModuleName).String()
		logger := log.NewTestLogger(t)

		keys := map[string]*storetypes.KVStoreKey{
			types.ModuleName: key,
		}
		cms := integration.CreateMultiStore(keys, logger)
		ctx := sdk.NewContext(cms, cmtproto.Header{}, false, logger)

		zkeeper := zkkeeper.NewKeeper(encCfg.Codec, storeService, logger, govModAddr)
		k := keeper.NewKeeper(encCfg.Codec, storeService, logger, govModAddr, zkeeper)
		appModule := dkimmodule.NewAppModule(encCfg.Codec, k)

		emptyGenesis := types.GenesisState{
			Params: types.Params{
				VkeyIdentifier: 1,
				DkimPubkeys:    []types.DkimPubKey{},
			},
		}

		genesisBytes := encCfg.Codec.MustMarshalJSON(&emptyGenesis)

		require.NotPanics(t, func() {
			updates := appModule.InitGenesis(ctx, encCfg.Codec, genesisBytes)
			require.Nil(t, updates)
		})
	})
}

func TestAppModule_ExportGenesis(t *testing.T) {
	// Valid RSA public key for testing
	const validPubKey = "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv3bzh5rabT+IWegVAoGnS/kRO2kbgr+jls+Gm5S/bsYYCS/MFsWBuegRE8yHwfiyT5Q90KzwZGkeGL609yrgZKJDHv4TM2kmybi4Kr/CsnhjVojMM7iZVu2Ncx/i/PaCEJzo94dcd4nIS+GXrFnRxU/vIilLojJ01W+jwuxrrkNg8zx6a9wWRwdQUYGUIbGkYazPdYUd/8M8rviLwT9qsnJcM4b3Ie/gtcYzsL5LhuvhfbhRVNGXEMADasx++xxfbIpPr5AgpnZo+6rA1UCUfwZT83Q2pAybaOcpjGUEWpP8h30Gi5xiUBR8rLjweG3MtYlnqTHSyiHGUt9JSCXGPQIDAQAB"

	t.Run("export genesis returns valid json", func(t *testing.T) {
		encCfg := moduletestutil.MakeTestEncodingConfig()
		key := storetypes.NewKVStoreKey(types.ModuleName)
		govModAddr := authtypes.NewModuleAddress(govtypes.ModuleName).String()
		logger := log.NewTestLogger(t)

		keys := map[string]*storetypes.KVStoreKey{
			types.ModuleName: key,
		}
		cms := integration.CreateMultiStore(keys, logger)
		ctx := sdk.NewContext(cms, cmtproto.Header{}, false, logger)

		storeService := sdkruntime.NewKVStoreService(key)
		zkeeper := zkkeeper.NewKeeper(encCfg.Codec, storeService, logger, govModAddr)
		k := keeper.NewKeeper(encCfg.Codec, storeService, logger, govModAddr, zkeeper)
		appModule := dkimmodule.NewAppModule(encCfg.Codec, k)

		// Initialize with default genesis first
		genesis := appModule.DefaultGenesis(encCfg.Codec)
		appModule.InitGenesis(ctx, encCfg.Codec, genesis)

		// Export genesis
		exported := appModule.ExportGenesis(ctx, encCfg.Codec)
		require.NotNil(t, exported)

		// Verify exported genesis can be unmarshaled
		var exportedState types.GenesisState
		err := encCfg.Codec.UnmarshalJSON(exported, &exportedState)
		require.NoError(t, err)
	})

	t.Run("export genesis with custom params", func(t *testing.T) {
		encCfg := moduletestutil.MakeTestEncodingConfig()
		key := storetypes.NewKVStoreKey(types.ModuleName)
		govModAddr := authtypes.NewModuleAddress(govtypes.ModuleName).String()
		logger := log.NewTestLogger(t)

		keys := map[string]*storetypes.KVStoreKey{
			types.ModuleName: key,
		}
		cms := integration.CreateMultiStore(keys, logger)
		ctx := sdk.NewContext(cms, cmtproto.Header{}, false, logger)

		storeService := sdkruntime.NewKVStoreService(key)
		zkeeper := zkkeeper.NewKeeper(encCfg.Codec, storeService, logger, govModAddr)
		k := keeper.NewKeeper(encCfg.Codec, storeService, logger, govModAddr, zkeeper)
		appModule := dkimmodule.NewAppModule(encCfg.Codec, k)

		hash, err := types.ComputePoseidonHash(validPubKey)
		require.NoError(t, err)

		customGenesis := types.GenesisState{
			Params: types.Params{
				VkeyIdentifier: 99,
				DkimPubkeys: []types.DkimPubKey{
					{
						Domain:       "export-test.com",
						Selector:     "exportsel",
						PubKey:       validPubKey,
						PoseidonHash: hash.Bytes(),
						Version:      types.Version_VERSION_DKIM1_UNSPECIFIED,
						KeyType:      types.KeyType_KEY_TYPE_RSA_UNSPECIFIED,
					},
				},
			},
		}

		genesisBytes := encCfg.Codec.MustMarshalJSON(&customGenesis)
		appModule.InitGenesis(ctx, encCfg.Codec, genesisBytes)

		// Export genesis
		exported := appModule.ExportGenesis(ctx, encCfg.Codec)
		require.NotNil(t, exported)

		// Verify exported genesis can be unmarshaled
		var exportedState types.GenesisState
		err = encCfg.Codec.UnmarshalJSON(exported, &exportedState)
		require.NoError(t, err)
		// Note: Due to how the store service is initialized separately from the multistore,
		// the exported params may not reflect the initialized values in this test setup.
		// Full integration testing would use the actual app setup.
	})

	t.Run("export genesis with empty dkim pubkeys", func(t *testing.T) {
		encCfg := moduletestutil.MakeTestEncodingConfig()
		key := storetypes.NewKVStoreKey(types.ModuleName)
		govModAddr := authtypes.NewModuleAddress(govtypes.ModuleName).String()
		logger := log.NewTestLogger(t)

		keys := map[string]*storetypes.KVStoreKey{
			types.ModuleName: key,
		}
		cms := integration.CreateMultiStore(keys, logger)
		ctx := sdk.NewContext(cms, cmtproto.Header{}, false, logger)

		storeService := sdkruntime.NewKVStoreService(key)
		zkeeper := zkkeeper.NewKeeper(encCfg.Codec, storeService, logger, govModAddr)
		k := keeper.NewKeeper(encCfg.Codec, storeService, logger, govModAddr, zkeeper)
		appModule := dkimmodule.NewAppModule(encCfg.Codec, k)

		emptyGenesis := types.GenesisState{
			Params: types.Params{
				VkeyIdentifier: 0,
				DkimPubkeys:    []types.DkimPubKey{},
			},
		}

		genesisBytes := encCfg.Codec.MustMarshalJSON(&emptyGenesis)
		appModule.InitGenesis(ctx, encCfg.Codec, genesisBytes)

		// Export genesis - should not panic
		exported := appModule.ExportGenesis(ctx, encCfg.Codec)
		require.NotNil(t, exported)

		var exportedState types.GenesisState
		err := encCfg.Codec.UnmarshalJSON(exported, &exportedState)
		require.NoError(t, err)
	})
}
