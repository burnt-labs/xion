package indexer

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	db "github.com/cosmos/cosmos-db"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/cosmos/cosmos-sdk/x/authz"

	indexerauthz "github.com/burnt-labs/xion/indexer/authz"
	indexerfeegrant "github.com/burnt-labs/xion/indexer/feegrant"
)

var errCodecFailure = errors.New("address codec failure")

// bytesToStringFailCodec parses addresses normally but fails the n-th
// BytesToString call, so result transformation errors can be exercised
// independently of request parsing.
type bytesToStringFailCodec struct {
	address.Codec
	failOn int
	calls  int
}

func (c *bytesToStringFailCodec) BytesToString(bz []byte) (string, error) {
	c.calls++
	if c.calls == c.failOn {
		return "", errCodecFailure
	}
	return c.Codec.BytesToString(bz)
}

// rawGranteeIndexKey returns the encoded grantee-index entry for a grant,
// without the collection prefix, which is the form MultiIterateRaw expects as
// a start position.
func rawGranteeIndexKey(t *testing.T, handler *AuthzHandler, grantee, granter sdk.AccAddress, msgType string) []byte {
	t.Helper()
	key := collections.Join(grantee, collections.Join3(granter, grantee, msgType))
	bz, err := collections.EncodeKeyWithPrefix(nil, handler.Authorizations.Indexes.Grantee.KeyCodec(), key)
	require.NoError(t, err)
	return bz
}

// writeCorruptGranteeIndexEntry stores an index entry under the grantee's
// prefix whose primary key part cannot be decoded.
func writeCorruptGranteeIndexEntry(t *testing.T, memDB db.DB, grantee sdk.AccAddress) {
	t.Helper()
	buf := make([]byte, sdk.AccAddressKey.SizeNonTerminal(grantee))
	n, err := sdk.AccAddressKey.EncodeNonTerminal(buf, grantee)
	require.NoError(t, err)

	key := append([]byte{}, AuthzGranteeIndexPrefix...)
	key = append(key, buf[:n]...)
	key = append(key, 0xff) // address length byte with no address bytes behind it
	require.NoError(t, memDB.Set(key, []byte{}))
}

type granteeQuerierFixture struct {
	memDB      db.DB
	handler    *AuthzHandler
	querier    indexerauthz.QueryServer
	addrCodec  address.Codec
	grantee    sdk.AccAddress
	granteeStr string
	grant      authz.Grant
	msgType    string
}

func newGranteeQuerierFixture(t *testing.T) *granteeQuerierFixture {
	t.Helper()
	memDB, cdc, addrCodec := setupTest(t)
	handler, err := NewAuthzHandler(&kvAccessor{db: memDB}, cdc)
	require.NoError(t, err)

	grantee := sdk.AccAddress([]byte("edge_grantee________"))
	granteeStr, err := addrCodec.BytesToString(grantee)
	require.NoError(t, err)
	grant, msgType := newSendGrant(t)

	return &granteeQuerierFixture{
		memDB:      memDB,
		handler:    handler,
		querier:    NewAuthzQuerier(handler, cdc, addrCodec),
		addrCodec:  addrCodec,
		grantee:    grantee,
		granteeStr: granteeStr,
		grant:      grant,
		msgType:    msgType,
	}
}

func (f *granteeQuerierFixture) granter(i int) sdk.AccAddress {
	return sdk.AccAddress([]byte(fmt.Sprintf("edge_granter_%02d_____", i)))
}

func (f *granteeQuerierFixture) setGrants(t *testing.T, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		require.NoError(t, f.handler.SetGrant(context.Background(), f.granter(i), f.grantee, f.msgType, f.grant))
	}
}

func (f *granteeQuerierFixture) query(pagination *query.PageRequest) (*indexerauthz.QueryGranteeGrantsResponse, error) {
	return f.querier.GranteeGrants(context.Background(), &indexerauthz.QueryGranteeGrantsRequest{
		Grantee:    f.granteeStr,
		Pagination: pagination,
	})
}

func TestPaginateRejectsOffsetWithKey(t *testing.T) {
	memDB, cdc, addrCodec := setupTest(t)
	ctx := context.Background()
	badPage := &query.PageRequest{Offset: 1, Key: []byte{0x01}}

	granter := sdk.AccAddress([]byte("granter_address_____"))
	grantee := sdk.AccAddress([]byte("grantee_address_____"))
	granterStr, err := addrCodec.BytesToString(granter)
	require.NoError(t, err)
	granteeStr, err := addrCodec.BytesToString(grantee)
	require.NoError(t, err)

	authzHandler, err := NewAuthzHandler(&kvAccessor{db: memDB}, cdc)
	require.NoError(t, err)
	authzQuerier := NewAuthzQuerier(authzHandler, cdc, addrCodec)

	feegrantHandler, err := NewFeeGrantHandler(&kvAccessor{db: memDB}, cdc)
	require.NoError(t, err)
	feegrantQuerier := NewFeegrantQuerier(feegrantHandler, cdc, addrCodec)

	t.Run("Grants", func(t *testing.T) {
		_, err := authzQuerier.Grants(ctx, &indexerauthz.QueryGrantsRequest{Granter: granterStr, Pagination: badPage})
		require.ErrorContains(t, err, "either offset or key is expected")
	})

	t.Run("GranterGrants", func(t *testing.T) {
		_, err := authzQuerier.GranterGrants(ctx, &indexerauthz.QueryGranterGrantsRequest{Granter: granterStr, Pagination: badPage})
		require.ErrorContains(t, err, "either offset or key is expected")
	})

	t.Run("Allowances", func(t *testing.T) {
		_, err := feegrantQuerier.Allowances(ctx, &indexerfeegrant.QueryAllowancesRequest{Grantee: granteeStr, Pagination: badPage})
		require.ErrorContains(t, err, "either offset or key is expected")
	})

	t.Run("AllowancesByGranter", func(t *testing.T) {
		_, err := feegrantQuerier.AllowancesByGranter(ctx, &indexerfeegrant.QueryAllowancesByGranterRequest{Granter: granterStr, Pagination: badPage})
		require.ErrorContains(t, err, "either offset or key is expected")
	})
}

func TestGranterGrantsTransformErrors(t *testing.T) {
	ctx := context.Background()

	granter := sdk.AccAddress([]byte("granter_address_____"))
	grantee := sdk.AccAddress([]byte("grantee_address_____"))

	t.Run("grant without authorization", func(t *testing.T) {
		memDB, cdc, addrCodec := setupTest(t)
		handler, err := NewAuthzHandler(&kvAccessor{db: memDB}, cdc)
		require.NoError(t, err)
		querier := NewAuthzQuerier(handler, cdc, addrCodec)

		expiration := time.Now().Add(time.Hour)
		require.NoError(t, handler.SetGrant(ctx, granter, grantee, "/cosmos.bank.v1beta1.MsgSend", authz.Grant{Expiration: &expiration}))

		granterStr, err := addrCodec.BytesToString(granter)
		require.NoError(t, err)
		_, err = querier.GranterGrants(ctx, &indexerauthz.QueryGranterGrantsRequest{Granter: granterStr})
		require.Error(t, err)
	})

	for _, tc := range []struct {
		name   string
		failOn int
	}{
		{name: "granter address conversion fails", failOn: 1},
		{name: "grantee address conversion fails", failOn: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			memDB, cdc, addrCodec := setupTest(t)
			handler, err := NewAuthzHandler(&kvAccessor{db: memDB}, cdc)
			require.NoError(t, err)
			grant, msgType := newSendGrant(t)
			require.NoError(t, handler.SetGrant(ctx, granter, grantee, msgType, grant))

			granterStr, err := addrCodec.BytesToString(granter)
			require.NoError(t, err)
			querier := NewAuthzQuerier(handler, cdc, &bytesToStringFailCodec{Codec: addrCodec, failOn: tc.failOn})

			_, err = querier.GranterGrants(ctx, &indexerauthz.QueryGranterGrantsRequest{Granter: granterStr})
			require.ErrorIs(t, err, errCodecFailure)
		})
	}
}

func TestGranteeGrantsTransformErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("grant without authorization", func(t *testing.T) {
		f := newGranteeQuerierFixture(t)
		expiration := time.Now().Add(time.Hour)
		require.NoError(t, f.handler.SetGrant(ctx, f.granter(0), f.grantee, f.msgType, authz.Grant{Expiration: &expiration}))

		_, err := f.query(nil)
		require.Error(t, err)

		_, err = f.query(&query.PageRequest{Key: rawGranteeIndexKey(t, f.handler, f.grantee, f.granter(0), f.msgType)})
		require.Error(t, err)
	})

	t.Run("granter address conversion fails", func(t *testing.T) {
		f := newGranteeQuerierFixture(t)
		f.setGrants(t, 1)
		_, cdc, _ := setupTest(t)
		newQuerier := func() indexerauthz.QueryServer {
			return NewAuthzQuerier(f.handler, cdc, &bytesToStringFailCodec{Codec: f.addrCodec, failOn: 1})
		}

		_, err := newQuerier().GranteeGrants(ctx, &indexerauthz.QueryGranteeGrantsRequest{Grantee: f.granteeStr})
		require.ErrorIs(t, err, errCodecFailure)

		_, err = newQuerier().GranteeGrants(ctx, &indexerauthz.QueryGranteeGrantsRequest{
			Grantee:    f.granteeStr,
			Pagination: &query.PageRequest{Key: rawGranteeIndexKey(t, f.handler, f.grantee, f.granter(0), f.msgType)},
		})
		require.ErrorIs(t, err, errCodecFailure)
	})
}

func TestGranteeGrantsCorruptIndexEntries(t *testing.T) {
	t.Run("standard iteration fails to decode the primary key", func(t *testing.T) {
		f := newGranteeQuerierFixture(t)
		f.setGrants(t, 1)
		writeCorruptGranteeIndexEntry(t, f.memDB, f.grantee)

		_, err := f.query(&query.PageRequest{Limit: 10})
		require.ErrorIs(t, err, collections.ErrEncoding)
	})

	t.Run("standard iteration fails to build the next key", func(t *testing.T) {
		f := newGranteeQuerierFixture(t)
		f.setGrants(t, 1)
		writeCorruptGranteeIndexEntry(t, f.memDB, f.grantee)

		_, err := f.query(&query.PageRequest{Limit: 1})
		require.ErrorIs(t, err, collections.ErrEncoding)
	})

	t.Run("raw iteration fails to decode the key", func(t *testing.T) {
		f := newGranteeQuerierFixture(t)
		f.setGrants(t, 1)
		writeCorruptGranteeIndexEntry(t, f.memDB, f.grantee)
		start := rawGranteeIndexKey(t, f.handler, f.grantee, f.granter(0), f.msgType)

		_, err := f.query(&query.PageRequest{Key: start, Limit: 10})
		require.ErrorIs(t, err, collections.ErrEncoding)
	})

	t.Run("raw iteration fails to build the next key", func(t *testing.T) {
		f := newGranteeQuerierFixture(t)
		f.setGrants(t, 1)
		writeCorruptGranteeIndexEntry(t, f.memDB, f.grantee)
		start := rawGranteeIndexKey(t, f.handler, f.grantee, f.granter(0), f.msgType)

		_, err := f.query(&query.PageRequest{Key: start, Limit: 1})
		require.ErrorIs(t, err, collections.ErrEncoding)
	})

	t.Run("raw iteration fails while counting the total", func(t *testing.T) {
		f := newGranteeQuerierFixture(t)
		f.setGrants(t, 2)
		writeCorruptGranteeIndexEntry(t, f.memDB, f.grantee)
		start := rawGranteeIndexKey(t, f.handler, f.grantee, f.granter(0), f.msgType)

		_, err := f.query(&query.PageRequest{Key: start, Limit: 1, CountTotal: true})
		require.ErrorIs(t, err, collections.ErrEncoding)
	})
}

func TestGranteeGrantsNextKeyEncodingErrors(t *testing.T) {
	// A message type containing the string delimiter can be stored (the
	// terminal encoding allows it) but cannot be re-encoded as a non-terminal
	// pagination key.
	const delimitedMsgType = "/cosmos.bank.v1beta1.Msg\x00Send"

	t.Run("standard iteration", func(t *testing.T) {
		f := newGranteeQuerierFixture(t)
		f.setGrants(t, 1)
		require.NoError(t, f.handler.SetGrant(context.Background(), f.granter(1), f.grantee, delimitedMsgType, f.grant))

		_, err := f.query(&query.PageRequest{Limit: 1})
		require.ErrorIs(t, err, collections.ErrEncoding)
	})

	t.Run("raw iteration", func(t *testing.T) {
		f := newGranteeQuerierFixture(t)
		f.setGrants(t, 1)
		require.NoError(t, f.handler.SetGrant(context.Background(), f.granter(1), f.grantee, delimitedMsgType, f.grant))
		start := rawGranteeIndexKey(t, f.handler, f.grantee, f.granter(0), f.msgType)

		_, err := f.query(&query.PageRequest{Key: start, Limit: 1})
		require.ErrorIs(t, err, collections.ErrEncoding)
	})
}

func TestGranteeGrantsRawIterationContinuation(t *testing.T) {
	f := newGranteeQuerierFixture(t)
	ctx := context.Background()
	f.setGrants(t, 6)

	// a grant for a grantee that sorts after ours is reached by the raw
	// iteration (it has no end bound) and must be filtered out everywhere
	otherGrantee := sdk.AccAddress([]byte("edge_grantee_other__"))
	require.NoError(t, f.handler.SetGrant(ctx, f.granter(0), otherGrantee, f.msgType, f.grant))

	page1, err := f.query(&query.PageRequest{Limit: 2})
	require.NoError(t, err)
	require.Len(t, page1.Grants, 2)
	require.NotEmpty(t, page1.Pagination.NextKey)

	page2, err := f.query(&query.PageRequest{Key: page1.Pagination.NextKey, Limit: 2, CountTotal: true})
	require.NoError(t, err)
	require.Len(t, page2.Grants, 2)
	require.NotEmpty(t, page2.Pagination.NextKey, "more grants remain after the second page")
	// the total covers this page and whatever is left for this grantee; the
	// other grantee's entry, which the raw iteration also visits, is not counted
	require.GreaterOrEqual(t, page2.Pagination.Total, uint64(len(page2.Grants)))
	require.LessOrEqual(t, page2.Pagination.Total, uint64(4))

	seen := map[string]struct{}{}
	for _, g := range append(page1.Grants, page2.Grants...) {
		require.Equal(t, f.granteeStr, g.Grantee)
		_, dup := seen[g.Granter]
		require.False(t, dup, "pages must not overlap")
		seen[g.Granter] = struct{}{}
	}

	page3, err := f.query(&query.PageRequest{Key: page2.Pagination.NextKey, Limit: 10})
	require.NoError(t, err)
	require.Empty(t, page3.Pagination.NextKey)
	for _, g := range page3.Grants {
		require.Equal(t, f.granteeStr, g.Grantee)
		_, dup := seen[g.Granter]
		require.False(t, dup, "pages must not overlap")
	}
}

func TestGranteeGrantsRawIterationFromExactPosition(t *testing.T) {
	f := newGranteeQuerierFixture(t)
	f.setGrants(t, 3)

	granter1Str, err := f.addrCodec.BytesToString(f.granter(1))
	require.NoError(t, err)

	start := rawGranteeIndexKey(t, f.handler, f.grantee, f.granter(1), f.msgType)
	resp, err := f.query(&query.PageRequest{Key: start, CountTotal: true})
	require.NoError(t, err)
	require.Len(t, resp.Grants, 2)
	require.Equal(t, granter1Str, resp.Grants[0].Granter)
	require.Empty(t, resp.Pagination.NextKey)
	require.Equal(t, uint64(2), resp.Pagination.Total)
}
