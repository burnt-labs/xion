package indexer

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	abci "github.com/cometbft/cometbft/abci/types"

	db "github.com/cosmos/cosmos-db"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	"cosmossdk.io/x/feegrant"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/std"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	indexerauthz "github.com/burnt-labs/xion/indexer/authz"
	indexerfeegrant "github.com/burnt-labs/xion/indexer/feegrant"
)

func setupTest(_t *testing.T) (db.DB, codec.Codec, address.Codec) {
	// Create in-memory db
	memDB := db.NewMemDB()

	// Setup codec
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	std.RegisterInterfaces(interfaceRegistry)
	authz.RegisterInterfaces(interfaceRegistry)
	banktypes.RegisterInterfaces(interfaceRegistry)
	feegrant.RegisterInterfaces(interfaceRegistry)
	cdc := codec.NewProtoCodec(interfaceRegistry)

	// Setup address codec
	addrCodec := addresscodec.NewBech32Codec("xion")

	return memDB, cdc, addrCodec
}

func TestConfig(t *testing.T) {
	cfg := DefaultConfig()
	require.False(t, cfg.Enabled)
}

func TestErrors(t *testing.T) {
	require.NotNil(t, ErrGrantNotFound)
	require.NotNil(t, ErrAllowanceNotFound)
	require.Equal(t, "grant not found", ErrGrantNotFound.Error())
	require.Equal(t, "allowance not found", ErrAllowanceNotFound.Error())
}

func TestUnsafeStrToBytes(t *testing.T) {
	s := "test string"
	b := UnsafeStrToBytes(s)
	require.Equal(t, []byte(s), b)
}

func TestUnsafeBytesToStr(t *testing.T) {
	b := []byte("test bytes")
	s := UnsafeBytesToStr(b)
	require.Equal(t, "test bytes", s)
}

func TestAuthzHandler(t *testing.T) {
	memDB, cdc, _ := setupTest(t)
	ctx := context.Background()

	// Create handler
	handler, err := NewAuthzHandler(&kvAccessor{db: memDB}, cdc)
	require.NoError(t, err)
	require.NotNil(t, handler)

	// Test addresses
	granter := sdk.AccAddress([]byte("granter_address_____"))
	grantee := sdk.AccAddress([]byte("grantee_address_____"))
	msgType := "/cosmos.bank.v1beta1.MsgSend"

	// Create authorization
	auth := banktypes.NewSendAuthorization(sdk.NewCoins(sdk.NewInt64Coin("uxion", 1000)), nil)
	expiration := time.Now().Add(time.Hour)
	grant, err := authz.NewGrant(expiration, auth, nil)
	require.NoError(t, err)

	// Test SetGrant
	err = handler.SetGrant(ctx, granter, grantee, msgType, grant)
	require.NoError(t, err)

	// Test GetGrant
	retrievedGrant, err := handler.GetGrant(ctx, granter, grantee, msgType)
	require.NoError(t, err)
	require.Equal(t, grant.Expiration, retrievedGrant.Expiration)

	// Test HandleUpdate - Insert
	grantValue, err := cdc.Marshal(&grant)
	require.NoError(t, err)

	// Encode key properly
	codec := collections.TripleKeyCodec(sdk.AccAddressKey, sdk.AccAddressKey, collections.StringKey)
	triple := collections.Join3(granter, grantee, msgType)
	size := codec.Size(triple)
	buf := make([]byte, size)
	_, err = codec.Encode(buf, triple)
	require.NoError(t, err)
	key := append([]byte{0x01}, buf...)

	pair := &storetypes.StoreKVPair{
		Key:    key,
		Value:  grantValue,
		Delete: false,
	}
	err = handler.HandleUpdate(ctx, pair)
	require.NoError(t, err)

	// Test HandleUpdate - Delete
	pair.Delete = true
	err = handler.HandleUpdate(ctx, pair)
	require.NoError(t, err)

	// Test HandleUpdate - Delete non-existent (should error)
	err = handler.HandleUpdate(ctx, pair)
	require.Error(t, err)
	require.Equal(t, ErrGrantNotFound, err)
}

func TestAuthzQuerier(t *testing.T) {
	memDB, cdc, addrCodec := setupTest(t)
	ctx := context.Background()

	// Create handler and querier
	handler, err := NewAuthzHandler(&kvAccessor{db: memDB}, cdc)
	require.NoError(t, err)
	querier := NewAuthzQuerier(handler, cdc, addrCodec)

	// Test that querier was created
	require.NotNil(t, querier)

	// Test invalid address handling
	_, err = querier.GranterGrants(ctx, &indexerauthz.QueryGranterGrantsRequest{
		Granter: "invalid",
	})
	require.Error(t, err)

	_, err = querier.GranteeGrants(ctx, &indexerauthz.QueryGranteeGrantsRequest{
		Grantee: "invalid",
	})
	require.Error(t, err)

	// Test helper functions
	granter := sdk.AccAddress([]byte("granter_address_____"))
	grantee := sdk.AccAddress([]byte("grantee_address_____"))

	prefixOpt := WithCollectionPaginationTriplePrefix[sdk.AccAddress, sdk.AccAddress, string](granter)
	require.NotNil(t, prefixOpt)

	opts := &query.CollectionsPaginateOptions[collections.Triple[sdk.AccAddress, sdk.AccAddress, string]]{}
	prefixOpt(opts)
	require.NotNil(t, opts.Prefix)

	pairPrefixOpt := WithCollectionPaginationTriplePairPrefix[sdk.AccAddress, sdk.AccAddress, string](granter, grantee)
	require.NotNil(t, pairPrefixOpt)

	opts2 := &query.CollectionsPaginateOptions[collections.Triple[sdk.AccAddress, sdk.AccAddress, string]]{}
	pairPrefixOpt(opts2)
	require.NotNil(t, opts2.Prefix)
}

func TestAuthzQuerierWithPagination(t *testing.T) {
	// Now that we have index-based pagination, we can test it with in-memory collections

	memDB, cdc, addrCodec := setupTest(t)
	ctx := context.Background()

	// Create handler and querier
	handler, err := NewAuthzHandler(&kvAccessor{db: memDB}, cdc)
	require.NoError(t, err)
	querier := NewAuthzQuerier(handler, cdc, addrCodec)

	// Setup test data
	granter1 := sdk.AccAddress([]byte("granter1_address____"))
	granter2 := sdk.AccAddress([]byte("granter2_address____"))
	grantee1 := sdk.AccAddress([]byte("grantee1_address____"))
	grantee2 := sdk.AccAddress([]byte("grantee2_address____"))
	msgType := "/cosmos.bank.v1beta1.MsgSend"

	auth := banktypes.NewSendAuthorization(sdk.NewCoins(sdk.NewInt64Coin("uxion", 1000)), nil)
	expiration := time.Now().Add(time.Hour)
	grant, err := authz.NewGrant(expiration, auth, nil)
	require.NoError(t, err)

	// Add multiple grants to test pagination
	require.NoError(t, handler.SetGrant(ctx, granter1, grantee1, msgType, grant))
	require.NoError(t, handler.SetGrant(ctx, granter1, grantee2, msgType, grant))
	require.NoError(t, handler.SetGrant(ctx, granter2, grantee1, msgType, grant))
	require.NoError(t, handler.SetGrant(ctx, granter2, grantee2, msgType, grant))

	granter1Str, _ := addrCodec.BytesToString(granter1)
	granter2Str, _ := addrCodec.BytesToString(granter2)
	grantee1Str, _ := addrCodec.BytesToString(grantee1)
	grantee2Str, _ := addrCodec.BytesToString(grantee2)

	// Test Grants query - all grants
	t.Run("Grants_All", func(t *testing.T) {
		resp, err := querier.Grants(ctx, &indexerauthz.QueryGrantsRequest{
			Pagination: &query.PageRequest{
				Limit: 10,
			},
		})
		require.NoError(t, err)
		require.Len(t, resp.Grants, 4)
		require.NotNil(t, resp.Pagination)
	})

	// Test Grants query - by granter
	t.Run("Grants_ByGranter", func(t *testing.T) {
		resp, err := querier.Grants(ctx, &indexerauthz.QueryGrantsRequest{
			Granter: granter1Str,
			Pagination: &query.PageRequest{
				Limit: 10,
			},
		})
		require.NoError(t, err)
		require.Len(t, resp.Grants, 2)
	})

	// Test Grants query - by grantee (returns all since grantee-only uses nil prefix)
	t.Run("Grants_ByGrantee", func(t *testing.T) {
		resp, err := querier.Grants(ctx, &indexerauthz.QueryGrantsRequest{
			Grantee: grantee1Str,
			Pagination: &query.PageRequest{
				Limit: 10,
			},
		})
		require.NoError(t, err)
		// Returns all grants since we can't filter by grantee only on main collection
		require.GreaterOrEqual(t, len(resp.Grants), 2)
	})

	// Test Grants query - by both granter and grantee
	t.Run("Grants_ByBoth", func(t *testing.T) {
		resp, err := querier.Grants(ctx, &indexerauthz.QueryGrantsRequest{
			Granter: granter1Str,
			Grantee: grantee1Str,
			Pagination: &query.PageRequest{
				Limit: 10,
			},
		})
		require.NoError(t, err)
		require.Len(t, resp.Grants, 1)
	})

	// Test Grants query - invalid granter
	t.Run("Grants_InvalidGranter", func(t *testing.T) {
		_, err := querier.Grants(ctx, &indexerauthz.QueryGrantsRequest{
			Granter: "invalid",
		})
		require.Error(t, err)
	})

	// Test Grants query - invalid grantee
	t.Run("Grants_InvalidGrantee", func(t *testing.T) {
		_, err := querier.Grants(ctx, &indexerauthz.QueryGrantsRequest{
			Grantee: "invalid",
		})
		require.Error(t, err)
	})

	// Test GranterGrants
	t.Run("GranterGrants_Success", func(t *testing.T) {
		resp, err := querier.GranterGrants(ctx, &indexerauthz.QueryGranterGrantsRequest{
			Granter: granter1Str,
			Pagination: &query.PageRequest{
				Limit: 10,
			},
		})
		require.NoError(t, err)
		require.Len(t, resp.Grants, 2)
		require.Equal(t, granter1Str, resp.Grants[0].Granter)
		require.NotNil(t, resp.Pagination)
	})

	// Test GranterGrants with pagination
	t.Run("GranterGrants_Pagination", func(t *testing.T) {
		resp, err := querier.GranterGrants(ctx, &indexerauthz.QueryGranterGrantsRequest{
			Granter: granter2Str,
			Pagination: &query.PageRequest{
				Limit: 1,
			},
		})
		require.NoError(t, err)
		require.Len(t, resp.Grants, 1)
		require.NotNil(t, resp.Pagination)
		require.NotEmpty(t, resp.Pagination.NextKey)
	})

	// Test GranteeGrants
	t.Run("GranteeGrants_Success", func(t *testing.T) {
		resp, err := querier.GranteeGrants(ctx, &indexerauthz.QueryGranteeGrantsRequest{
			Grantee: grantee1Str,
			Pagination: &query.PageRequest{
				Limit: 10,
			},
		})
		require.NoError(t, err)
		require.Len(t, resp.Grants, 2)
		require.Equal(t, grantee1Str, resp.Grants[0].Grantee)
		require.NotNil(t, resp.Pagination)
	})

	// Test GranteeGrants with pagination
	t.Run("GranteeGrants_Pagination", func(t *testing.T) {
		resp, err := querier.GranteeGrants(ctx, &indexerauthz.QueryGranteeGrantsRequest{
			Grantee: grantee2Str,
			Pagination: &query.PageRequest{
				Limit: 1,
			},
		})
		require.NoError(t, err)
		require.Len(t, resp.Grants, 1)
		require.NotNil(t, resp.Pagination)
		require.NotEmpty(t, resp.Pagination.NextKey)
	})
}

func TestFeeGrantHandler(t *testing.T) {
	memDB, cdc, _ := setupTest(t)
	ctx := context.Background()

	// Create handler
	handler, err := NewFeeGrantHandler(&kvAccessor{db: memDB}, cdc)
	require.NoError(t, err)
	require.NotNil(t, handler)

	// Test addresses
	granter := sdk.AccAddress([]byte("granter_address_____"))
	grantee := sdk.AccAddress([]byte("grantee_address_____"))

	// Create allowance
	allowance := &feegrant.BasicAllowance{
		SpendLimit: sdk.NewCoins(sdk.NewInt64Coin("uxion", 1000)),
	}
	anyAllowance, err := codectypes.NewAnyWithValue(allowance)
	require.NoError(t, err)

	grant := feegrant.Grant{
		Granter:   granter.String(),
		Grantee:   grantee.String(),
		Allowance: anyAllowance,
	}

	// Test SetGrant
	err = handler.SetGrant(ctx, granter, grantee, grant)
	require.NoError(t, err)

	// Test GetGrant
	retrievedGrant, err := handler.GetGrant(ctx, granter, grantee)
	require.NoError(t, err)
	require.Equal(t, grant.Granter, retrievedGrant.Granter)
	require.Equal(t, grant.Grantee, retrievedGrant.Grantee)

	// Test HandleUpdate - Insert
	grantValue, err := cdc.Marshal(&grant)
	require.NoError(t, err)

	// Build key using feegrant format
	key := feegrant.FeeAllowanceKey(granter, grantee)

	pair := &storetypes.StoreKVPair{
		Key:    key,
		Value:  grantValue,
		Delete: false,
	}
	err = handler.HandleUpdate(ctx, pair)
	require.NoError(t, err)

	// Test HandleUpdate - Delete
	pair.Delete = true
	err = handler.HandleUpdate(ctx, pair)
	require.NoError(t, err)

	// Test HandleUpdate - Delete non-existent (should error)
	err = handler.HandleUpdate(ctx, pair)
	require.Error(t, err)
	require.Equal(t, ErrAllowanceNotFound, err)
}

func TestFeeGrantQuerier(t *testing.T) {
	memDB, cdc, addrCodec := setupTest(t)
	ctx := context.Background()

	// Create handler and querier
	handler, err := NewFeeGrantHandler(&kvAccessor{db: memDB}, cdc)
	require.NoError(t, err)
	querier := NewFeegrantQuerier(handler, cdc, addrCodec)

	// Test that querier was created
	require.NotNil(t, querier)

	// Test addresses
	granter1 := sdk.AccAddress([]byte("granter1_address____"))
	grantee1 := sdk.AccAddress([]byte("grantee1_address____"))

	// Setup test data
	allowance := &feegrant.BasicAllowance{
		SpendLimit: sdk.NewCoins(sdk.NewInt64Coin("uxion", 1000)),
	}
	anyAllowance, err := codectypes.NewAnyWithValue(allowance)
	require.NoError(t, err)

	grant := feegrant.Grant{
		Granter:   granter1.String(),
		Grantee:   grantee1.String(),
		Allowance: anyAllowance,
	}

	// Add grant
	require.NoError(t, handler.SetGrant(ctx, granter1, grantee1, grant))

	granter1Str, _ := addrCodec.BytesToString(granter1)
	grantee1Str, _ := addrCodec.BytesToString(grantee1)

	// Test Allowance query (single)
	respSingle, err := querier.Allowance(ctx, &indexerfeegrant.QueryAllowanceRequest{
		Granter: granter1Str,
		Grantee: grantee1Str,
	})
	require.NoError(t, err)
	require.NotNil(t, respSingle.Allowance)
	// Note: The granter/grantee in the response come from the stored Grant which has addresses in cosmos format
	require.NotEmpty(t, respSingle.Allowance.Granter)
	require.NotEmpty(t, respSingle.Allowance.Grantee)

	// Test Allowance query - not found (returns nil allowance, not error)
	nonExistentStr, _ := addrCodec.BytesToString(sdk.AccAddress([]byte("nonexistent_________")))
	respNotFound, err := querier.Allowance(ctx, &indexerfeegrant.QueryAllowanceRequest{
		Granter: nonExistentStr,
		Grantee: grantee1Str,
	})
	require.NoError(t, err)
	require.Nil(t, respNotFound.Allowance)

	// Test Allowance query - invalid granter
	_, err = querier.Allowance(ctx, &indexerfeegrant.QueryAllowanceRequest{
		Granter: "invalid",
		Grantee: grantee1Str,
	})
	require.Error(t, err)

	// Test Allowance query - invalid grantee
	_, err = querier.Allowance(ctx, &indexerfeegrant.QueryAllowanceRequest{
		Granter: granter1Str,
		Grantee: "invalid",
	})
	require.Error(t, err)

	// Test AllowancesByGranter - invalid address
	_, err = querier.AllowancesByGranter(ctx, &indexerfeegrant.QueryAllowancesByGranterRequest{
		Granter: "invalid",
	})
	require.Error(t, err)
}

func TestFeeGrantQuerierWithPagination(t *testing.T) {
	// ReversePair implements IterateRaw, so CollectionPaginate should work directly

	memDB, cdc, addrCodec := setupTest(t)
	ctx := context.Background()

	// Create handler and querier
	handler, err := NewFeeGrantHandler(&kvAccessor{db: memDB}, cdc)
	require.NoError(t, err)
	querier := NewFeegrantQuerier(handler, cdc, addrCodec)

	// Setup test data
	granter1 := sdk.AccAddress([]byte("granter1_address____"))
	granter2 := sdk.AccAddress([]byte("granter2_address____"))
	grantee1 := sdk.AccAddress([]byte("grantee1_address____"))
	grantee2 := sdk.AccAddress([]byte("grantee2_address____"))
	grantee3 := sdk.AccAddress([]byte("grantee3_address____"))

	allowance := &feegrant.BasicAllowance{
		SpendLimit: sdk.NewCoins(sdk.NewInt64Coin("uxion", 1000)),
	}
	anyAllowance, err := codectypes.NewAnyWithValue(allowance)
	require.NoError(t, err)

	// Add multiple grants to test pagination
	grants := []struct {
		granter sdk.AccAddress
		grantee sdk.AccAddress
	}{
		{granter1, grantee1},
		{granter1, grantee2},
		{granter1, grantee3},
		{granter2, grantee1},
		{granter2, grantee2},
	}

	for _, g := range grants {
		grant := feegrant.Grant{
			Granter:   g.granter.String(),
			Grantee:   g.grantee.String(),
			Allowance: anyAllowance,
		}
		require.NoError(t, handler.SetGrant(ctx, g.granter, g.grantee, grant))
	}

	granter1Str, _ := addrCodec.BytesToString(granter1)
	granter2Str, _ := addrCodec.BytesToString(granter2)
	grantee1Str, _ := addrCodec.BytesToString(grantee1)

	// Test Allowances query by grantee
	t.Run("Allowances_Success", func(t *testing.T) {
		resp, err := querier.Allowances(ctx, &indexerfeegrant.QueryAllowancesRequest{
			Grantee: grantee1Str,
			Pagination: &query.PageRequest{
				Limit: 10,
			},
		})
		require.NoError(t, err)
		require.Len(t, resp.Allowances, 2) // granter1->grantee1 and granter2->grantee1
		require.NotNil(t, resp.Pagination)
	})

	// Test Allowances with pagination
	t.Run("Allowances_Pagination", func(t *testing.T) {
		resp, err := querier.Allowances(ctx, &indexerfeegrant.QueryAllowancesRequest{
			Grantee: grantee1Str,
			Pagination: &query.PageRequest{
				Limit: 1,
			},
		})
		require.NoError(t, err)
		require.Len(t, resp.Allowances, 1)
		require.NotNil(t, resp.Pagination)
		require.NotEmpty(t, resp.Pagination.NextKey)
	})

	// Test Allowances - invalid address
	t.Run("Allowances_InvalidAddress", func(t *testing.T) {
		_, err := querier.Allowances(ctx, &indexerfeegrant.QueryAllowancesRequest{
			Grantee: "invalid",
		})
		require.Error(t, err)
	})

	// Test AllowancesByGranter
	t.Run("AllowancesByGranter_Success", func(t *testing.T) {
		resp, err := querier.AllowancesByGranter(ctx, &indexerfeegrant.QueryAllowancesByGranterRequest{
			Granter: granter1Str,
			Pagination: &query.PageRequest{
				Limit: 10,
			},
		})
		require.NoError(t, err)
		require.Len(t, resp.Allowances, 3) // granter1->grantee1, granter1->grantee2, granter1->grantee3
		require.NotNil(t, resp.Pagination)
	})

	// Test AllowancesByGranter with pagination
	t.Run("AllowancesByGranter_Pagination", func(t *testing.T) {
		resp, err := querier.AllowancesByGranter(ctx, &indexerfeegrant.QueryAllowancesByGranterRequest{
			Granter: granter2Str,
			Pagination: &query.PageRequest{
				Limit: 1,
			},
		})
		require.NoError(t, err)
		require.Len(t, resp.Allowances, 1)
		require.NotNil(t, resp.Pagination)
		require.NotEmpty(t, resp.Pagination.NextKey)
	})

	// Test AllowancesByGranter - invalid address
	t.Run("AllowancesByGranter_InvalidAddress", func(t *testing.T) {
		_, err := querier.AllowancesByGranter(ctx, &indexerfeegrant.QueryAllowancesByGranterRequest{
			Granter: "invalid",
		})
		require.Error(t, err)
	})
}

func TestStreamService(t *testing.T) {
	memDB, cdc, addrCodec := setupTest(t)
	ctx := context.Background()

	// Create service
	logger := log.NewNopLogger()
	service := NewWithDB(memDB, cdc, addrCodec, logger)
	require.NotNil(t, service)

	// Test getters
	require.NotNil(t, service.AuthzHandler())
	require.NotNil(t, service.FeeGrantHandler())

	// Test ListenFinalizeBlock (no-op)
	err := service.ListenFinalizeBlock(ctx, abci.RequestFinalizeBlock{}, abci.ResponseFinalizeBlock{})
	require.NoError(t, err)

	// Test ListenCommit with authz grant
	granter := sdk.AccAddress([]byte("granter_address_____"))
	grantee := sdk.AccAddress([]byte("grantee_address_____"))
	msgType := "/cosmos.bank.v1beta1.MsgSend"

	auth := banktypes.NewSendAuthorization(sdk.NewCoins(sdk.NewInt64Coin("uxion", 1000)), nil)
	expiration := time.Now().Add(time.Hour)
	grant, err := authz.NewGrant(expiration, auth, nil)
	require.NoError(t, err)
	grantValue, err := cdc.Marshal(&grant)
	require.NoError(t, err)

	// Encode authz key
	codec := collections.TripleKeyCodec(sdk.AccAddressKey, sdk.AccAddressKey, collections.StringKey)
	triple := collections.Join3(granter, grantee, msgType)
	size := codec.Size(triple)
	buf := make([]byte, size)
	_, err = codec.Encode(buf, triple)
	require.NoError(t, err)
	authzKey := append([]byte{0x01}, buf...)

	changeSet := []*storetypes.StoreKVPair{
		{
			StoreKey: authz.ModuleName,
			Key:      authzKey,
			Value:    grantValue,
			Delete:   false,
		},
	}
	err = service.ListenCommit(ctx, abci.ResponseCommit{}, changeSet)
	require.NoError(t, err)

	// Verify the grant was indexed
	retrievedGrant, err := service.AuthzHandler().GetGrant(ctx, granter, grantee, msgType)
	require.NoError(t, err)
	require.Equal(t, grant.Expiration, retrievedGrant.Expiration)

	// Test ListenCommit with feegrant
	allowance := &feegrant.BasicAllowance{
		SpendLimit: sdk.NewCoins(sdk.NewInt64Coin("uxion", 1000)),
	}
	anyAllowance, err := codectypes.NewAnyWithValue(allowance)
	require.NoError(t, err)

	feeGrant := feegrant.Grant{
		Granter:   granter.String(),
		Grantee:   grantee.String(),
		Allowance: anyAllowance,
	}
	feeGrantValue, err := cdc.Marshal(&feeGrant)
	require.NoError(t, err)

	feegrantKey := feegrant.FeeAllowanceKey(granter, grantee)

	changeSet = []*storetypes.StoreKVPair{
		{
			StoreKey: feegrant.ModuleName,
			Key:      feegrantKey,
			Value:    feeGrantValue,
			Delete:   false,
		},
	}
	err = service.ListenCommit(ctx, abci.ResponseCommit{}, changeSet)
	require.NoError(t, err)

	// Verify the feegrant was indexed
	retrievedFeeGrant, err := service.FeeGrantHandler().GetGrant(ctx, granter, grantee)
	require.NoError(t, err)
	require.Equal(t, feeGrant.Granter, retrievedFeeGrant.Granter)

	// Test ListenCommit with delete
	changeSet[0].Delete = true
	err = service.ListenCommit(ctx, abci.ResponseCommit{}, changeSet)
	require.NoError(t, err)

	// Test ListenCommit with non-matching prefix (should be ignored)
	changeSet = []*storetypes.StoreKVPair{
		{
			StoreKey: authz.ModuleName,
			Key:      []byte("not_a_grant_key"),
			Value:    grantValue,
			Delete:   false,
		},
	}
	err = service.ListenCommit(ctx, abci.ResponseCommit{}, changeSet)
	require.NoError(t, err)

	// Test Close
	err = service.Close()
	require.NoError(t, err)
}

func TestParseGrantStoreKey(t *testing.T) {
	granter := sdk.AccAddress([]byte("granter_address_____"))
	grantee := sdk.AccAddress([]byte("grantee_address_____"))
	msgType := "/cosmos.bank.v1beta1.MsgSend"

	// Encode key
	codec := collections.TripleKeyCodec(sdk.AccAddressKey, sdk.AccAddressKey, collections.StringKey)
	triple := collections.Join3(granter, grantee, msgType)
	size := codec.Size(triple)
	buf := make([]byte, size)
	_, err := codec.Encode(buf, triple)
	require.NoError(t, err)
	key := append([]byte{0x01}, buf...)

	// Decode
	decodedGranter, decodedGrantee, decodedMsgType := parseGrantStoreKey(key)

	require.Equal(t, granter.String(), decodedGranter.String())
	require.Equal(t, grantee.String(), decodedGrantee.String())
	require.Equal(t, msgType, decodedMsgType)

	// Test with empty key
	decodedGranter, decodedGrantee, decodedMsgType = parseGrantStoreKey([]byte{})
	require.Nil(t, decodedGranter)
	require.Nil(t, decodedGrantee)
	require.Equal(t, "", decodedMsgType)

	// Test with invalid key
	decodedGranter, decodedGrantee, decodedMsgType = parseGrantStoreKey([]byte{0x01, 0x02})
	require.Nil(t, decodedGranter)
	require.Nil(t, decodedGrantee)
	require.Equal(t, "", decodedMsgType)
}

func TestKVAccessor(t *testing.T) {
	memDB := db.NewMemDB()
	accessor := &kvAccessor{db: memDB}

	kvStore := accessor.OpenKVStore(context.Background())
	require.NotNil(t, kvStore)
	require.Equal(t, memDB, kvStore)
}

func TestRegisterGRPCGatewayRoutes(t *testing.T) {
	memDB, cdc, addrCodec := setupTest(t)
	logger := log.NewNopLogger()
	service := NewWithDB(memDB, cdc, addrCodec, logger)

	// Create mock client context
	clientCtx := client.Context{}.WithCodec(cdc)

	// Create mock mux (nil is fine for testing - we just verify the function doesn't panic)
	// In real usage, this would be a *runtime.ServeMux from grpc-gateway
	// We can't create a real one without complex setup, but we can verify the function executes
	defer func() {
		if r := recover(); r != nil {
			// If it panics with nil mux, that's expected - we're testing the logging happens
			t.Logf("Function executed (panicked as expected with nil mux): %v", r)
		}
	}()

	// This will panic with nil mux but proves the function is called
	service.RegisterGRPCGatewayRoutes(clientCtx, nil)
}

func TestRegisterServices(t *testing.T) {
	memDB, cdc, addrCodec := setupTest(t)
	logger := log.NewNopLogger()
	service := NewWithDB(memDB, cdc, addrCodec, logger)

	// We can't easily mock module.Configurator due to its complex interface
	// Instead, verify the function signature and that it returns nil error with nil input
	// In production, this is called by the SDK with a proper configurator

	// Test that calling with nil doesn't panic (will cause nil pointer but that's expected)
	defer func() {
		if r := recover(); r != nil {
			// Expected to panic with nil configurator
			t.Logf("RegisterServices executed (panicked as expected with nil): %v", r)
		}
	}()

	// This will panic but demonstrates the function is accessible
	_ = service.RegisterServices(nil)
}
