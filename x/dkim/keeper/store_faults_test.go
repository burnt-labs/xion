package keeper_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"

	"github.com/burnt-labs/xion/x/dkim/keeper"
	"github.com/burnt-labs/xion/x/dkim/types"
)

var errStoreFault = errors.New("injected store fault")

// faultyStoreService wraps a KVStoreService and, when armed, fails writes,
// deletes or iterator creation for keys under failPrefix. With nothing armed
// it is transparent.
type faultyStoreService struct {
	inner        store.KVStoreService
	failPrefix   []byte
	failSet      bool
	failDelete   bool
	failIterator bool
}

func (s *faultyStoreService) OpenKVStore(ctx context.Context) store.KVStore {
	return &faultyStore{KVStore: s.inner.OpenKVStore(ctx), svc: s}
}

// arm makes every operation listed fail for keys under prefix.
func (s *faultyStoreService) arm(prefix collections.Prefix, set, del, iterate bool) {
	s.failPrefix = prefix.Bytes()
	s.failSet, s.failDelete, s.failIterator = set, del, iterate
}

func (s *faultyStoreService) disarm() {
	s.failSet, s.failDelete, s.failIterator = false, false, false
}

type faultyStore struct {
	store.KVStore
	svc *faultyStoreService
}

func (s *faultyStore) Set(key, value []byte) error {
	if s.svc.failSet && bytes.HasPrefix(key, s.svc.failPrefix) {
		return errStoreFault
	}
	return s.KVStore.Set(key, value)
}

func (s *faultyStore) Delete(key []byte) error {
	if s.svc.failDelete && bytes.HasPrefix(key, s.svc.failPrefix) {
		return errStoreFault
	}
	return s.KVStore.Delete(key)
}

func (s *faultyStore) Iterator(start, end []byte) (store.Iterator, error) {
	if s.svc.failIterator && bytes.HasPrefix(start, s.svc.failPrefix) {
		return nil, errStoreFault
	}
	return s.KVStore.Iterator(start, end)
}

func exampleDkimRecord() types.DkimPubKey {
	return types.DkimPubKey{
		Domain:   "example.com",
		Selector: "s1",
		PubKey:   testPubKey,
		Version:  types.Version_VERSION_DKIM1_UNSPECIFIED,
		KeyType:  types.KeyType_KEY_TYPE_RSA_UNSPECIFIED,
	}
}

func TestStoreWriteFailures(t *testing.T) {
	t.Run("saving a pubkey", func(t *testing.T) {
		f := SetupTest(t)
		f.faults.arm(types.DkimPrefix, true, false, false)

		saved, err := keeper.SaveDkimPubKey(f.ctx, exampleDkimRecord(), &f.k)
		require.False(t, saved)
		require.ErrorIs(t, err, errStoreFault)

		saved, err = keeper.SaveDkimPubKeys(f.ctx, []types.DkimPubKey{exampleDkimRecord()}, &f.k)
		require.False(t, saved)
		require.ErrorIs(t, err, errStoreFault)

		_, err = f.msgServer.AddDkimPubKeys(f.ctx, &types.MsgAddDkimPubKeys{
			Authority:   f.govModAddr,
			DkimPubkeys: []types.DkimPubKey{exampleDkimRecord()},
		})
		require.ErrorIs(t, err, errStoreFault)
	})

	t.Run("removing a pubkey", func(t *testing.T) {
		f := SetupTest(t)
		_, err := keeper.SaveDkimPubKey(f.ctx, exampleDkimRecord(), &f.k)
		require.NoError(t, err)
		f.faults.arm(types.DkimPrefix, false, true, false)

		_, err = f.msgServer.RemoveDkimPubKey(f.ctx, &types.MsgRemoveDkimPubKey{
			Authority: f.govModAddr,
			Domain:    "example.com",
			Selector:  "s1",
		})
		require.ErrorIs(t, err, errStoreFault)
	})

	t.Run("init genesis", func(t *testing.T) {
		genesis := func() *types.GenesisState {
			return &types.GenesisState{
				Params:         types.DefaultParams(),
				DkimPubkeys:    []types.DkimPubKey{exampleDkimRecord()},
				RevokedPubkeys: []string{base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32))},
			}
		}

		f := SetupTest(t)
		f.faults.arm(types.ParamsKey, true, false, false)
		require.ErrorIs(t, f.k.InitGenesis(f.ctx, genesis()), errStoreFault)

		f.faults.arm(types.DkimRevokedPrefix, true, false, false)
		require.ErrorIs(t, f.k.InitGenesis(f.ctx, genesis()), errStoreFault)

		f.faults.arm(types.DkimPrefix, true, false, false)
		require.ErrorIs(t, f.k.InitGenesis(f.ctx, genesis()), errStoreFault)

		f.faults.disarm()
		require.NoError(t, f.k.InitGenesis(f.ctx, genesis()))
	})
}

func TestStoreIteratorFailures(t *testing.T) {
	pemKey := genRSAPrivKeyPEM(t)

	t.Run("listing pubkeys", func(t *testing.T) {
		f := SetupTest(t)
		f.faults.arm(types.DkimPrefix, false, false, true)

		_, err := f.queryServer.DkimPubKeys(f.ctx, &types.QueryDkimPubKeysRequest{})
		require.ErrorIs(t, err, errStoreFault)

		_, err = f.msgServer.RevokeDkimPubKey(f.ctx, &types.MsgRevokeDkimPubKey{
			Signer:  f.addrs[0].String(),
			Domain:  "example.com",
			PrivKey: pemKey,
		})
		require.ErrorIs(t, err, errStoreFault)

		require.PanicsWithError(t, errStoreFault.Error(), func() { f.k.ExportGenesis(f.ctx) })
	})

	t.Run("listing revoked keys", func(t *testing.T) {
		f := SetupTest(t)
		f.faults.arm(types.DkimRevokedPrefix, false, false, true)

		require.PanicsWithError(t, errStoreFault.Error(), func() { f.k.ExportGenesis(f.ctx) })
	})
}

func TestRevokeDkimPubKeyStoreFailures(t *testing.T) {
	pemKey, pubKey := genRSAKeyPair(t)

	// store a key matching the private key so that revocation has work to do
	f := SetupTest(t)
	record := exampleDkimRecord()
	record.PubKey = pubKey
	_, err := keeper.SaveDkimPubKey(f.ctx, record, &f.k)
	require.NoError(t, err)

	t.Run("removing the matched key", func(t *testing.T) {
		f.faults.arm(types.DkimPrefix, false, true, false)
		defer f.faults.disarm()
		_, err := f.msgServer.RevokeDkimPubKey(f.ctx, &types.MsgRevokeDkimPubKey{
			Signer:  f.addrs[0].String(),
			Domain:  "example.com",
			PrivKey: pemKey,
		})
		require.ErrorIs(t, err, errStoreFault)
	})

	t.Run("recording the revocation", func(t *testing.T) {
		f.faults.arm(types.DkimRevokedPrefix, true, false, false)
		defer f.faults.disarm()
		_, err := f.msgServer.RevokeDkimPubKey(f.ctx, &types.MsgRevokeDkimPubKey{
			Signer:  f.addrs[0].String(),
			Domain:  "example.com",
			PrivKey: pemKey,
		})
		require.ErrorIs(t, err, errStoreFault)
	})

	t.Run("revocation succeeds once the store recovers", func(t *testing.T) {
		_, err := f.msgServer.RevokeDkimPubKey(f.ctx, &types.MsgRevokeDkimPubKey{
			Signer:  f.addrs[0].String(),
			Domain:  "example.com",
			PrivKey: pemKey,
		})
		require.NoError(t, err)
		_, err = f.k.DkimPubKeys.Get(f.ctx, collections.Join("example.com", "s1"))
		require.ErrorIs(t, err, collections.ErrNotFound)
	})
}

func TestInitGenesisDefaultsMaxPubkeySize(t *testing.T) {
	f := SetupTest(t)
	p := types.DefaultParams()
	p.MaxPubkeySizeBytes = 0

	require.NoError(t, f.k.InitGenesis(f.ctx, &types.GenesisState{Params: p}))

	stored, err := f.k.GetParams(f.ctx)
	require.NoError(t, err)
	require.Equal(t, types.DefaultMaxPubKeySizeBytes, stored.MaxPubkeySizeBytes)
}
