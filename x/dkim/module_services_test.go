package module_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	grpc1 "github.com/cosmos/gogoproto/grpc"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log/v2"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"

	sdkruntime "github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	"github.com/cosmos/cosmos-sdk/types/module"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	dkimmodule "github.com/burnt-labs/xion/x/dkim"
	"github.com/burnt-labs/xion/x/dkim/keeper"
	"github.com/burnt-labs/xion/x/dkim/types"
	zkkeeper "github.com/burnt-labs/xion/x/zk/keeper"
)

type noopGRPCServer struct{}

func (noopGRPCServer) RegisterService(*grpc.ServiceDesc, interface{}) {}

func (noopGRPCServer) GetServiceInfo() map[string]grpc.ServiceInfo { return nil }

// stubConfigurator records migration registrations and can be told to reject them.
type stubConfigurator struct {
	noopGRPCServer
	migrationErr error
	registered   []uint64
}

func (c *stubConfigurator) MsgServer() grpc1.Server   { return c }
func (c *stubConfigurator) QueryServer() grpc1.Server { return c }
func (c *stubConfigurator) Error() error              { return nil }

func (c *stubConfigurator) RegisterMigration(_ string, fromVersion uint64, _ module.MigrationHandler) error {
	c.registered = append(c.registered, fromVersion)
	return c.migrationErr
}

func TestAppModule_RegisterServicesMigrations(t *testing.T) {
	appModule := setupModule(t)

	t.Run("registers the v1 to v2 migration", func(t *testing.T) {
		cfg := &stubConfigurator{}
		require.NotPanics(t, func() { appModule.RegisterServices(cfg) })
		require.Equal(t, []uint64{1}, cfg.registered)
	})

	t.Run("panics when the migration cannot be registered", func(t *testing.T) {
		cfg := &stubConfigurator{migrationErr: errors.New("already registered")}
		require.PanicsWithError(t, "already registered", func() { appModule.RegisterServices(cfg) })
	})
}

func TestAppModule_ValidateGenesisMalformedJSON(t *testing.T) {
	appModule := setupModule(t)
	encCfg := moduletestutil.MakeTestEncodingConfig()

	err := appModule.ValidateGenesis(encCfg.Codec, nil, []byte(`{"params": `))
	require.Error(t, err)
}

var errStoreFault = errors.New("injected store fault")

// failingWritesService wraps a KVStoreService so that every Set under
// failPrefix fails.
type failingWritesService struct {
	inner      store.KVStoreService
	failPrefix []byte
}

func (s *failingWritesService) OpenKVStore(ctx context.Context) store.KVStore {
	return &failingWritesStore{KVStore: s.inner.OpenKVStore(ctx), prefix: s.failPrefix}
}

type failingWritesStore struct {
	store.KVStore
	prefix []byte
}

func (s *failingWritesStore) Set(key, value []byte) error {
	if bytes.HasPrefix(key, s.prefix) {
		return errStoreFault
	}
	return s.KVStore.Set(key, value)
}

func TestAppModule_InitGenesisStoreFailures(t *testing.T) {
	encCfg := moduletestutil.MakeTestEncodingConfig()
	govModAddr := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	genesis := encCfg.Codec.MustMarshalJSON(types.DefaultGenesis())

	cases := []struct {
		name   string
		prefix []byte
	}{
		{"params cannot be written", types.ParamsKey.Bytes()},
		{"pubkeys cannot be written", types.DkimPrefix.Bytes()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			key := storetypes.NewKVStoreKey(types.ModuleName)
			testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))
			logger := log.NewTestLogger(t)
			storeService := &failingWritesService{inner: sdkruntime.NewKVStoreService(key), failPrefix: tc.prefix}

			zkeeper := zkkeeper.NewKeeper(encCfg.Codec, storeService, logger, govModAddr)
			k := keeper.NewKeeper(encCfg.Codec, storeService, logger, govModAddr, zkeeper)
			appModule := dkimmodule.NewAppModule(encCfg.Codec, k)

			require.PanicsWithError(t, errStoreFault.Error(), func() {
				appModule.InitGenesis(testCtx.Ctx, encCfg.Codec, genesis)
			})
		})
	}
}
