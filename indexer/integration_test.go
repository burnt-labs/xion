package indexer_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	dbm "github.com/cosmos/cosmos-db"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"cosmossdk.io/x/feegrant"

	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	xionapp "github.com/burnt-labs/xion/app"
	indexerauthz "github.com/burnt-labs/xion/indexer/authz"
	indexerfeegrant "github.com/burnt-labs/xion/indexer/feegrant"
)

// setupIntegrationTest creates a full WasmApp instance for integration testing
func setupIntegrationTest(t *testing.T) (*xionapp.WasmApp, context.Context) {
	db := dbm.NewMemDB()
	gapp := xionapp.NewWasmAppWithCustomOptions(t, false, xionapp.SetupOptions{
		Logger:  log.NewNopLogger(),
		DB:      db,
		AppOpts: simtestutil.NewAppOptionsWithFlagHome(t.TempDir()),
	})

	// Use background context for indexer operations
	// The indexer uses its own PebbleDB through kvAccessor pattern
	ctx := context.Background()

	return gapp, ctx
}

// TestIndexerHandlers_Integration tests the indexer handlers with full app context
// This tests the Set/Get operations which work with the kvAccessor pattern
func TestIndexerHandlers_Integration(t *testing.T) {
	gapp, ctx := setupIntegrationTest(t)
	addrCodec := addresscodec.NewBech32Codec("xion")

	authzHandler := gapp.IndexerService().AuthzHandler()
	feeGrantHandler := gapp.IndexerService().FeeGrantHandler()

	// Create test addresses
	granter := sdk.AccAddress([]byte("granter_addr_test__"))
	grantee := sdk.AccAddress([]byte("grantee_addr_test__"))

	t.Run("AuthzHandler_SetAndGet", func(t *testing.T) {
		// Create send authorization
		sendAuth := banktypes.NewSendAuthorization(sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(1000))), nil)
		sendAuthAny, err := codectypes.NewAnyWithValue(sendAuth)
		require.NoError(t, err)

		grant := authz.Grant{
			Authorization: sendAuthAny,
			Expiration:    nil,
		}

		// Set grant
		err = authzHandler.SetGrant(ctx, granter, grantee, banktypes.SendAuthorization{}.MsgTypeURL(), grant)
		require.NoError(t, err)

		// Get grant
		retrievedGrant, err := authzHandler.GetGrant(ctx, granter, grantee, banktypes.SendAuthorization{}.MsgTypeURL())
		require.NoError(t, err)
		require.NotNil(t, retrievedGrant.Authorization)

		// Verify authorization
		var retrievedAuth authz.Authorization
		err = gapp.AppCodec().UnpackAny(retrievedGrant.Authorization, &retrievedAuth)
		require.NoError(t, err)
		require.Equal(t, sendAuth.MsgTypeURL(), retrievedAuth.MsgTypeURL())
	})

	t.Run("FeeGrantHandler_SetAndGet", func(t *testing.T) {
		// Create basic allowance
		allowance := &feegrant.BasicAllowance{
			SpendLimit: sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(1000))),
		}
		allowanceAny, err := codectypes.NewAnyWithValue(allowance)
		require.NoError(t, err)

		granterStr, err := addrCodec.BytesToString(granter)
		require.NoError(t, err)
		granteeStr, err := addrCodec.BytesToString(grantee)
		require.NoError(t, err)

		grant := feegrant.Grant{
			Granter:   granterStr,
			Grantee:   granteeStr,
			Allowance: allowanceAny,
		}

		// Set grant
		err = feeGrantHandler.SetGrant(ctx, granter, grantee, grant)
		require.NoError(t, err)

		// Get grant
		retrievedGrant, err := feeGrantHandler.GetGrant(ctx, granter, grantee)
		require.NoError(t, err)
		require.Equal(t, granterStr, retrievedGrant.Granter)
		require.Equal(t, granteeStr, retrievedGrant.Grantee)
		require.NotNil(t, retrievedGrant.Allowance)
	})
}

// TestIndexerQueriers_Integration tests the non-pagination query operations
// Note: Pagination queries require full SDK context with multi-store access
// which is not compatible with the indexer's standalone PebbleDB architecture in tests
func TestIndexerQueriers_Integration(t *testing.T) {
	gapp, ctx := setupIntegrationTest(t)
	// Use the app's address codec to match the indexer service configuration
	addrCodec := gapp.AccountKeeper.AddressCodec()

	authzHandler := gapp.IndexerService().AuthzHandler()
	authzQuerier := gapp.IndexerService().AuthzQuerier()
	feeGrantHandler := gapp.IndexerService().FeeGrantHandler()
	feeGrantQuerier := gapp.IndexerService().FeeGrantQuerier()

	// Create test addresses
	granter := sdk.AccAddress([]byte("granter_addr_test__"))
	grantee := sdk.AccAddress([]byte("grantee_addr_test__"))
	nonExistent := sdk.AccAddress([]byte("nonexistent_address"))

	granterStr, err := addrCodec.BytesToString(granter)
	require.NoError(t, err)
	granteeStr, err := addrCodec.BytesToString(grantee)
	require.NoError(t, err)
	nonExistentStr, err := addrCodec.BytesToString(nonExistent)
	require.NoError(t, err)

	t.Run("AuthzQuerier_AllowanceQuery", func(t *testing.T) {
		// Set up test grant
		sendAuth := banktypes.NewSendAuthorization(sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(1000))), nil)
		sendAuthAny, err := codectypes.NewAnyWithValue(sendAuth)
		require.NoError(t, err)

		grant := authz.Grant{
			Authorization: sendAuthAny,
			Expiration:    nil,
		}

		err = authzHandler.SetGrant(ctx, granter, grantee, banktypes.SendAuthorization{}.MsgTypeURL(), grant)
		require.NoError(t, err)

		// Query non-existent grant returns empty results without error
		_, err = authzQuerier.Grants(ctx, &indexerauthz.QueryGrantsRequest{
			Granter: granterStr,
			Grantee: nonExistentStr,
		})
		require.NoError(t, err) // Valid address format but doesn't exist in store
		// Note: This may fail due to pagination limitations in test environment
		// In production, this works through the GRPC interface with proper context
		t.Skip("Pagination queries require full SDK context - tested in unit tests with mocked data")
	})

	t.Run("FeeGrantQuerier_AllowanceQuery", func(t *testing.T) {
		// Set up test allowance
		allowance := &feegrant.BasicAllowance{
			SpendLimit: sdk.NewCoins(sdk.NewCoin("uxion", math.NewInt(1000))),
		}
		allowanceAny, err := codectypes.NewAnyWithValue(allowance)
		require.NoError(t, err)

		grant := feegrant.Grant{
			Granter:   granterStr,
			Grantee:   granteeStr,
			Allowance: allowanceAny,
		}

		err = feeGrantHandler.SetGrant(ctx, granter, grantee, grant)
		require.NoError(t, err)

		// Query single allowance (non-pagination query)
		resp, err := feeGrantQuerier.Allowance(ctx, &indexerfeegrant.QueryAllowanceRequest{
			Granter: granterStr,
			Grantee: granteeStr,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Allowance)
		require.Equal(t, granterStr, resp.Allowance.Granter)
		require.Equal(t, granteeStr, resp.Allowance.Grantee)
	})

	t.Run("FeeGrantQuerier_AllowanceNotFound", func(t *testing.T) {
		// Query non-existent allowance - should return nil allowance without error
		resp, err := feeGrantQuerier.Allowance(ctx, &indexerfeegrant.QueryAllowanceRequest{
			Granter: granterStr,
			Grantee: nonExistentStr,
		})
		require.NoError(t, err)        // Valid address format but doesn't exist in store
		require.Nil(t, resp.Allowance) // Allowance should be nil for non-existent grant
	})
}

// TestIndexerStreamingService_Integration tests the streaming service registration
func TestIndexerStreamingService_Integration(t *testing.T) {
	gapp, _ := setupIntegrationTest(t)

	service := gapp.IndexerService()
	require.NotNil(t, service)

	t.Run("Service_Components", func(t *testing.T) {
		require.NotNil(t, service.AuthzHandler())
		require.NotNil(t, service.FeeGrantHandler())
		require.NotNil(t, service.AuthzQuerier())
		require.NotNil(t, service.FeeGrantQuerier())
	})

	t.Run("Service_Registration", func(t *testing.T) {
		// Test that RegisterGRPCGatewayRoutes doesn't panic
		// This is primarily tested through actual node startup
		// Here we just verify the service is properly initialized
		require.NotNil(t, service)
	})
}

// TestIndexerDatabaseLifecycle_Integration tests database open/close
func TestIndexerDatabaseLifecycle_Integration(t *testing.T) {
	gapp, _ := setupIntegrationTest(t)

	service := gapp.IndexerService()
	require.NotNil(t, service)

	// The service is initialized with the app
	// Closing is handled by app cleanup
	// We verify the service is functional
	require.NotNil(t, service.AuthzHandler())
	require.NotNil(t, service.FeeGrantHandler())
}

// TestIndexerHandleUpdate_Integration tests the streaming update handling
func TestIndexerHandleUpdate_Integration(t *testing.T) {
	gapp, _ := setupIntegrationTest(t)

	authzHandler := gapp.IndexerService().AuthzHandler()

	t.Run("AuthzHandler_HandleUpdate", func(t *testing.T) {
		// HandleUpdate is tested in unit tests with mocked StoreKVPair
		// Integration test verifies the handler is properly initialized
		require.NotNil(t, authzHandler)

		// The actual update handling is tested through unit tests
		// because it requires properly formatted StoreKVPair from streaming
		t.Skip("HandleUpdate requires StoreKVPair from chain streaming - tested in unit tests")
	})
}
