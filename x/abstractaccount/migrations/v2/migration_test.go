package v2_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	xionapp "github.com/burnt-labs/xion/app"
	v2 "github.com/burnt-labs/xion/x/abstractaccount/migrations/v2"
	"github.com/burnt-labs/xion/x/abstractaccount/types"
)

type MigrationTestSuite struct {
	suite.Suite

	ctx sdk.Context
	app *xionapp.WasmApp
}

func TestMigrationTestSuite(t *testing.T) {
	suite.Run(t, new(MigrationTestSuite))
}

func (s *MigrationTestSuite) SetupTest() {
	s.app = xionapp.Setup(s.T())
	s.ctx = s.app.NewContext(false)
}

func (s *MigrationTestSuite) TestMigrateStore_Success() {
	// Setup: Set some initial params using the keeper
	initialParams := types.DefaultParams()
	initialParams.AllowAllCodeIDs = false
	initialParams.AllowedCodeIDs = []uint64{1, 2, 3}
	initialParams.MaxGasBefore = 1000000
	initialParams.MaxGasAfter = 2000000

	err := s.app.AbstractAccountKeeper.SetParams(s.ctx, initialParams)
	s.Require().NoError(err)

	// Execute migration using the keeper's migrator
	migrator := s.app.AbstractAccountKeeper.Migrator()
	err = migrator.Migrate1to2(s.ctx)
	s.Require().NoError(err)

	// Verify params are still accessible and unchanged
	migratedParams, err := s.app.AbstractAccountKeeper.GetParams(s.ctx)
	s.Require().NoError(err)

	s.Equal(initialParams.AllowAllCodeIDs, migratedParams.AllowAllCodeIDs)
	s.Equal(initialParams.AllowedCodeIDs, migratedParams.AllowedCodeIDs)
	s.Equal(initialParams.MaxGasBefore, migratedParams.MaxGasBefore)
	s.Equal(initialParams.MaxGasAfter, migratedParams.MaxGasAfter)
}

func (s *MigrationTestSuite) TestMigrateStore_NoExistingParams() {
	// The app starts with default params, so this tests that scenario
	// Execute migration
	migrator := s.app.AbstractAccountKeeper.Migrator()
	err := migrator.Migrate1to2(s.ctx)
	s.Require().NoError(err)

	// Verify default params are accessible
	params, err := s.app.AbstractAccountKeeper.GetParams(s.ctx)
	s.Require().NoError(err)

	defaultParams := types.DefaultParams()
	s.Equal(defaultParams.AllowAllCodeIDs, params.AllowAllCodeIDs)
	s.ElementsMatch(defaultParams.AllowedCodeIDs, params.AllowedCodeIDs)
	s.Equal(defaultParams.MaxGasBefore, params.MaxGasBefore)
	s.Equal(defaultParams.MaxGasAfter, params.MaxGasAfter)
}

// Test the v2.MigrateStore function directly with a mock store
func TestMigrateStoreDirect(t *testing.T) {
	tests := []struct {
		name        string
		setupStore  func(store storetypes.KVStore, cdc codec.BinaryCodec)
		expectError bool
		validate    func(t *testing.T, store storetypes.KVStore, cdc codec.BinaryCodec)
	}{
		{
			name: "existing valid params",
			setupStore: func(store storetypes.KVStore, cdc codec.BinaryCodec) {
				params := &types.Params{
					AllowAllCodeIDs: false,
					AllowedCodeIDs:  []uint64{1, 2},
					MaxGasBefore:    1000000,
					MaxGasAfter:     2000000,
				}
				bz, _ := cdc.Marshal(params)
				store.Set(types.KeyParams, bz)
			},
			expectError: false,
			validate: func(t *testing.T, store storetypes.KVStore, cdc codec.BinaryCodec) {
				bz := store.Get(types.KeyParams)
				require.NotNil(t, bz)

				var params types.Params
				err := cdc.Unmarshal(bz, &params)
				require.NoError(t, err)
				require.Equal(t, false, params.AllowAllCodeIDs)
				require.Equal(t, []uint64{1, 2}, params.AllowedCodeIDs)
			},
		},
		{
			name: "no existing params - sets defaults",
			setupStore: func(_ storetypes.KVStore, _ codec.BinaryCodec) {
				// Don't set anything
			},
			expectError: false,
			validate: func(t *testing.T, store storetypes.KVStore, cdc codec.BinaryCodec) {
				bz := store.Get(types.KeyParams)
				require.NotNil(t, bz)

				var params types.Params
				err := cdc.Unmarshal(bz, &params)
				require.NoError(t, err)

				defaultParams := types.DefaultParams()
				require.Equal(t, defaultParams.AllowAllCodeIDs, params.AllowAllCodeIDs)
				require.ElementsMatch(t, defaultParams.AllowedCodeIDs, params.AllowedCodeIDs)
			},
		},
		{
			name: "invalid params data",
			setupStore: func(store storetypes.KVStore, _ codec.BinaryCodec) {
				store.Set(types.KeyParams, []byte("invalid"))
			},
			expectError: true,
			validate:    func(_ *testing.T, _ storetypes.KVStore, _ codec.BinaryCodec) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := xionapp.Setup(t)
			ctx := app.NewContext(false)
			storeKey := storetypes.NewKVStoreKey("test-" + tt.name)

			// Create a basic multi-store for testing
			ms := app.CommitMultiStore()
			ms.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, nil)
			_ = ms.LoadLatestVersion()

			store := ctx.WithMultiStore(ms).KVStore(storeKey)
			cdc := app.AppCodec()

			tt.setupStore(store, cdc)

			// Execute migration
			err := v2.MigrateStore(ctx.WithMultiStore(ms), storeKey, cdc)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				tt.validate(t, store, cdc)
			}
		})
	}
}

// Test edge cases and error conditions
func TestMigrateStore_EdgeCases(t *testing.T) {
	tests := []struct {
		name          string
		setupParams   func() *types.Params
		expectError   bool
		errorContains string
	}{
		{
			name: "params with large code ID list",
			setupParams: func() *types.Params {
				codeIDs := make([]uint64, 1000)
				for i := range codeIDs {
					// nolint: gosec
					codeIDs[i] = uint64(i + 1)
				}
				return &types.Params{
					AllowAllCodeIDs: false,
					AllowedCodeIDs:  codeIDs,
					MaxGasBefore:    1000000,
					MaxGasAfter:     2000000,
				}
			},
			expectError:   false,
			errorContains: "",
		},
		{
			name: "params with maximum uint64 values",
			setupParams: func() *types.Params {
				return &types.Params{
					AllowAllCodeIDs: false,
					AllowedCodeIDs:  []uint64{^uint64(0)}, // Max uint64
					MaxGasBefore:    ^uint64(0),           // Max uint64
					MaxGasAfter:     ^uint64(0),           // Max uint64
				}
			},
			expectError:   false,
			errorContains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := xionapp.Setup(t)
			ctx := app.NewContext(false)
			storeKey := storetypes.NewKVStoreKey("test-edge-" + tt.name)

			// Create a basic multi-store for testing
			ms := app.CommitMultiStore()
			ms.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, nil)
			_ = ms.LoadLatestVersion()

			store := ctx.WithMultiStore(ms).KVStore(storeKey)
			cdc := app.AppCodec()

			// Setup initial params
			params := tt.setupParams()
			bz, err := cdc.Marshal(params)
			require.NoError(t, err)
			store.Set(types.KeyParams, bz)

			// Execute migration
			err = v2.MigrateStore(ctx.WithMultiStore(ms), storeKey, cdc)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					require.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)

				// Verify params were migrated correctly
				migratedBz := store.Get(types.KeyParams)
				require.NotNil(t, migratedBz)

				var migratedParams types.Params
				err = cdc.Unmarshal(migratedBz, &migratedParams)
				require.NoError(t, err)

				require.Equal(t, params.AllowAllCodeIDs, migratedParams.AllowAllCodeIDs)
				require.Equal(t, params.AllowedCodeIDs, migratedParams.AllowedCodeIDs)
				require.Equal(t, params.MaxGasBefore, migratedParams.MaxGasBefore)
				require.Equal(t, params.MaxGasAfter, migratedParams.MaxGasAfter)
			}
		})
	}
}

// Benchmark tests
func BenchmarkMigrateStore(b *testing.B) {
	app := xionapp.Setup(b)
	ctx := app.NewContext(false)
	storeKey := storetypes.NewKVStoreKey("benchmark")

	// Create a basic multi-store for testing
	ms := app.CommitMultiStore()
	ms.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, nil)
	_ = ms.LoadLatestVersion()

	cdc := app.AppCodec()
	testCtx := ctx.WithMultiStore(ms)

	// Setup initial params
	params := types.DefaultParams()
	store := testCtx.KVStore(storeKey)
	bz, _ := cdc.Marshal(params)
	store.Set(types.KeyParams, bz)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = v2.MigrateStore(testCtx, storeKey, cdc)
	}
}
