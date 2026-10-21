package v2_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	"cosmossdk.io/log/v2"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"

	v2 "github.com/burnt-labs/xion/x/dkim/migrations/v2"
	"github.com/burnt-labs/xion/x/dkim/types"
)

var errStoreFault = errors.New("injected store fault")

// failingWritesService wraps a KVStoreService so that writes fail while armed.
type failingWritesService struct {
	inner store.KVStoreService
	armed bool
}

func (s *failingWritesService) OpenKVStore(ctx context.Context) store.KVStore {
	return &failingWritesStore{KVStore: s.inner.OpenKVStore(ctx), svc: s}
}

type failingWritesStore struct {
	store.KVStore
	svc *failingWritesService
}

func (s *failingWritesStore) Set(key, value []byte) error {
	if s.svc.armed {
		return errStoreFault
	}
	return s.KVStore.Set(key, value)
}

func TestMigrateStoreWriteFailures(t *testing.T) {
	encCfg := moduletestutil.MakeTestEncodingConfig()
	types.RegisterInterfaces(encCfg.InterfaceRegistry)

	newCollections := func(t *testing.T) (*failingWritesService, collections.Item[types.Params], collections.Map[collections.Pair[string, string], types.DkimPubKey], sdk.Context) {
		t.Helper()
		key := storetypes.NewKVStoreKey(types.ModuleName)
		testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))
		svc := &failingWritesService{inner: runtime.NewKVStoreService(key)}
		sb := collections.NewSchemaBuilder(svc)
		params := collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](encCfg.Codec))
		pubKeys := collections.NewMap(
			sb,
			types.DkimPrefix,
			"dkim_pubkeys",
			collections.PairKeyCodec(collections.StringKey, collections.StringKey),
			codec.CollValue[types.DkimPubKey](encCfg.Codec),
		)
		_, err := sb.Build()
		require.NoError(t, err)
		return svc, params, pubKeys, testCtx.Ctx.WithLogger(log.NewNopLogger())
	}

	t.Run("defaults cannot be written when params are missing", func(t *testing.T) {
		svc, params, pubKeys, ctx := newCollections(t)
		svc.armed = true
		err := v2.MigrateStore(ctx, params, pubKeys)
		require.ErrorIs(t, err, errStoreFault)
	})

	t.Run("updated params cannot be written", func(t *testing.T) {
		svc, params, pubKeys, ctx := newCollections(t)
		require.NoError(t, params.Set(ctx, types.Params{VkeyIdentifier: 1, MaxPubkeySizeBytes: 2048}))
		svc.armed = true
		err := v2.MigrateStore(ctx, params, pubKeys)
		require.ErrorIs(t, err, errStoreFault)
	})
}
