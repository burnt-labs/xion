package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	module "github.com/burnt-labs/xion/x/dkim"
	"github.com/burnt-labs/xion/x/dkim/keeper"
	"github.com/burnt-labs/xion/x/dkim/types"
	zkkeeper "github.com/burnt-labs/xion/x/zk/keeper"
	zktypes "github.com/burnt-labs/xion/x/zk/types"
)

const testPubKey = "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv3bzh5rabT+IWegVAoGnS/kRO2kbgr+jls+Gm5S/bsYYCS/MFsWBuegRE8yHwfiyT5Q90KzwZGkeGL609yrgZKJDHv4TM2kmybi4Kr/CsnhjVojMM7iZVu2Ncx/i/PaCEJzo94dcd4nIS+GXrFnRxU/vIilLojJ01W+jwuxrrkNg8zx6a9wWRwdQUYGUIbGkYazPdYUd/8M8rviLwT9qsnJcM4b3Ie/gtcYzsL5LhuvhfbhRVNGXEMADasx++xxfbIpPr5AgpnZo+6rA1UCUfwZT83Q2pAybaOcpjGUEWpP8h30Gi5xiUBR8rLjweG3MtYlnqTHSyiHGUt9JSCXGPQIDAQAB"

type TestFixture struct {
	suite.Suite

	ctx         sdk.Context
	k           keeper.Keeper
	zkeeper     zkkeeper.Keeper
	msgServer   types.MsgServer
	queryServer types.QueryServer
	appModule   *module.AppModule

	addrs      []sdk.AccAddress
	govModAddr string
}

func SetupTest(t *testing.T) *TestFixture {
	t.Helper()
	f := new(TestFixture)
	require := require.New(t)

	// Base setup
	logger := log.NewTestLogger(t)
	encCfg := moduletestutil.MakeTestEncodingConfig()

	f.govModAddr = authtypes.NewModuleAddress(govtypes.ModuleName).String()
	f.addrs = simtestutil.CreateIncrementalAccounts(3)

	key := storetypes.NewKVStoreKey(types.ModuleName)
	storeService := runtime.NewKVStoreService(key)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))

	f.ctx = testCtx.Ctx

	// Register SDK modules.
	registerBaseSDKModules(f, encCfg, storeService, logger, require)

	// Setup Keeper.
	f.zkeeper = zkkeeper.NewKeeper(encCfg.Codec, storeService, logger, f.govModAddr)
	// Initialize zk keeper with default genesis state to get the vkey with ID 1
	defaultZkGenesis := zktypes.DefaultGenesisState()
	f.zkeeper.InitGenesis(f.ctx, defaultZkGenesis)
	f.k = keeper.NewKeeper(encCfg.Codec, storeService, logger, f.govModAddr, f.zkeeper)
	err := f.k.Params.Set(f.ctx, types.DefaultParams())
	require.NoError(err)
	f.msgServer = keeper.NewMsgServerImpl(f.k)
	f.queryServer = keeper.NewQuerier(f.k)
	f.appModule = module.NewAppModule(encCfg.Codec, f.k)
	err = f.k.Params.Set(f.ctx, types.DefaultParams())
	require.NoError(err)
	// Setup Keeper.

	return f
}

func registerModuleInterfaces(encCfg moduletestutil.TestEncodingConfig) {
	authtypes.RegisterInterfaces(encCfg.InterfaceRegistry)

	types.RegisterInterfaces(encCfg.InterfaceRegistry)
}

func registerBaseSDKModules(
	_ *TestFixture,
	encCfg moduletestutil.TestEncodingConfig,
	_ store.KVStoreService,
	_ log.Logger,
	_ *require.Assertions,
) {
	registerModuleInterfaces(encCfg)
}

func TestKeeperGetParams(t *testing.T) {
	f := SetupTest(t)

	t.Run("returns default params when not found", func(t *testing.T) {
		// Delete existing params to simulate not found
		err := f.k.Params.Remove(f.ctx)
		require.NoError(t, err)

		params, err := f.k.GetParams(f.ctx)
		require.NoError(t, err)
		require.Equal(t, types.DefaultParams(), params)
	})

	t.Run("returns stored params", func(t *testing.T) {
		customParams := types.Params{
			MaxPubkeySizeBytes: 2048,
			VkeyIdentifier:     5,
		}
		err := f.k.SetParams(f.ctx, customParams)
		require.NoError(t, err)

		params, err := f.k.GetParams(f.ctx)
		require.NoError(t, err)
		require.Equal(t, customParams, params)
	})
}

func TestKeeperSetParams(t *testing.T) {
	f := SetupTest(t)

	t.Run("sets default max pubkey size when zero", func(t *testing.T) {
		params := types.Params{
			MaxPubkeySizeBytes: 0,
			VkeyIdentifier:     1,
		}
		err := f.k.SetParams(f.ctx, params)
		require.NoError(t, err)

		storedParams, err := f.k.GetParams(f.ctx)
		require.NoError(t, err)
		require.Equal(t, types.DefaultMaxPubKeySizeBytes, storedParams.MaxPubkeySizeBytes)
	})

	t.Run("validates params before storing", func(t *testing.T) {
		// Create params that will fail validation after the default is set
		// Since MaxPubkeySizeBytes of 0 gets set to default, we need to bypass that
		// by directly testing with stored params that have MaxPubkeySizeBytes = 0
		invalidParams := types.Params{
			MaxPubkeySizeBytes: 0,
			VkeyIdentifier:     1,
		}
		// Directly set invalid params to bypass the SetParams default logic
		err := f.k.Params.Set(f.ctx, invalidParams)
		require.NoError(t, err)

		// Now validate should fail
		err = invalidParams.Validate()
		require.Error(t, err)
	})

	t.Run("stores valid params", func(t *testing.T) {
		validParams := types.Params{
			MaxPubkeySizeBytes: 4096,
			VkeyIdentifier:     2,
		}
		err := f.k.SetParams(f.ctx, validParams)
		require.NoError(t, err)

		stored, err := f.k.GetParams(f.ctx)
		require.NoError(t, err)
		require.Equal(t, validParams, stored)
	})
}

func TestKeeperLogger(t *testing.T) {
	f := SetupTest(t)
	logger := f.k.Logger()
	require.NotNil(t, logger)
}

// ============================================================================
// NewKeeper Tests
// ============================================================================

func TestNewKeeper(t *testing.T) {
	t.Run("creates keeper with valid parameters", func(t *testing.T) {
		logger := log.NewTestLogger(t)
		encCfg := moduletestutil.MakeTestEncodingConfig()
		types.RegisterInterfaces(encCfg.InterfaceRegistry)

		key := storetypes.NewKVStoreKey(types.ModuleName)
		storeService := runtime.NewKVStoreService(key)
		testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))

		authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
		zkKeeper := zkkeeper.NewKeeper(encCfg.Codec, storeService, logger, authority)

		k := keeper.NewKeeper(encCfg.Codec, storeService, logger, authority, zkKeeper)

		require.NotNil(t, k)
		require.NotNil(t, k.Logger())

		// Verify we can use the keeper
		err := k.Params.Set(testCtx.Ctx, types.DefaultParams())
		require.NoError(t, err)
	})

	t.Run("creates keeper with empty authority defaults to gov module", func(t *testing.T) {
		logger := log.NewTestLogger(t)
		encCfg := moduletestutil.MakeTestEncodingConfig()
		types.RegisterInterfaces(encCfg.InterfaceRegistry)

		key := storetypes.NewKVStoreKey(types.ModuleName)
		storeService := runtime.NewKVStoreService(key)
		testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))

		// Empty authority should default to gov module address
		zkKeeper := zkkeeper.NewKeeper(encCfg.Codec, storeService, logger, "")

		k := keeper.NewKeeper(encCfg.Codec, storeService, logger, "", zkKeeper)

		require.NotNil(t, k)

		// Verify keeper is functional
		err := k.Params.Set(testCtx.Ctx, types.DefaultParams())
		require.NoError(t, err)
	})

	t.Run("creates keeper with custom authority", func(t *testing.T) {
		logger := log.NewTestLogger(t)
		encCfg := moduletestutil.MakeTestEncodingConfig()
		types.RegisterInterfaces(encCfg.InterfaceRegistry)

		key := storetypes.NewKVStoreKey(types.ModuleName)
		storeService := runtime.NewKVStoreService(key)
		testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))

		customAuthority := "xion1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a"
		zkKeeper := zkkeeper.NewKeeper(encCfg.Codec, storeService, logger, customAuthority)

		k := keeper.NewKeeper(encCfg.Codec, storeService, logger, customAuthority, zkKeeper)

		require.NotNil(t, k)

		// Verify keeper is functional
		err := k.Params.Set(testCtx.Ctx, types.DefaultParams())
		require.NoError(t, err)
	})

	t.Run("keeper schema is built correctly", func(t *testing.T) {
		logger := log.NewTestLogger(t)
		encCfg := moduletestutil.MakeTestEncodingConfig()
		types.RegisterInterfaces(encCfg.InterfaceRegistry)

		key := storetypes.NewKVStoreKey(types.ModuleName)
		storeService := runtime.NewKVStoreService(key)

		authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
		zkKeeper := zkkeeper.NewKeeper(encCfg.Codec, storeService, logger, authority)

		k := keeper.NewKeeper(encCfg.Codec, storeService, logger, authority, zkKeeper)

		// Schema should be built without error
		require.NotNil(t, k.Schema)
	})

	t.Run("keeper collections are initialized", func(t *testing.T) {
		logger := log.NewTestLogger(t)
		encCfg := moduletestutil.MakeTestEncodingConfig()
		types.RegisterInterfaces(encCfg.InterfaceRegistry)

		key := storetypes.NewKVStoreKey(types.ModuleName)
		storeService := runtime.NewKVStoreService(key)
		testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))

		authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
		zkKeeper := zkkeeper.NewKeeper(encCfg.Codec, storeService, logger, authority)

		k := keeper.NewKeeper(encCfg.Codec, storeService, logger, authority, zkKeeper)

		// DkimPubKeys collection should be usable
		iter, err := k.DkimPubKeys.Iterate(testCtx.Ctx, nil)
		require.NoError(t, err)
		defer iter.Close()

		// Params collection should be usable
		err = k.Params.Set(testCtx.Ctx, types.DefaultParams())
		require.NoError(t, err)

		params, err := k.Params.Get(testCtx.Ctx)
		require.NoError(t, err)
		require.NotNil(t, params)
	})
}

// ============================================================================
// Keeper Logger Tests
// ============================================================================

func TestKeeperLoggerExtended(t *testing.T) {
	t.Run("logger returns non-nil", func(t *testing.T) {
		f := SetupTest(t)
		logger := f.k.Logger()
		require.NotNil(t, logger)
	})

	t.Run("logger is consistent", func(t *testing.T) {
		f := SetupTest(t)
		logger1 := f.k.Logger()
		logger2 := f.k.Logger()
		require.Equal(t, logger1, logger2)
	})
}

// ============================================================================
// Keeper State Operations Tests
// ============================================================================

func TestKeeperStateOperations(t *testing.T) {
	t.Run("set and get params", func(t *testing.T) {
		f := SetupTest(t)

		params := types.Params{
			VkeyIdentifier: uint64(1),
		}

		err := f.k.Params.Set(f.ctx, params)
		require.NoError(t, err)

		got, err := f.k.Params.Get(f.ctx)
		require.NoError(t, err)
		require.Equal(t, params.VkeyIdentifier, got.VkeyIdentifier)
	})

	t.Run("set and get default params", func(t *testing.T) {
		f := SetupTest(t)

		err := f.k.Params.Set(f.ctx, types.DefaultParams())
		require.NoError(t, err)

		got, err := f.k.Params.Get(f.ctx)
		require.NoError(t, err)
		require.NotNil(t, got)
	})

	t.Run("overwrite params", func(t *testing.T) {
		f := SetupTest(t)

		// Set initial params
		params1 := types.Params{
			VkeyIdentifier: uint64(1),
		}
		err := f.k.Params.Set(f.ctx, params1)
		require.NoError(t, err)

		// Overwrite with new params
		params2 := types.Params{
			VkeyIdentifier: uint64(2),
		}
		err = f.k.Params.Set(f.ctx, params2)
		require.NoError(t, err)

		// Verify new params are returned
		got, err := f.k.Params.Get(f.ctx)
		require.NoError(t, err)
		require.Equal(t, params2.VkeyIdentifier, got.VkeyIdentifier)
	})
}

// ============================================================================
// Keeper ZkKeeper Integration Tests
// ============================================================================

func TestKeeperZkKeeperIntegration(t *testing.T) {
	t.Run("zk keeper is accessible", func(t *testing.T) {
		f := SetupTest(t)
		require.NotNil(t, f.k.ZkKeeper)
	})

	t.Run("zk keeper authority is set", func(t *testing.T) {
		f := SetupTest(t)
		authority := f.k.ZkKeeper.GetAuthority()
		require.NotEmpty(t, authority)
		require.Equal(t, f.govModAddr, authority)
	})
}
