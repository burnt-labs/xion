package indexer

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	db "github.com/cosmos/cosmos-db"

	"cosmossdk.io/collections"
	"cosmossdk.io/collections/corecompat"
	"cosmossdk.io/log/v2"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
	feegrantv2 "github.com/cosmos/cosmos-sdk/x/feegrant/migrations/v2"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	indexerauthz "github.com/burnt-labs/xion/indexer/authz"
	indexerfeegrant "github.com/burnt-labs/xion/indexer/feegrant"
)

var errStoreFailure = errors.New("store failure")

// faultyStore wraps an in-memory db and lets individual operations be made to
// fail on demand. It doubles as its own KVStoreService so handlers can be built
// on top of it directly.
type faultyStore struct {
	db.DB
	failGet      bool
	failHas      bool
	failSet      bool
	failDelete   bool
	failIterator bool
}

func newFaultyStore() *faultyStore {
	return &faultyStore{DB: db.NewMemDB()}
}

func (s *faultyStore) OpenKVStore(_ context.Context) corecompat.KVStore {
	return s
}

func (s *faultyStore) Get(key []byte) ([]byte, error) {
	if s.failGet {
		return nil, errStoreFailure
	}
	return s.DB.Get(key)
}

func (s *faultyStore) Has(key []byte) (bool, error) {
	if s.failHas {
		return false, errStoreFailure
	}
	return s.DB.Has(key)
}

func (s *faultyStore) Set(key, value []byte) error {
	if s.failSet {
		return errStoreFailure
	}
	return s.DB.Set(key, value)
}

func (s *faultyStore) Delete(key []byte) error {
	if s.failDelete {
		return errStoreFailure
	}
	return s.DB.Delete(key)
}

func (s *faultyStore) Iterator(start, end []byte) (db.Iterator, error) {
	if s.failIterator {
		return nil, errStoreFailure
	}
	return s.DB.Iterator(start, end)
}

func (s *faultyStore) ReverseIterator(start, end []byte) (db.Iterator, error) {
	if s.failIterator {
		return nil, errStoreFailure
	}
	return s.DB.ReverseIterator(start, end)
}

func newSendGrant(t *testing.T) (authz.Grant, string) {
	t.Helper()
	sendAuth := banktypes.NewSendAuthorization(sdk.NewCoins(sdk.NewInt64Coin("uxion", 1000)), nil)
	expiration := time.Now().Add(24 * time.Hour)
	grant, err := authz.NewGrant(expiration, sendAuth, nil)
	require.NoError(t, err)
	return grant, sendAuth.MsgTypeURL()
}

func newBasicAllowance(t *testing.T, granter, grantee string) feegrant.Grant {
	t.Helper()
	allowance := &feegrant.BasicAllowance{
		SpendLimit: sdk.NewCoins(sdk.NewInt64Coin("uxion", 1000)),
	}
	allowanceAny, err := codectypes.NewAnyWithValue(allowance)
	require.NoError(t, err)
	return feegrant.Grant{Granter: granter, Grantee: grantee, Allowance: allowanceAny}
}

func TestAuthzHandlerHandleUpdate_Errors(t *testing.T) {
	_, cdc, _ := setupTest(t)
	ctx := context.Background()
	store := newFaultyStore()

	handler, err := NewAuthzHandler(store, cdc)
	require.NoError(t, err)

	granter := sdk.AccAddress([]byte("granter_address_____"))
	grantee := sdk.AccAddress([]byte("grantee_address_____"))
	key := createGrantStoreKey(granter, grantee, "/cosmos.bank.v1beta1.MsgSend")

	t.Run("has failure on delete is returned", func(t *testing.T) {
		store.failHas = true
		defer func() { store.failHas = false }()

		err := handler.HandleUpdate(ctx, &storetypes.StoreKVPair{Key: key, Delete: true})
		require.ErrorIs(t, err, errStoreFailure)
	})

	t.Run("undecodable value is returned", func(t *testing.T) {
		err := handler.HandleUpdate(ctx, &storetypes.StoreKVPair{Key: key, Value: []byte("invalid protobuf data")})
		require.Error(t, err)

		has, err := handler.Authorizations.Has(ctx, collections.Join3(granter, grantee, "/cosmos.bank.v1beta1.MsgSend"))
		require.NoError(t, err)
		require.False(t, has)
	})
}

func TestFeeGrantHandlerHandleUpdate_Errors(t *testing.T) {
	_, cdc, _ := setupTest(t)
	ctx := context.Background()
	store := newFaultyStore()

	handler, err := NewFeeGrantHandler(store, cdc)
	require.NoError(t, err)

	granter := sdk.AccAddress([]byte("granter_address_____"))
	grantee := sdk.AccAddress([]byte("grantee_address_____"))
	key := feegrantv2.FeeAllowanceKey(granter, grantee)

	t.Run("has failure on delete is returned", func(t *testing.T) {
		store.failHas = true
		defer func() { store.failHas = false }()

		err := handler.HandleUpdate(ctx, &storetypes.StoreKVPair{Key: key, Delete: true})
		require.ErrorIs(t, err, errStoreFailure)
	})

	t.Run("undecodable value is returned", func(t *testing.T) {
		err := handler.HandleUpdate(ctx, &storetypes.StoreKVPair{Key: key, Value: []byte("invalid protobuf data")})
		require.Error(t, err)

		_, err = handler.GetGrant(ctx, granter, grantee)
		require.Error(t, err)
	})
}

func TestSafeAuthzHandlerUpdate_StoreFailures(t *testing.T) {
	_, cdc, _ := setupTest(t)
	ctx := context.Background()
	logger := log.NewTestLogger(t)
	store := newFaultyStore()

	handler, err := NewAuthzHandler(store, cdc)
	require.NoError(t, err)

	granter := sdk.AccAddress([]byte("granter_address_____"))
	grantee := sdk.AccAddress([]byte("grantee_address_____"))
	grant, msgType := newSendGrant(t)
	key := createGrantStoreKey(granter, grantee, msgType)
	grantBz, err := cdc.Marshal(&grant)
	require.NoError(t, err)

	t.Run("has failure on delete is swallowed", func(t *testing.T) {
		store.failHas = true
		defer func() { store.failHas = false }()

		err := SafeAuthzHandlerUpdate(ctx, handler, &storetypes.StoreKVPair{Key: key, Delete: true}, logger)
		require.NoError(t, err)
	})

	t.Run("remove failure on delete is swallowed", func(t *testing.T) {
		require.NoError(t, handler.SetGrant(ctx, granter, grantee, msgType, grant))
		store.failDelete = true
		defer func() { store.failDelete = false }()

		err := SafeAuthzHandlerUpdate(ctx, handler, &storetypes.StoreKVPair{Key: key, Delete: true}, logger)
		require.NoError(t, err)

		// the grant survived the failed removal
		_, err = handler.GetGrant(ctx, granter, grantee, msgType)
		require.NoError(t, err)
	})

	t.Run("set failure on create is swallowed", func(t *testing.T) {
		require.NoError(t, handler.Authorizations.Remove(ctx, collections.Join3(granter, grantee, msgType)))
		store.failSet = true
		defer func() { store.failSet = false }()

		err := SafeAuthzHandlerUpdate(ctx, handler, &storetypes.StoreKVPair{Key: key, Value: grantBz}, logger)
		require.NoError(t, err)

		has, err := handler.Authorizations.Has(ctx, collections.Join3(granter, grantee, msgType))
		require.NoError(t, err)
		require.False(t, has)
	})
}

func TestSafeFeeGrantHandlerUpdate_StoreFailures(t *testing.T) {
	_, cdc, addrCodec := setupTest(t)
	ctx := context.Background()
	logger := log.NewTestLogger(t)
	store := newFaultyStore()

	handler, err := NewFeeGrantHandler(store, cdc)
	require.NoError(t, err)

	granter := sdk.AccAddress([]byte("granter_address_____"))
	grantee := sdk.AccAddress([]byte("grantee_address_____"))
	granterStr, err := addrCodec.BytesToString(granter)
	require.NoError(t, err)
	granteeStr, err := addrCodec.BytesToString(grantee)
	require.NoError(t, err)
	grant := newBasicAllowance(t, granterStr, granteeStr)
	key := feegrantv2.FeeAllowanceKey(granter, grantee)
	grantBz, err := cdc.Marshal(&grant)
	require.NoError(t, err)

	t.Run("has failure on delete is swallowed", func(t *testing.T) {
		store.failHas = true
		defer func() { store.failHas = false }()

		err := SafeFeeGrantHandlerUpdate(ctx, handler, &storetypes.StoreKVPair{Key: key, Delete: true}, logger)
		require.NoError(t, err)
	})

	t.Run("remove failure on delete is swallowed", func(t *testing.T) {
		require.NoError(t, handler.SetGrant(ctx, granter, grantee, grant))
		store.failDelete = true
		defer func() { store.failDelete = false }()

		err := SafeFeeGrantHandlerUpdate(ctx, handler, &storetypes.StoreKVPair{Key: key, Delete: true}, logger)
		require.NoError(t, err)

		_, err = handler.GetGrant(ctx, granter, grantee)
		require.NoError(t, err)
	})

	t.Run("set failure on create is swallowed", func(t *testing.T) {
		require.NoError(t, handler.FeeAllowances.Remove(ctx, collections.Join(granter, grantee)))
		store.failSet = true
		defer func() { store.failSet = false }()

		err := SafeFeeGrantHandlerUpdate(ctx, handler, &storetypes.StoreKVPair{Key: key, Value: grantBz}, logger)
		require.NoError(t, err)

		_, err = handler.GetGrant(ctx, granter, grantee)
		require.Error(t, err)
	})

	t.Run("key with out of range address length is skipped", func(t *testing.T) {
		// long enough to pass the length check, but the grantee length byte
		// points past the end of the key so address parsing panics
		key := []byte{0x00, 0xff, 0x01, 0x02, 0x03}

		err := SafeFeeGrantHandlerUpdate(ctx, handler, &storetypes.StoreKVPair{Key: key, Value: grantBz}, logger)
		require.NoError(t, err)
	})
}

func TestAuthzQuerier_StoreFailures(t *testing.T) {
	_, cdc, addrCodec := setupTest(t)
	ctx := context.Background()
	store := newFaultyStore()

	handler, err := NewAuthzHandler(store, cdc)
	require.NoError(t, err)
	querier := NewAuthzQuerier(handler, cdc, addrCodec)

	granter := sdk.AccAddress([]byte("granter_address_____"))
	grantee := sdk.AccAddress([]byte("grantee_address_____"))
	grant, msgType := newSendGrant(t)
	require.NoError(t, handler.SetGrant(ctx, granter, grantee, msgType, grant))

	granteeStr, err := addrCodec.BytesToString(grantee)
	require.NoError(t, err)
	rawKey := rawGranteeIndexKey(t, handler, grantee, granter, msgType)

	t.Run("index iteration failure", func(t *testing.T) {
		store.failIterator = true
		defer func() { store.failIterator = false }()

		_, err := querier.GranteeGrants(ctx, &indexerauthz.QueryGranteeGrantsRequest{Grantee: granteeStr})
		require.ErrorIs(t, err, errStoreFailure)
	})

	t.Run("grant lookup failure", func(t *testing.T) {
		store.failGet = true
		defer func() { store.failGet = false }()

		_, err := querier.GranteeGrants(ctx, &indexerauthz.QueryGranteeGrantsRequest{Grantee: granteeStr})
		require.ErrorIs(t, err, errStoreFailure)
	})

	t.Run("raw index iteration failure", func(t *testing.T) {
		store.failIterator = true
		defer func() { store.failIterator = false }()

		_, err := querier.GranteeGrants(ctx, &indexerauthz.QueryGranteeGrantsRequest{
			Grantee:    granteeStr,
			Pagination: &query.PageRequest{Key: rawKey},
		})
		require.ErrorIs(t, err, errStoreFailure)
	})

	t.Run("raw grant lookup failure", func(t *testing.T) {
		store.failGet = true
		defer func() { store.failGet = false }()

		_, err := querier.GranteeGrants(ctx, &indexerauthz.QueryGranteeGrantsRequest{
			Grantee:    granteeStr,
			Pagination: &query.PageRequest{Key: rawKey},
		})
		require.ErrorIs(t, err, errStoreFailure)
	})
}

func TestFeegrantQuerier_StoreFailures(t *testing.T) {
	_, cdc, addrCodec := setupTest(t)
	ctx := context.Background()
	store := newFaultyStore()

	handler, err := NewFeeGrantHandler(store, cdc)
	require.NoError(t, err)
	querier := NewFeegrantQuerier(handler, cdc, addrCodec)

	granter := sdk.AccAddress([]byte("granter_address_____"))
	grantee := sdk.AccAddress([]byte("grantee_address_____"))
	granterStr, err := addrCodec.BytesToString(granter)
	require.NoError(t, err)
	granteeStr, err := addrCodec.BytesToString(grantee)
	require.NoError(t, err)
	require.NoError(t, handler.SetGrant(ctx, granter, grantee, newBasicAllowance(t, granterStr, granteeStr)))

	store.failGet = true
	defer func() { store.failGet = false }()

	t.Run("allowance lookup failure", func(t *testing.T) {
		_, err := querier.Allowance(ctx, &indexerfeegrant.QueryAllowanceRequest{Granter: granterStr, Grantee: granteeStr})
		require.ErrorIs(t, err, errStoreFailure)
	})

	t.Run("allowances lookup failure behind the grantee index", func(t *testing.T) {
		_, err := querier.Allowances(ctx, &indexerfeegrant.QueryAllowancesRequest{Grantee: granteeStr})
		require.ErrorIs(t, err, errStoreFailure)
	})
}
