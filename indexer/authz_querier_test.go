package indexer

import (
	"testing"

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
