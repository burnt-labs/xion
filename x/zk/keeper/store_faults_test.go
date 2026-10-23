package keeper_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"cosmossdk.io/core/store"
)

var errStoreUnavailable = errors.New("store unavailable")

// storeFaults selects which store operations fail. When prefix is set only
// keys under that prefix are affected, so a test can break a single
// collection (params, vkeys, sequence, name index) while the others keep
// working.
type storeFaults struct {
	get, has, set, del, iter bool
	prefix                   []byte
}

func (f *storeFaults) hit(key []byte) bool {
	return f.prefix == nil || bytes.HasPrefix(key, f.prefix)
}

func (f *storeFaults) failGet(prefix []byte) { *f = storeFaults{get: true, prefix: prefix} }
func (f *storeFaults) failHas(prefix []byte) { *f = storeFaults{has: true, prefix: prefix} }
func (f *storeFaults) failSet(prefix []byte) { *f = storeFaults{set: true, prefix: prefix} }
func (f *storeFaults) failDel(prefix []byte) { *f = storeFaults{del: true, prefix: prefix} }
func (f *storeFaults) failIterators()        { *f = storeFaults{iter: true} }
func (f *storeFaults) clear()                { *f = storeFaults{} }

type faultyKVStoreService struct {
	inner  store.KVStoreService
	faults *storeFaults
}

func (s faultyKVStoreService) OpenKVStore(ctx context.Context) store.KVStore {
	return faultyKVStore{inner: s.inner.OpenKVStore(ctx), faults: s.faults}
}

type faultyKVStore struct {
	inner  store.KVStore
	faults *storeFaults
}

func (s faultyKVStore) Get(key []byte) ([]byte, error) {
	if s.faults.get && s.faults.hit(key) {
		return nil, errStoreUnavailable
	}
	return s.inner.Get(key)
}

func (s faultyKVStore) Has(key []byte) (bool, error) {
	if s.faults.has && s.faults.hit(key) {
		return false, errStoreUnavailable
	}
	return s.inner.Has(key)
}

func (s faultyKVStore) Set(key, value []byte) error {
	if s.faults.set && s.faults.hit(key) {
		return errStoreUnavailable
	}
	return s.inner.Set(key, value)
}

func (s faultyKVStore) Delete(key []byte) error {
	if s.faults.del && s.faults.hit(key) {
		return errStoreUnavailable
	}
	return s.inner.Delete(key)
}

func (s faultyKVStore) Iterator(start, end []byte) (store.Iterator, error) {
	if s.faults.iter {
		return nil, errStoreUnavailable
	}
	return s.inner.Iterator(start, end)
}

func (s faultyKVStore) ReverseIterator(start, end []byte) (store.Iterator, error) {
	if s.faults.iter {
		return nil, errStoreUnavailable
	}
	return s.inner.ReverseIterator(start, end)
}

// setupFaultyFixture returns a fixture whose keeper runs on a store that can
// be made to fail on demand through the returned storeFaults.
func setupFaultyFixture(t *testing.T) (*TestFixture, *storeFaults) {
	t.Helper()
	faults := &storeFaults{}
	f := setupFixture(t, func(inner store.KVStoreService) store.KVStoreService {
		return faultyKVStoreService{inner: inner, faults: faults}
	})
	return f, faults
}
