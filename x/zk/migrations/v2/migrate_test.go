package v2_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"

	v2 "github.com/burnt-labs/xion/x/zk/migrations/v2"
	"github.com/burnt-labs/xion/x/zk/types"
)

func TestMigrateStore(t *testing.T) {
	encCfg := moduletestutil.MakeTestEncodingConfig()
	types.RegisterInterfaces(encCfg.InterfaceRegistry)

	key := storetypes.NewKVStoreKey(types.ModuleName)
	tkey := storetypes.NewTransientStoreKey("transient_test")
	testCtx := testutil.DefaultContextWithDB(t, key, tkey)
	ctx := testCtx.Ctx

	storeService := runtime.NewKVStoreService(key)
	sb := collections.NewSchemaBuilder(storeService)

	vkeys := collections.NewMap(sb, types.VKeyPrefix, "vkeys", collections.Uint64Key, codec.CollValue[types.VKey](encCfg.Codec))

	_, err := sb.Build()
	require.NoError(t, err)

	t.Run("migrate vkeys to update default vkey with id 1", func(t *testing.T) {
		// Set an old vkey for id 1
		oldVkey := types.VKey{}
		err := vkeys.Set(ctx, 1, oldVkey)
		require.NoError(t, err)

		// Perform migration
		err = v2.MigrateStore(ctx, vkeys)
		require.NoError(t, err)
		// Verify that the vkey with id 1 has been updated to the new default vkey
		newVkey, err := vkeys.Get(ctx, 1)
		require.NoError(t, err)
		defaultVkeys := types.DefaultGenesisState().Vkeys
		var expectedVkey types.VKey
		for _, vk := range defaultVkeys {
			if vk.Id == 1 {
				expectedVkey = vk.Vkey
				break
			}
		}
		require.Equal(t, expectedVkey.KeyBytes, newVkey.KeyBytes)
	})
}

var errWriteFailed = errors.New("write failed")

// unwritableStoreService serves a store whose writes always fail.
type unwritableStoreService struct {
	store.KVStoreService
}

func (s unwritableStoreService) OpenKVStore(ctx context.Context) store.KVStore {
	return unwritableStore{KVStore: s.KVStoreService.OpenKVStore(ctx)}
}

type unwritableStore struct {
	store.KVStore
}

func (unwritableStore) Set([]byte, []byte) error { return errWriteFailed }

func TestMigrateStoreWriteFailure(t *testing.T) {
	encCfg := moduletestutil.MakeTestEncodingConfig()
	types.RegisterInterfaces(encCfg.InterfaceRegistry)

	key := storetypes.NewKVStoreKey(types.ModuleName)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))

	sb := collections.NewSchemaBuilder(unwritableStoreService{KVStoreService: runtime.NewKVStoreService(key)})
	vkeys := collections.NewMap(sb, types.VKeyPrefix, "vkeys", collections.Uint64Key, codec.CollValue[types.VKey](encCfg.Codec))
	_, err := sb.Build()
	require.NoError(t, err)

	err = v2.MigrateStore(testCtx.Ctx, vkeys)
	require.ErrorContains(t, err, errWriteFailed.Error())
}
