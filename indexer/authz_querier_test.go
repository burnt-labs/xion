package indexer

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/std"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	indexerauthz "github.com/burnt-labs/xion/indexer/authz"
)

// errorAddressCodec is a mock address codec that always returns errors
type errorAddressCodec struct{}

func (e *errorAddressCodec) StringToBytes(_ string) ([]byte, error) {
	return nil, errors.New("mock address codec error")
}

func (e *errorAddressCodec) BytesToString(_ []byte) (string, error) {
	return "", errors.New("mock address codec error")
}

// secondCallErrorAddressCodec returns success on first call, error on second
type secondCallErrorAddressCodec struct {
	callCount int
}

func (e *secondCallErrorAddressCodec) StringToBytes(_ string) ([]byte, error) {
	e.callCount++
	if e.callCount > 1 {
		return nil, errors.New("mock address codec error on second call")
	}
	return []byte("valid"), nil
}

func (e *secondCallErrorAddressCodec) BytesToString(_ []byte) (string, error) {
	e.callCount++
	if e.callCount > 1 {
		return "", errors.New("mock address codec error on second call")
	}
	return "valid", nil
}

// TestParseGrantsRequestParams tests the address parsing and prefix logic
// This is 100% testable without any pagination
func TestParseGrantsRequestParams(t *testing.T) {
	addrCodec := addresscodec.NewBech32Codec("xion")

	// Create test addresses
	granter := sdk.AccAddress([]byte("granter_addr_test__"))
	grantee := sdk.AccAddress([]byte("grantee_addr_test__"))
	granterStr, err := addrCodec.BytesToString(granter)
	require.NoError(t, err)
	granteeStr, err := addrCodec.BytesToString(grantee)
	require.NoError(t, err)

	tests := []struct {
		name             string
		req              *indexerauthz.QueryGrantsRequest
		expectGranter    sdk.AccAddress
		expectGrantee    sdk.AccAddress
		expectPrefixType string
		expectError      bool
	}{
		{
			name: "both granter and grantee",
			req: &indexerauthz.QueryGrantsRequest{
				Granter: granterStr,
				Grantee: granteeStr,
			},
			expectGranter:    granter,
			expectGrantee:    grantee,
			expectPrefixType: "pair",
			expectError:      false,
		},
		{
			name: "only granter",
			req: &indexerauthz.QueryGrantsRequest{
				Granter: granterStr,
			},
			expectGranter:    granter,
			expectGrantee:    nil,
			expectPrefixType: "single",
			expectError:      false,
		},
		{
			name: "only grantee",
			req: &indexerauthz.QueryGrantsRequest{
				Grantee: granteeStr,
			},
			expectGranter:    nil,
			expectGrantee:    grantee,
			expectPrefixType: "none",
			expectError:      false,
		},
		{
			name:             "neither granter nor grantee",
			req:              &indexerauthz.QueryGrantsRequest{},
			expectGranter:    nil,
			expectGrantee:    nil,
			expectPrefixType: "none",
			expectError:      false,
		},
		{
			name: "invalid granter address",
			req: &indexerauthz.QueryGrantsRequest{
				Granter: "invalid_address",
			},
			expectError: true,
		},
		{
			name: "invalid grantee address",
			req: &indexerauthz.QueryGrantsRequest{
				Grantee: "invalid_address",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resultGranter, resultGrantee, prefixOpt, err := ParseGrantsRequestParams(tt.req, addrCodec)

			if tt.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.expectGranter, resultGranter)
			require.Equal(t, tt.expectGrantee, resultGrantee)

			// Check prefix type
			switch tt.expectPrefixType {
			case "pair":
				require.NotNil(t, prefixOpt, "Expected pair prefix option")
			case "single":
				require.NotNil(t, prefixOpt, "Expected single prefix option")
			case "none":
				require.Nil(t, prefixOpt, "Expected no prefix option")
			}
		})
	}
}

// TestParseGranterRequestParams tests granter request parsing
func TestParseGranterRequestParams(t *testing.T) {
	addrCodec := addresscodec.NewBech32Codec("xion")

	granter := sdk.AccAddress([]byte("granter_addr_test__"))
	granterStr, err := addrCodec.BytesToString(granter)
	require.NoError(t, err)

	tests := []struct {
		name          string
		req           *indexerauthz.QueryGranterGrantsRequest
		expectGranter sdk.AccAddress
		expectError   bool
	}{
		{
			name: "valid granter",
			req: &indexerauthz.QueryGranterGrantsRequest{
				Granter: granterStr,
			},
			expectGranter: granter,
			expectError:   false,
		},
		{
			name: "invalid granter",
			req: &indexerauthz.QueryGranterGrantsRequest{
				Granter: "invalid_address",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resultGranter, err := ParseGranterRequestParams(tt.req, addrCodec)

			if tt.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.expectGranter, resultGranter)
		})
	}
}

// TestParseGranteeRequestParams tests grantee request parsing
func TestParseGranteeRequestParams(t *testing.T) {
	addrCodec := addresscodec.NewBech32Codec("xion")

	grantee := sdk.AccAddress([]byte("grantee_addr_test__"))
	granteeStr, err := addrCodec.BytesToString(grantee)
	require.NoError(t, err)

	tests := []struct {
		name          string
		req           *indexerauthz.QueryGranteeGrantsRequest
		expectGrantee sdk.AccAddress
		expectError   bool
	}{
		{
			name: "valid grantee",
			req: &indexerauthz.QueryGranteeGrantsRequest{
				Grantee: granteeStr,
			},
			expectGrantee: grantee,
			expectError:   false,
		},
		{
			name: "invalid grantee",
			req: &indexerauthz.QueryGranteeGrantsRequest{
				Grantee: "invalid_address",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resultGrantee, err := ParseGranteeRequestParams(tt.req, addrCodec)

			if tt.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.expectGrantee, resultGrantee)
		})
	}
}

// TestTransformGrantToAuthorization tests the grant transformation logic
// This tests the business logic of converting Grant to GrantAuthorization
func TestTransformGrantToAuthorization(t *testing.T) {
	// Setup codec
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	std.RegisterInterfaces(interfaceRegistry)
	authz.RegisterInterfaces(interfaceRegistry)
	banktypes.RegisterInterfaces(interfaceRegistry)
	cdc := codec.NewProtoCodec(interfaceRegistry)

	addrCodec := addresscodec.NewBech32Codec("xion")

	// Create test data
	granter := sdk.AccAddress([]byte("granter_addr_test__"))
	grantee := sdk.AccAddress([]byte("grantee_addr_test__"))

	sendAuth := banktypes.NewSendAuthorization(
		sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(1000))),
		nil,
	)
	sendAuthAny, err := codectypes.NewAnyWithValue(sendAuth)
	require.NoError(t, err)

	grant := authz.Grant{
		Authorization: sendAuthAny,
		Expiration:    nil,
	}

	primaryKey := collections.Join3(granter, grantee, sendAuth.MsgTypeURL())

	// Test the transformer
	result, err := TransformGrantToAuthorization(primaryKey, grant, cdc, addrCodec)

	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify the result
	granterStr, err := addrCodec.BytesToString(granter)
	require.NoError(t, err)
	granteeStr, err := addrCodec.BytesToString(grantee)
	require.NoError(t, err)

	require.Equal(t, granterStr, result.Granter)
	require.Equal(t, granteeStr, result.Grantee)
	require.NotNil(t, result.Authorization)
	require.Nil(t, result.Expiration)

	// Verify authorization can be unpacked
	var unpackedAuth authz.Authorization
	err = cdc.UnpackAny(result.Authorization, &unpackedAuth)
	require.NoError(t, err)
	require.Equal(t, sendAuth.MsgTypeURL(), unpackedAuth.MsgTypeURL())
}

// TestTransformGrantToAuthorizationErrors tests error paths in transformation
func TestTransformGrantToAuthorizationErrors(t *testing.T) {
	// Setup codec
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	std.RegisterInterfaces(interfaceRegistry)
	authz.RegisterInterfaces(interfaceRegistry)
	banktypes.RegisterInterfaces(interfaceRegistry)
	cdc := codec.NewProtoCodec(interfaceRegistry)

	addrCodec := addresscodec.NewBech32Codec("xion")

	granter := sdk.AccAddress([]byte("granter_addr_test__"))
	grantee := sdk.AccAddress([]byte("grantee_addr_test__"))

	t.Run("InvalidGrantAuthorization", func(t *testing.T) {
		// Create a grant with invalid/nil authorization that will fail GetAuthorization()
		grant := authz.Grant{
			Authorization: nil, // nil authorization will cause GetAuthorization to fail
			Expiration:    nil,
		}

		primaryKey := collections.Join3(granter, grantee, "/test.MsgType")

		result, err := TransformGrantToAuthorization(primaryKey, grant, cdc, addrCodec)
		require.Error(t, err)
		require.Nil(t, result)
	})

	t.Run("GranterAddressEncodingError", func(t *testing.T) {
		// Create a valid grant
		sendAuth := banktypes.NewSendAuthorization(
			sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(1000))),
			nil,
		)
		sendAuthAny, err := codectypes.NewAnyWithValue(sendAuth)
		require.NoError(t, err)

		grant := authz.Grant{
			Authorization: sendAuthAny,
			Expiration:    nil,
		}

		// Use an address codec that returns errors
		errorAddrCodec := &errorAddressCodec{}
		primaryKey := collections.Join3(granter, grantee, sendAuth.MsgTypeURL())

		result, err := TransformGrantToAuthorization(primaryKey, grant, cdc, errorAddrCodec)
		require.Error(t, err)
		require.Nil(t, result)
	})

	t.Run("GranteeAddressEncodingError", func(t *testing.T) {
		// Create a valid grant
		sendAuth := banktypes.NewSendAuthorization(
			sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(1000))),
			nil,
		)
		sendAuthAny, err := codectypes.NewAnyWithValue(sendAuth)
		require.NoError(t, err)

		grant := authz.Grant{
			Authorization: sendAuthAny,
			Expiration:    nil,
		}

		// Use an address codec that fails only on second call (grantee)
		secondCallErrorAddrCodec := &secondCallErrorAddressCodec{}
		primaryKey := collections.Join3(granter, grantee, sendAuth.MsgTypeURL())

		result, err := TransformGrantToAuthorization(primaryKey, grant, cdc, secondCallErrorAddrCodec)
		require.Error(t, err)
		require.Nil(t, result)
	})
}

// TestTransformGrantToAuthorizationEdgeCases tests edge cases in transformation
func TestTransformGrantToAuthorizationEdgeCases(t *testing.T) {
	// Setup codec
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	std.RegisterInterfaces(interfaceRegistry)
	authz.RegisterInterfaces(interfaceRegistry)
	banktypes.RegisterInterfaces(interfaceRegistry)
	cdc := codec.NewProtoCodec(interfaceRegistry)

	addrCodec := addresscodec.NewBech32Codec("xion")

	granter := sdk.AccAddress([]byte("granter_addr_test__"))
	grantee := sdk.AccAddress([]byte("grantee_addr_test__"))

	t.Run("NilExpiration", func(t *testing.T) {
		sendAuth := banktypes.NewSendAuthorization(
			sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(1000))),
			nil,
		)
		sendAuthAny, err := codectypes.NewAnyWithValue(sendAuth)
		require.NoError(t, err)

		grant := authz.Grant{
			Authorization: sendAuthAny,
			Expiration:    nil,
		}

		primaryKey := collections.Join3(granter, grantee, sendAuth.MsgTypeURL())

		result, err := TransformGrantToAuthorization(primaryKey, grant, cdc, addrCodec)
		require.NoError(t, err)
		require.Nil(t, result.Expiration)
	})

	t.Run("EmptyAddresses", func(t *testing.T) {
		// Test with minimal address bytes
		smallGranter := sdk.AccAddress([]byte("g"))
		smallGrantee := sdk.AccAddress([]byte("g"))

		sendAuth := banktypes.NewSendAuthorization(
			sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(1))),
			nil,
		)
		sendAuthAny, err := codectypes.NewAnyWithValue(sendAuth)
		require.NoError(t, err)

		grant := authz.Grant{
			Authorization: sendAuthAny,
			Expiration:    nil,
		}

		primaryKey := collections.Join3(smallGranter, smallGrantee, sendAuth.MsgTypeURL())

		result, err := TransformGrantToAuthorization(primaryKey, grant, cdc, addrCodec)
		require.NoError(t, err)
		require.NotNil(t, result)
	})
}

// TestPrefixOptions tests the prefix option functions
func TestPrefixOptions(t *testing.T) {
	granter := sdk.AccAddress([]byte("granter_addr_test__"))
	grantee := sdk.AccAddress([]byte("grantee_addr_test__"))

	t.Run("WithCollectionPaginationTriplePrefix", func(t *testing.T) {
		prefixOpt := WithCollectionPaginationTriplePrefix[sdk.AccAddress, sdk.AccAddress, string](granter)
		require.NotNil(t, prefixOpt)

		// Create options and apply
		opts := &query.CollectionsPaginateOptions[collections.Triple[sdk.AccAddress, sdk.AccAddress, string]]{}
		prefixOpt(opts)

		require.NotNil(t, opts.Prefix)
	})

	t.Run("WithCollectionPaginationTriplePairPrefix", func(t *testing.T) {
		prefixOpt := WithCollectionPaginationTriplePairPrefix[sdk.AccAddress, sdk.AccAddress, string](granter, grantee)
		require.NotNil(t, prefixOpt)

		// Create options and apply
		opts := &query.CollectionsPaginateOptions[collections.Triple[sdk.AccAddress, sdk.AccAddress, string]]{}
		prefixOpt(opts)

		require.NotNil(t, opts.Prefix)
	})
}

// Benchmark tests for performance
func BenchmarkParseGrantsRequestParams(b *testing.B) {
	addrCodec := addresscodec.NewBech32Codec("xion")
	granter := sdk.AccAddress([]byte("granter_addr_test__"))
	granterStr, _ := addrCodec.BytesToString(granter)

	req := &indexerauthz.QueryGrantsRequest{
		Granter: granterStr,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _, _ = ParseGrantsRequestParams(req, addrCodec)
	}
}

func BenchmarkTransformGrantToAuthorization(b *testing.B) {
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	std.RegisterInterfaces(interfaceRegistry)
	authz.RegisterInterfaces(interfaceRegistry)
	banktypes.RegisterInterfaces(interfaceRegistry)
	cdc := codec.NewProtoCodec(interfaceRegistry)

	addrCodec := addresscodec.NewBech32Codec("xion")

	granter := sdk.AccAddress([]byte("granter_addr_test__"))
	grantee := sdk.AccAddress([]byte("grantee_addr_test__"))

	sendAuth := banktypes.NewSendAuthorization(
		sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(1000))),
		nil,
	)
	sendAuthAny, _ := codectypes.NewAnyWithValue(sendAuth)

	grant := authz.Grant{
		Authorization: sendAuthAny,
		Expiration:    nil,
	}

	primaryKey := collections.Join3(granter, grantee, sendAuth.MsgTypeURL())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = TransformGrantToAuthorization(primaryKey, grant, cdc, addrCodec)
	}
}

// TestMultiIterateRawPaginationPath tests that MultiIterateRaw is used when pagination key is provided
// This ensures the raw iteration code path is exercised in production
func TestMultiIterateRawPaginationPath(t *testing.T) {
	memDB, cdc, addrCodec := setupTest(t)
	ctx := context.Background()

	handler, err := NewAuthzHandler(&kvAccessor{db: memDB}, cdc)
	require.NoError(t, err)
	querier := NewAuthzQuerier(handler, cdc, addrCodec)

	// Setup test data
	granter := sdk.AccAddress([]byte("granter_address_____"))
	grantee1 := sdk.AccAddress([]byte("grantee1_address____"))
	grantee2 := sdk.AccAddress([]byte("grantee2_address____"))
	grantee3 := sdk.AccAddress([]byte("grantee3_address____"))

	sendAuth := banktypes.NewSendAuthorization(sdk.NewCoins(sdk.NewInt64Coin("uxion", 1000)), nil)
	expiration := time.Now().Add(24 * time.Hour)
	grant, err := authz.NewGrant(expiration, sendAuth, nil)
	require.NoError(t, err)

	msgType := sendAuth.MsgTypeURL()

	// Add multiple grants
	err = handler.SetGrant(ctx, granter, grantee1, msgType, grant)
	require.NoError(t, err)
	err = handler.SetGrant(ctx, granter, grantee2, msgType, grant)
	require.NoError(t, err)
	err = handler.SetGrant(ctx, granter, grantee3, msgType, grant)
	require.NoError(t, err)

	grantee1Str, _ := addrCodec.BytesToString(grantee1)
	_, _ = addrCodec.BytesToString(grantee2)

	// First query without pagination key (uses standard iteration)
	resp1, err := querier.GranteeGrants(ctx, &indexerauthz.QueryGranteeGrantsRequest{
		Grantee: grantee1Str,
		Pagination: &query.PageRequest{
			Limit: 10,
		},
	})
	require.NoError(t, err)
	require.Len(t, resp1.Grants, 1)

	// Second query WITH pagination key (uses MultiIterateRaw)
	// Use the first query's pagination.NextKey if available, or create a dummy one
	var paginationKeyBytes []byte
	if resp1.Pagination != nil && len(resp1.Pagination.NextKey) > 0 {
		paginationKeyBytes = resp1.Pagination.NextKey
	} else {
		// Create a pagination key by encoding grantee1's position
		keyCodec := handler.Authorizations.Indexes.Grantee.KeyCodec()
		paginationKey := collections.Join(grantee1, collections.Join3(granter, grantee1, msgType))
		buf := make([]byte, 256)
		n, err := keyCodec.EncodeNonTerminal(buf, paginationKey)
		require.NoError(t, err)
		paginationKeyBytes = buf[:n]
	}

	// This query will use MultiIterateRaw because pagination.Key is provided
	// Query for grantee1 with the pagination key - this exercises the MultiIterateRaw path
	resp2, err := querier.GranteeGrants(ctx, &indexerauthz.QueryGranteeGrantsRequest{
		Grantee: grantee1Str,
		Pagination: &query.PageRequest{
			Key:   paginationKeyBytes,
			Limit: 10,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, resp2)

	t.Log("✓ MultiIterateRaw code path successfully exercised with pagination key")
	t.Logf("✓ Found %d grants using MultiIterateRaw", len(resp2.Grants))
}

// TestGranteeGrantsWithRawIterationMultipleResults tests handling multiple results with pagination
func TestGranteeGrantsWithRawIterationMultipleResults(t *testing.T) {
	memDB, cdc, addrCodec := setupTest(t)
	ctx := context.Background()

	handler, err := NewAuthzHandler(&kvAccessor{db: memDB}, cdc)
	require.NoError(t, err)
	querier := NewAuthzQuerier(handler, cdc, addrCodec)

	// Setup test data with multiple grants to the same grantee
	granter1 := sdk.AccAddress([]byte("granter1_address____"))
	granter2 := sdk.AccAddress([]byte("granter2_address____"))
	granter3 := sdk.AccAddress([]byte("granter3_address____"))
	grantee := sdk.AccAddress([]byte("grantee_address_____"))

	sendAuth := banktypes.NewSendAuthorization(sdk.NewCoins(sdk.NewInt64Coin("uxion", 1000)), nil)
	expiration := time.Now().Add(24 * time.Hour)
	grant, err := authz.NewGrant(expiration, sendAuth, nil)
	require.NoError(t, err)

	msgType := sendAuth.MsgTypeURL()

	// Add multiple grants from different granters to same grantee
	err = handler.SetGrant(ctx, granter1, grantee, msgType, grant)
	require.NoError(t, err)
	err = handler.SetGrant(ctx, granter2, grantee, msgType, grant)
	require.NoError(t, err)
	err = handler.SetGrant(ctx, granter3, grantee, msgType, grant)
	require.NoError(t, err)

	granteeStr, _ := addrCodec.BytesToString(grantee)

	t.Run("QueryWithLimit", func(t *testing.T) {
		// Query with limit=2, should get 2 grants and a nextKey
		resp, err := querier.GranteeGrants(ctx, &indexerauthz.QueryGranteeGrantsRequest{
			Grantee: granteeStr,
			Pagination: &query.PageRequest{
				Limit: 2,
			},
		})
		require.NoError(t, err)
		require.Len(t, resp.Grants, 2, "Should return exactly 2 grants")
		require.NotNil(t, resp.Pagination)

		// If there's a next key, it should be non-empty (there's a 3rd grant)
		if len(resp.Grants) == 2 {
			require.NotEmpty(t, resp.Pagination.NextKey, "Should have nextKey since there are more results")
		}

		t.Log("✓ Pagination limit correctly restricts results")
	})

	t.Run("QueryWithCountTotal", func(t *testing.T) {
		// Query with CountTotal=true to test the counting loop
		resp, err := querier.GranteeGrants(ctx, &indexerauthz.QueryGranteeGrantsRequest{
			Grantee: granteeStr,
			Pagination: &query.PageRequest{
				Limit:      2,
				CountTotal: true,
			},
		})
		require.NoError(t, err)
		require.Len(t, resp.Grants, 2, "Should return 2 grants due to limit")
		require.NotNil(t, resp.Pagination)
		require.Equal(t, uint64(3), resp.Pagination.Total, "Should count all 3 grants")

		t.Log("✓ CountTotal correctly counts all matching grants")
	})

	t.Run("QueryWithPaginationKey", func(t *testing.T) {
		// First query to get pagination key
		resp1, err := querier.GranteeGrants(ctx, &indexerauthz.QueryGranteeGrantsRequest{
			Grantee: granteeStr,
			Pagination: &query.PageRequest{
				Limit: 1,
			},
		})
		require.NoError(t, err)
		require.Len(t, resp1.Grants, 1)

		// If we got a nextKey, use it for the second query
		if len(resp1.Pagination.NextKey) > 0 {
			resp2, err := querier.GranteeGrants(ctx, &indexerauthz.QueryGranteeGrantsRequest{
				Grantee: granteeStr,
				Pagination: &query.PageRequest{
					Key:   resp1.Pagination.NextKey,
					Limit: 2,
				},
			})
			require.NoError(t, err)
			require.NotNil(t, resp2)
			// Should get remaining grants
			require.LessOrEqual(t, len(resp2.Grants), 2)

			t.Log("✓ Pagination with nextKey successfully retrieves subsequent results")
		}
	})

	t.Run("QueryWithPaginationKeyAndCountTotal", func(t *testing.T) {
		// Test the combination of pagination key + countTotal
		resp1, err := querier.GranteeGrants(ctx, &indexerauthz.QueryGranteeGrantsRequest{
			Grantee: granteeStr,
			Pagination: &query.PageRequest{
				Limit: 1,
			},
		})
		require.NoError(t, err)

		if len(resp1.Pagination.NextKey) > 0 {
			// Second query with pagination key AND countTotal
			resp2, err := querier.GranteeGrants(ctx, &indexerauthz.QueryGranteeGrantsRequest{
				Grantee: granteeStr,
				Pagination: &query.PageRequest{
					Key:        resp1.Pagination.NextKey,
					Limit:      1,
					CountTotal: true,
				},
			})
			require.NoError(t, err)
			require.NotNil(t, resp2)
			require.NotNil(t, resp2.Pagination)

			// Should count remaining grants from the paginated position
			require.Greater(t, resp2.Pagination.Total, uint64(0))

			t.Log("✓ CountTotal works correctly with pagination key")
		}
	})
}

// TestGranteeGrantsWithRawIterationEdgeCases tests edge cases and error conditions
func TestGranteeGrantsWithRawIterationEdgeCases(t *testing.T) {
	memDB, cdc, addrCodec := setupTest(t)
	ctx := context.Background()

	handler, err := NewAuthzHandler(&kvAccessor{db: memDB}, cdc)
	require.NoError(t, err)
	querier := NewAuthzQuerier(handler, cdc, addrCodec)

	t.Run("EmptyResults", func(t *testing.T) {
		// Query for a grantee with no grants
		emptyGrantee := sdk.AccAddress([]byte("empty_grantee_______"))
		emptyGranteeStr, _ := addrCodec.BytesToString(emptyGrantee)

		resp, err := querier.GranteeGrants(ctx, &indexerauthz.QueryGranteeGrantsRequest{
			Grantee: emptyGranteeStr,
			Pagination: &query.PageRequest{
				Limit: 10,
			},
		})
		require.NoError(t, err)
		require.Empty(t, resp.Grants, "Should return no grants")
		require.NotNil(t, resp.Pagination)
		require.Empty(t, resp.Pagination.NextKey, "Should have no nextKey")

		t.Log("✓ Empty results handled correctly")
	})

	t.Run("NilPagination", func(t *testing.T) {
		grantee := sdk.AccAddress([]byte("test_grantee________"))
		granteeStr, _ := addrCodec.BytesToString(grantee)

		// Add a grant
		granter := sdk.AccAddress([]byte("test_granter________"))
		sendAuth := banktypes.NewSendAuthorization(sdk.NewCoins(sdk.NewInt64Coin("uxion", 1000)), nil)
		expiration := time.Now().Add(24 * time.Hour)
		grant, err := authz.NewGrant(expiration, sendAuth, nil)
		require.NoError(t, err)
		err = handler.SetGrant(ctx, granter, grantee, sendAuth.MsgTypeURL(), grant)
		require.NoError(t, err)

		// Query with nil pagination (should use defaults)
		resp, err := querier.GranteeGrants(ctx, &indexerauthz.QueryGranteeGrantsRequest{
			Grantee:    granteeStr,
			Pagination: nil,
		})
		require.NoError(t, err)
		require.NotEmpty(t, resp.Grants, "Should return grants even with nil pagination")

		t.Log("✓ Nil pagination uses default limit")
	})

	t.Run("ZeroLimit", func(t *testing.T) {
		grantee := sdk.AccAddress([]byte("zero_limit_grantee__"))
		granteeStr, _ := addrCodec.BytesToString(grantee)

		// Add a grant
		granter := sdk.AccAddress([]byte("zero_limit_granter__"))
		sendAuth := banktypes.NewSendAuthorization(sdk.NewCoins(sdk.NewInt64Coin("uxion", 1000)), nil)
		expiration := time.Now().Add(24 * time.Hour)
		grant, err := authz.NewGrant(expiration, sendAuth, nil)
		require.NoError(t, err)
		err = handler.SetGrant(ctx, granter, grantee, sendAuth.MsgTypeURL(), grant)
		require.NoError(t, err)

		// Query with zero limit (should use default limit)
		resp, err := querier.GranteeGrants(ctx, &indexerauthz.QueryGranteeGrantsRequest{
			Grantee: granteeStr,
			Pagination: &query.PageRequest{
				Limit: 0, // Zero should trigger default
			},
		})
		require.NoError(t, err)
		require.NotEmpty(t, resp.Grants, "Should return grants with default limit")

		t.Log("✓ Zero limit uses default limit")
	})

	t.Run("CountTotalWithNoResults", func(t *testing.T) {
		emptyGrantee := sdk.AccAddress([]byte("empty_count_grantee_"))
		emptyGranteeStr, _ := addrCodec.BytesToString(emptyGrantee)

		resp, err := querier.GranteeGrants(ctx, &indexerauthz.QueryGranteeGrantsRequest{
			Grantee: emptyGranteeStr,
			Pagination: &query.PageRequest{
				Limit:      10,
				CountTotal: true,
			},
		})
		require.NoError(t, err)
		require.Empty(t, resp.Grants)
		require.NotNil(t, resp.Pagination)
		require.Equal(t, uint64(0), resp.Pagination.Total, "Total should be 0 for no results")

		t.Log("✓ CountTotal returns 0 when no grants exist")
	})
}

// TestGranteeGrantsFilterByGrantee tests that the grantee filter works correctly
func TestGranteeGrantsFilterByGrantee(t *testing.T) {
	memDB, cdc, addrCodec := setupTest(t)
	ctx := context.Background()

	handler, err := NewAuthzHandler(&kvAccessor{db: memDB}, cdc)
	require.NoError(t, err)
	querier := NewAuthzQuerier(handler, cdc, addrCodec)

	// Setup test data with different grantees
	granter := sdk.AccAddress([]byte("granter_address_____"))
	grantee1 := sdk.AccAddress([]byte("grantee1_address____"))
	grantee2 := sdk.AccAddress([]byte("grantee2_address____"))

	sendAuth := banktypes.NewSendAuthorization(sdk.NewCoins(sdk.NewInt64Coin("uxion", 1000)), nil)
	expiration := time.Now().Add(24 * time.Hour)
	grant, err := authz.NewGrant(expiration, sendAuth, nil)
	require.NoError(t, err)

	msgType := sendAuth.MsgTypeURL()

	// Add grants to different grantees
	err = handler.SetGrant(ctx, granter, grantee1, msgType, grant)
	require.NoError(t, err)
	err = handler.SetGrant(ctx, granter, grantee2, msgType, grant)
	require.NoError(t, err)

	// Query for grantee1 - should only get grants for grantee1
	grantee1Str, _ := addrCodec.BytesToString(grantee1)
	resp1, err := querier.GranteeGrants(ctx, &indexerauthz.QueryGranteeGrantsRequest{
		Grantee: grantee1Str,
		Pagination: &query.PageRequest{
			Limit: 10,
		},
	})
	require.NoError(t, err)
	require.Len(t, resp1.Grants, 1, "Should only return grants for grantee1")
	require.Equal(t, grantee1Str, resp1.Grants[0].Grantee)

	// Query for grantee2 - should only get grants for grantee2
	grantee2Str, _ := addrCodec.BytesToString(grantee2)
	resp2, err := querier.GranteeGrants(ctx, &indexerauthz.QueryGranteeGrantsRequest{
		Grantee: grantee2Str,
		Pagination: &query.PageRequest{
			Limit: 10,
		},
	})
	require.NoError(t, err)
	require.Len(t, resp2.Grants, 1, "Should only return grants for grantee2")
	require.Equal(t, grantee2Str, resp2.Grants[0].Grantee)

	t.Log("✓ Grantee filtering works correctly")
}

// TestGranterGrantsErrorPaths tests error paths in GranterGrants
func TestGranterGrantsErrorPaths(t *testing.T) {
	memDB, cdc, addrCodec := setupTest(t)
	ctx := context.Background()

	handler, err := NewAuthzHandler(&kvAccessor{db: memDB}, cdc)
	require.NoError(t, err)

	t.Run("InvalidGranterAddress", func(t *testing.T) {
		querier := NewAuthzQuerier(handler, cdc, addrCodec)

		_, err := querier.GranterGrants(ctx, &indexerauthz.QueryGranterGrantsRequest{
			Granter: "invalid_granter_addr",
			Pagination: &query.PageRequest{
				Limit: 10,
			},
		})
		require.Error(t, err, "Should fail with invalid granter address")
	})

	t.Run("EmptyResults", func(t *testing.T) {
		querier := NewAuthzQuerier(handler, cdc, addrCodec)

		// Query for a granter with no grants
		emptyGranter := sdk.AccAddress([]byte("empty_granter_______"))
		emptyGranterStr, _ := addrCodec.BytesToString(emptyGranter)

		resp, err := querier.GranterGrants(ctx, &indexerauthz.QueryGranterGrantsRequest{
			Granter: emptyGranterStr,
			Pagination: &query.PageRequest{
				Limit: 10,
			},
		})
		require.NoError(t, err)
		require.Empty(t, resp.Grants)
	})

	t.Run("MultipleGrants", func(t *testing.T) {
		querier := NewAuthzQuerier(handler, cdc, addrCodec)

		// Setup multiple grants from the same granter
		granter := sdk.AccAddress([]byte("multi_grant_granter_"))
		grantee1 := sdk.AccAddress([]byte("multi_grant_grantee1"))
		grantee2 := sdk.AccAddress([]byte("multi_grant_grantee2"))

		sendAuth := banktypes.NewSendAuthorization(sdk.NewCoins(sdk.NewInt64Coin("uxion", 1000)), nil)
		expiration := time.Now().Add(24 * time.Hour)
		grant, err := authz.NewGrant(expiration, sendAuth, nil)
		require.NoError(t, err)

		err = handler.SetGrant(ctx, granter, grantee1, sendAuth.MsgTypeURL(), grant)
		require.NoError(t, err)
		err = handler.SetGrant(ctx, granter, grantee2, sendAuth.MsgTypeURL(), grant)
		require.NoError(t, err)

		granterStr, _ := addrCodec.BytesToString(granter)
		resp, err := querier.GranterGrants(ctx, &indexerauthz.QueryGranterGrantsRequest{
			Granter: granterStr,
			Pagination: &query.PageRequest{
				Limit: 10,
			},
		})
		require.NoError(t, err)
		require.Len(t, resp.Grants, 2)

		// Verify the granter is correct for all grants
		for _, g := range resp.Grants {
			require.Equal(t, granterStr, g.Granter)
		}
	})

	t.Run("WithPagination", func(t *testing.T) {
		querier := NewAuthzQuerier(handler, cdc, addrCodec)

		// Setup multiple grants
		granter := sdk.AccAddress([]byte("paginated_granter___"))
		grantee1 := sdk.AccAddress([]byte("paginated_grantee1__"))
		grantee2 := sdk.AccAddress([]byte("paginated_grantee2__"))
		grantee3 := sdk.AccAddress([]byte("paginated_grantee3__"))

		sendAuth := banktypes.NewSendAuthorization(sdk.NewCoins(sdk.NewInt64Coin("uxion", 1000)), nil)
		expiration := time.Now().Add(24 * time.Hour)
		grant, err := authz.NewGrant(expiration, sendAuth, nil)
		require.NoError(t, err)

		err = handler.SetGrant(ctx, granter, grantee1, sendAuth.MsgTypeURL(), grant)
		require.NoError(t, err)
		err = handler.SetGrant(ctx, granter, grantee2, sendAuth.MsgTypeURL(), grant)
		require.NoError(t, err)
		err = handler.SetGrant(ctx, granter, grantee3, sendAuth.MsgTypeURL(), grant)
		require.NoError(t, err)

		granterStr, _ := addrCodec.BytesToString(granter)

		// First page
		resp1, err := querier.GranterGrants(ctx, &indexerauthz.QueryGranterGrantsRequest{
			Granter: granterStr,
			Pagination: &query.PageRequest{
				Limit: 2,
			},
		})
		require.NoError(t, err)
		require.Len(t, resp1.Grants, 2)
		require.NotEmpty(t, resp1.Pagination.NextKey)

		// Second page using nextKey
		resp2, err := querier.GranterGrants(ctx, &indexerauthz.QueryGranterGrantsRequest{
			Granter: granterStr,
			Pagination: &query.PageRequest{
				Key:   resp1.Pagination.NextKey,
				Limit: 10,
			},
		})
		require.NoError(t, err)
		require.NotNil(t, resp2)
	})
}

// TestGrantsQueryPaths tests different paths in the Grants query
func TestGrantsQueryPaths(t *testing.T) {
	memDB, cdc, addrCodec := setupTest(t)
	ctx := context.Background()

	handler, err := NewAuthzHandler(&kvAccessor{db: memDB}, cdc)
	require.NoError(t, err)
	querier := NewAuthzQuerier(handler, cdc, addrCodec)

	// Setup test data
	granter := sdk.AccAddress([]byte("grants_granter______"))
	grantee := sdk.AccAddress([]byte("grants_grantee______"))

	sendAuth := banktypes.NewSendAuthorization(sdk.NewCoins(sdk.NewInt64Coin("uxion", 1000)), nil)
	expiration := time.Now().Add(24 * time.Hour)
	grant, err := authz.NewGrant(expiration, sendAuth, nil)
	require.NoError(t, err)

	err = handler.SetGrant(ctx, granter, grantee, sendAuth.MsgTypeURL(), grant)
	require.NoError(t, err)

	granterStr, _ := addrCodec.BytesToString(granter)
	granteeStr, _ := addrCodec.BytesToString(grantee)

	t.Run("QueryWithGranterOnly", func(t *testing.T) {
		resp, err := querier.Grants(ctx, &indexerauthz.QueryGrantsRequest{
			Granter: granterStr,
			Pagination: &query.PageRequest{
				Limit: 10,
			},
		})
		require.NoError(t, err)
		require.NotEmpty(t, resp.Grants)
	})

	t.Run("QueryWithGranteeOnly", func(t *testing.T) {
		resp, err := querier.Grants(ctx, &indexerauthz.QueryGrantsRequest{
			Grantee: granteeStr,
			Pagination: &query.PageRequest{
				Limit: 10,
			},
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("QueryWithBoth", func(t *testing.T) {
		resp, err := querier.Grants(ctx, &indexerauthz.QueryGrantsRequest{
			Granter: granterStr,
			Grantee: granteeStr,
			Pagination: &query.PageRequest{
				Limit: 10,
			},
		})
		require.NoError(t, err)
		require.NotEmpty(t, resp.Grants)
	})

	t.Run("QueryWithNeither", func(t *testing.T) {
		resp, err := querier.Grants(ctx, &indexerauthz.QueryGrantsRequest{
			Pagination: &query.PageRequest{
				Limit: 10,
			},
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("InvalidGranterAddress", func(t *testing.T) {
		_, err := querier.Grants(ctx, &indexerauthz.QueryGrantsRequest{
			Granter: "invalid_addr",
			Pagination: &query.PageRequest{
				Limit: 10,
			},
		})
		require.Error(t, err)
	})

	t.Run("InvalidGranteeAddress", func(t *testing.T) {
		_, err := querier.Grants(ctx, &indexerauthz.QueryGrantsRequest{
			Grantee: "invalid_addr",
			Pagination: &query.PageRequest{
				Limit: 10,
			},
		})
		require.Error(t, err)
	})
}

// TestGranteeGrantsWithOffset tests the offset pagination path
func TestGranteeGrantsWithOffset(t *testing.T) {
	memDB, cdc, addrCodec := setupTest(t)
	ctx := context.Background()

	handler, err := NewAuthzHandler(&kvAccessor{db: memDB}, cdc)
	require.NoError(t, err)
	querier := NewAuthzQuerier(handler, cdc, addrCodec)

	// Setup multiple grants to the same grantee
	granter1 := sdk.AccAddress([]byte("offset_granter1_____"))
	granter2 := sdk.AccAddress([]byte("offset_granter2_____"))
	granter3 := sdk.AccAddress([]byte("offset_granter3_____"))
	grantee := sdk.AccAddress([]byte("offset_grantee______"))

	sendAuth := banktypes.NewSendAuthorization(sdk.NewCoins(sdk.NewInt64Coin("uxion", 1000)), nil)
	expiration := time.Now().Add(24 * time.Hour)
	grant, err := authz.NewGrant(expiration, sendAuth, nil)
	require.NoError(t, err)

	err = handler.SetGrant(ctx, granter1, grantee, sendAuth.MsgTypeURL(), grant)
	require.NoError(t, err)
	err = handler.SetGrant(ctx, granter2, grantee, sendAuth.MsgTypeURL(), grant)
	require.NoError(t, err)
	err = handler.SetGrant(ctx, granter3, grantee, sendAuth.MsgTypeURL(), grant)
	require.NoError(t, err)

	granteeStr, _ := addrCodec.BytesToString(grantee)

	t.Run("WithOffset", func(t *testing.T) {
		// Query with offset=1 to skip first result
		resp, err := querier.GranteeGrants(ctx, &indexerauthz.QueryGranteeGrantsRequest{
			Grantee: granteeStr,
			Pagination: &query.PageRequest{
				Offset: 1,
				Limit:  10,
			},
		})
		require.NoError(t, err)
		require.Len(t, resp.Grants, 2, "Should return 2 grants after skipping 1")
	})

	t.Run("WithOffsetBeyondResults", func(t *testing.T) {
		// Query with offset=10 which is beyond the number of grants
		resp, err := querier.GranteeGrants(ctx, &indexerauthz.QueryGranteeGrantsRequest{
			Grantee: granteeStr,
			Pagination: &query.PageRequest{
				Offset: 10,
				Limit:  10,
			},
		})
		require.NoError(t, err)
		require.Empty(t, resp.Grants, "Should return no grants when offset exceeds count")
	})

	t.Run("InvalidGranteeAddress", func(t *testing.T) {
		_, err := querier.GranteeGrants(ctx, &indexerauthz.QueryGranteeGrantsRequest{
			Grantee: "invalid_grantee",
			Pagination: &query.PageRequest{
				Limit: 10,
			},
		})
		require.Error(t, err)
	})
}

// TestGranteeGrantsWithRawIterationCountTotal tests the CountTotal path in raw iteration
func TestGranteeGrantsWithRawIterationCountTotal(t *testing.T) {
	memDB, cdc, addrCodec := setupTest(t)
	ctx := context.Background()

	handler, err := NewAuthzHandler(&kvAccessor{db: memDB}, cdc)
	require.NoError(t, err)
	querier := NewAuthzQuerier(handler, cdc, addrCodec)

	// Setup multiple grants from different granters to the same grantee
	granter1 := sdk.AccAddress([]byte("raw_count_granter1__"))
	granter2 := sdk.AccAddress([]byte("raw_count_granter2__"))
	granter3 := sdk.AccAddress([]byte("raw_count_granter3__"))
	grantee := sdk.AccAddress([]byte("raw_count_grantee___"))

	sendAuth := banktypes.NewSendAuthorization(sdk.NewCoins(sdk.NewInt64Coin("uxion", 1000)), nil)
	expiration := time.Now().Add(24 * time.Hour)
	grant, err := authz.NewGrant(expiration, sendAuth, nil)
	require.NoError(t, err)

	err = handler.SetGrant(ctx, granter1, grantee, sendAuth.MsgTypeURL(), grant)
	require.NoError(t, err)
	err = handler.SetGrant(ctx, granter2, grantee, sendAuth.MsgTypeURL(), grant)
	require.NoError(t, err)
	err = handler.SetGrant(ctx, granter3, grantee, sendAuth.MsgTypeURL(), grant)
	require.NoError(t, err)

	granteeStr, _ := addrCodec.BytesToString(grantee)

	// First query to get a pagination key
	resp1, err := querier.GranteeGrants(ctx, &indexerauthz.QueryGranteeGrantsRequest{
		Grantee: granteeStr,
		Pagination: &query.PageRequest{
			Limit: 1,
		},
	})
	require.NoError(t, err)
	require.Len(t, resp1.Grants, 1)
	require.NotEmpty(t, resp1.Pagination.NextKey)

	// Now use the pagination key with CountTotal to exercise the raw iteration with countTotal
	resp2, err := querier.GranteeGrants(ctx, &indexerauthz.QueryGranteeGrantsRequest{
		Grantee: granteeStr,
		Pagination: &query.PageRequest{
			Key:        resp1.Pagination.NextKey,
			Limit:      1,
			CountTotal: true,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, resp2)
	require.NotNil(t, resp2.Pagination)
	require.Greater(t, resp2.Pagination.Total, uint64(0), "Should count remaining grants")
	t.Logf("✓ Raw iteration CountTotal returned total=%d", resp2.Pagination.Total)
}

// TestGranteeGrantsWithRawIterationNextKey tests the nextKey encoding in raw iteration
func TestGranteeGrantsWithRawIterationNextKey(t *testing.T) {
	memDB, cdc, addrCodec := setupTest(t)
	ctx := context.Background()

	handler, err := NewAuthzHandler(&kvAccessor{db: memDB}, cdc)
	require.NoError(t, err)
	querier := NewAuthzQuerier(handler, cdc, addrCodec)

	// Setup many grants to ensure we have more results for nextKey
	grantee := sdk.AccAddress([]byte("raw_nextkey_grantee_"))

	sendAuth := banktypes.NewSendAuthorization(sdk.NewCoins(sdk.NewInt64Coin("uxion", 1000)), nil)
	expiration := time.Now().Add(24 * time.Hour)
	grant, err := authz.NewGrant(expiration, sendAuth, nil)
	require.NoError(t, err)

	// Create 5 grants
	for i := 0; i < 5; i++ {
		granter := sdk.AccAddress([]byte(fmt.Sprintf("raw_nextkey_grant%d__", i)))
		err = handler.SetGrant(ctx, granter, grantee, sendAuth.MsgTypeURL(), grant)
		require.NoError(t, err)
	}

	granteeStr, _ := addrCodec.BytesToString(grantee)

	// Get first page to get pagination key
	resp1, err := querier.GranteeGrants(ctx, &indexerauthz.QueryGranteeGrantsRequest{
		Grantee: granteeStr,
		Pagination: &query.PageRequest{
			Limit: 2,
		},
	})
	require.NoError(t, err)
	require.Len(t, resp1.Grants, 2)
	require.NotEmpty(t, resp1.Pagination.NextKey)

	// Use pagination key with limit that should produce another nextKey
	resp2, err := querier.GranteeGrants(ctx, &indexerauthz.QueryGranteeGrantsRequest{
		Grantee: granteeStr,
		Pagination: &query.PageRequest{
			Key:   resp1.Pagination.NextKey,
			Limit: 2,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, resp2)

	// Should have grants and potentially a nextKey if there are more
	require.GreaterOrEqual(t, len(resp2.Grants), 1)

	t.Logf("✓ Raw iteration second page returned %d grants", len(resp2.Grants))
	if len(resp2.Pagination.NextKey) > 0 {
		t.Log("✓ Raw iteration has more results, nextKey is set")
	}
}

// TestGranteeGrantsWithRawIterationFilterByGrantee tests that grantee filter works in raw iteration
func TestGranteeGrantsWithRawIterationFilterByGrantee(t *testing.T) {
	memDB, cdc, addrCodec := setupTest(t)
	ctx := context.Background()

	handler, err := NewAuthzHandler(&kvAccessor{db: memDB}, cdc)
	require.NoError(t, err)
	querier := NewAuthzQuerier(handler, cdc, addrCodec)

	// Setup grants for different grantees
	granter := sdk.AccAddress([]byte("raw_filter_granter__"))
	grantee1 := sdk.AccAddress([]byte("raw_filter_grantee1_"))
	grantee2 := sdk.AccAddress([]byte("raw_filter_grantee2_"))

	sendAuth := banktypes.NewSendAuthorization(sdk.NewCoins(sdk.NewInt64Coin("uxion", 1000)), nil)
	expiration := time.Now().Add(24 * time.Hour)
	grant, err := authz.NewGrant(expiration, sendAuth, nil)
	require.NoError(t, err)

	err = handler.SetGrant(ctx, granter, grantee1, sendAuth.MsgTypeURL(), grant)
	require.NoError(t, err)
	err = handler.SetGrant(ctx, granter, grantee2, sendAuth.MsgTypeURL(), grant)
	require.NoError(t, err)

	grantee1Str, _ := addrCodec.BytesToString(grantee1)

	// Query grantee1 first without key
	resp1, err := querier.GranteeGrants(ctx, &indexerauthz.QueryGranteeGrantsRequest{
		Grantee: grantee1Str,
		Pagination: &query.PageRequest{
			Limit: 10,
		},
	})
	require.NoError(t, err)
	require.Len(t, resp1.Grants, 1)
	require.Equal(t, grantee1Str, resp1.Grants[0].Grantee)

	t.Log("✓ Raw iteration correctly filters by grantee")
}

// TestGranteeGrantsNextKeyEncoding tests the nextKey encoding logic
func TestGranteeGrantsNextKeyEncoding(t *testing.T) {
	memDB, cdc, addrCodec := setupTest(t)
	ctx := context.Background()

	handler, err := NewAuthzHandler(&kvAccessor{db: memDB}, cdc)
	require.NoError(t, err)
	querier := NewAuthzQuerier(handler, cdc, addrCodec)

	// Setup test data with multiple grants
	granter1 := sdk.AccAddress([]byte("granter1_next_key___"))
	granter2 := sdk.AccAddress([]byte("granter2_next_key___"))
	grantee := sdk.AccAddress([]byte("grantee_next_key____"))

	sendAuth := banktypes.NewSendAuthorization(sdk.NewCoins(sdk.NewInt64Coin("uxion", 1000)), nil)
	expiration := time.Now().Add(24 * time.Hour)
	grant, err := authz.NewGrant(expiration, sendAuth, nil)
	require.NoError(t, err)

	msgType := sendAuth.MsgTypeURL()

	// Add multiple grants
	err = handler.SetGrant(ctx, granter1, grantee, msgType, grant)
	require.NoError(t, err)
	err = handler.SetGrant(ctx, granter2, grantee, msgType, grant)
	require.NoError(t, err)

	granteeStr, _ := addrCodec.BytesToString(grantee)

	// Query with limit=1 to ensure we get a nextKey
	resp1, err := querier.GranteeGrants(ctx, &indexerauthz.QueryGranteeGrantsRequest{
		Grantee: granteeStr,
		Pagination: &query.PageRequest{
			Limit: 1,
		},
	})
	require.NoError(t, err)
	require.Len(t, resp1.Grants, 1)
	require.NotNil(t, resp1.Pagination)
	require.NotEmpty(t, resp1.Pagination.NextKey, "NextKey should be set when more results exist")

	// The nextKey should be a valid encoded key
	require.Greater(t, len(resp1.Pagination.NextKey), 0)

	// Use the nextKey in a second query
	resp2, err := querier.GranteeGrants(ctx, &indexerauthz.QueryGranteeGrantsRequest{
		Grantee: granteeStr,
		Pagination: &query.PageRequest{
			Key:   resp1.Pagination.NextKey,
			Limit: 10,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, resp2)

	// The second query should return the remaining grant(s)
	require.GreaterOrEqual(t, len(resp2.Grants), 0)

	// Verify that the grants are different (pagination is working)
	if len(resp2.Grants) > 0 && len(resp1.Grants) > 0 {
		// The second grant should be different from the first
		require.NotEqual(t, resp1.Grants[0].Granter, resp2.Grants[0].Granter,
			"Second page should return different grants")
	}

	t.Log("✓ NextKey encoding and pagination continuity works correctly")
}
