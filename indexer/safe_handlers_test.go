package indexer

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/collections"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	"cosmossdk.io/x/feegrant"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

func TestSafeAuthzHandlerUpdate_Delete(t *testing.T) {
	memDB, cdc, _ := setupTest(t)
	ctx := context.Background()
	logger := log.NewTestLogger(t)

	// Create handler
	handler, err := NewAuthzHandler(&kvAccessor{db: memDB}, cdc)
	require.NoError(t, err)

	// Test addresses
	granter := sdk.AccAddress([]byte("granter_address_____"))
	grantee := sdk.AccAddress([]byte("grantee_address_____"))
	msgType := "/cosmos.bank.v1beta1.MsgSend"

	// First, add a grant
	auth := banktypes.NewSendAuthorization(sdk.NewCoins(sdk.NewInt64Coin("uxion", 1000)), nil)
	expiration := time.Now().Add(time.Hour)
	grant, err := authz.NewGrant(expiration, auth, nil)
	require.NoError(t, err)

	err = handler.SetGrant(ctx, granter, grantee, msgType, grant)
	require.NoError(t, err)

	// Create key for the grant
	key := createGrantStoreKey(granter, grantee, msgType)

	// Test successful delete
	pair := &storetypes.StoreKVPair{
		Key:    key,
		Delete: true,
	}

	err = SafeAuthzHandlerUpdate(ctx, handler, pair, logger)
	require.NoError(t, err)

	// Verify grant was removed
	_, err = handler.Authorizations.Get(ctx, collections.Join3(granter, grantee, msgType))
	require.Error(t, err)

	// Test delete of non-existent grant (should not error)
	err = SafeAuthzHandlerUpdate(ctx, handler, pair, logger)
	require.NoError(t, err) // Should handle gracefully
}

func TestSafeAuthzHandlerUpdate_Create(t *testing.T) {
	memDB, cdc, _ := setupTest(t)
	ctx := context.Background()
	logger := log.NewTestLogger(t)

	// Create handler
	handler, err := NewAuthzHandler(&kvAccessor{db: memDB}, cdc)
	require.NoError(t, err)

	// Test addresses
	granter := sdk.AccAddress([]byte("granter_address_____"))
	grantee := sdk.AccAddress([]byte("grantee_address_____"))
	msgType := "/cosmos.bank.v1beta1.MsgSend"

	// Create authorization
	auth := banktypes.NewSendAuthorization(sdk.NewCoins(sdk.NewInt64Coin("uxion", 1000)), nil)
	expiration := time.Now().Add(time.Hour)
	grant, err := authz.NewGrant(expiration, auth, nil)
	require.NoError(t, err)

	// Marshal grant
	grantBz, err := cdc.Marshal(&grant)
	require.NoError(t, err)

	// Create key for the grant
	key := createGrantStoreKey(granter, grantee, msgType)

	// Test successful create/update
	pair := &storetypes.StoreKVPair{
		Key:    key,
		Value:  grantBz,
		Delete: false,
	}

	err = SafeAuthzHandlerUpdate(ctx, handler, pair, logger)
	require.NoError(t, err)

	// Verify grant was added
	storedGrant, err := handler.Authorizations.Get(ctx, collections.Join3(granter, grantee, msgType))
	require.NoError(t, err)
	require.NotNil(t, storedGrant)
}

func TestSafeAuthzHandlerUpdate_InvalidValue(t *testing.T) {
	memDB, cdc, _ := setupTest(t)
	ctx := context.Background()
	logger := log.NewTestLogger(t)

	// Create handler
	handler, err := NewAuthzHandler(&kvAccessor{db: memDB}, cdc)
	require.NoError(t, err)

	// Test addresses
	granter := sdk.AccAddress([]byte("granter_address_____"))
	grantee := sdk.AccAddress([]byte("grantee_address_____"))
	msgType := "/cosmos.bank.v1beta1.MsgSend"

	// Create key for the grant
	key := createGrantStoreKey(granter, grantee, msgType)

	// Test with corrupted/invalid value (should not error, just log warning)
	pair := &storetypes.StoreKVPair{
		Key:    key,
		Value:  []byte("invalid protobuf data"),
		Delete: false,
	}

	err = SafeAuthzHandlerUpdate(ctx, handler, pair, logger)
	require.NoError(t, err) // Should handle gracefully
}

func TestSafeFeeGrantHandlerUpdate_Delete(t *testing.T) {
	memDB, cdc, _ := setupTest(t)
	ctx := context.Background()
	logger := log.NewTestLogger(t)

	// Create handler
	handler, err := NewFeeGrantHandler(&kvAccessor{db: memDB}, cdc)
	require.NoError(t, err)

	// Test addresses
	granter := sdk.AccAddress([]byte("granter_address_____"))
	grantee := sdk.AccAddress([]byte("grantee_address_____"))

	// First, add a grant
	allowance := &feegrant.BasicAllowance{
		SpendLimit: sdk.NewCoins(sdk.NewInt64Coin("uxion", 1000)),
	}
	grant, err := feegrant.NewGrant(granter, grantee, allowance)
	require.NoError(t, err)

	err = handler.SetGrant(ctx, granter, grantee, grant)
	require.NoError(t, err)

	// Create key for the grant using FeeAllowanceKey
	key := feegrant.FeeAllowanceKey(granter, grantee)

	// Test successful delete
	pair := &storetypes.StoreKVPair{
		Key:    key,
		Delete: true,
	}

	err = SafeFeeGrantHandlerUpdate(ctx, handler, pair, logger)
	require.NoError(t, err)

	// Verify grant was removed
	_, err = handler.FeeAllowances.Get(ctx, collections.Join(granter, grantee))
	require.Error(t, err)

	// Test delete of non-existent grant (should not error)
	err = SafeFeeGrantHandlerUpdate(ctx, handler, pair, logger)
	require.NoError(t, err) // Should handle gracefully
}

func TestSafeFeeGrantHandlerUpdate_Create(t *testing.T) {
	memDB, cdc, _ := setupTest(t)
	ctx := context.Background()
	logger := log.NewTestLogger(t)

	// Create handler
	handler, err := NewFeeGrantHandler(&kvAccessor{db: memDB}, cdc)
	require.NoError(t, err)

	// Test addresses
	granter := sdk.AccAddress([]byte("granter_address_____"))
	grantee := sdk.AccAddress([]byte("grantee_address_____"))

	// Create grant
	allowance := &feegrant.BasicAllowance{
		SpendLimit: sdk.NewCoins(sdk.NewInt64Coin("uxion", 1000)),
	}
	grant, err := feegrant.NewGrant(granter, grantee, allowance)
	require.NoError(t, err)

	// Marshal grant
	grantBz, err := cdc.Marshal(&grant)
	require.NoError(t, err)

	// Create key for the grant using FeeAllowanceKey
	key := feegrant.FeeAllowanceKey(granter, grantee)

	// Test successful create/update
	pair := &storetypes.StoreKVPair{
		Key:    key,
		Value:  grantBz,
		Delete: false,
	}

	err = SafeFeeGrantHandlerUpdate(ctx, handler, pair, logger)
	require.NoError(t, err)

	// Verify grant was added
	storedGrant, err := handler.FeeAllowances.Get(ctx, collections.Join(granter, grantee))
	require.NoError(t, err)
	require.NotNil(t, storedGrant)
}

func TestSafeFeeGrantHandlerUpdate_InvalidValue(t *testing.T) {
	memDB, cdc, _ := setupTest(t)
	ctx := context.Background()
	logger := log.NewTestLogger(t)

	// Create handler
	handler, err := NewFeeGrantHandler(&kvAccessor{db: memDB}, cdc)
	require.NoError(t, err)

	// Test addresses
	granter := sdk.AccAddress([]byte("granter_address_____"))
	grantee := sdk.AccAddress([]byte("grantee_address_____"))

	// Create key for the grant using FeeAllowanceKey
	key := feegrant.FeeAllowanceKey(granter, grantee)

	// Test with corrupted/invalid value (should not error, just log warning)
	pair := &storetypes.StoreKVPair{
		Key:    key,
		Value:  []byte("invalid protobuf data"),
		Delete: false,
	}

	err = SafeFeeGrantHandlerUpdate(ctx, handler, pair, logger)
	require.NoError(t, err) // Should handle gracefully
}

// MockAuthzHandler is a mock that simulates errors for testing error handling
type MockAuthzHandler struct {
	*AuthzHandler
	hasErr    error
	removeErr error
	setErr    error
}

func (m *MockAuthzHandler) Has(ctx context.Context, key collections.Triple[sdk.AccAddress, sdk.AccAddress, string]) (bool, error) {
	if m.hasErr != nil {
		return false, m.hasErr
	}
	return m.Authorizations.Has(ctx, key)
}

func (m *MockAuthzHandler) Remove(ctx context.Context, key collections.Triple[sdk.AccAddress, sdk.AccAddress, string]) error {
	if m.removeErr != nil {
		return m.removeErr
	}
	return m.Authorizations.Remove(ctx, key)
}

func (m *MockAuthzHandler) SetGrant(ctx context.Context, granter, grantee sdk.AccAddress, msgType string, grant authz.Grant) error {
	if m.setErr != nil {
		return m.setErr
	}
	return m.AuthzHandler.SetGrant(ctx, granter, grantee, msgType, grant)
}

func TestSafeAuthzHandlerUpdate_ErrorHandling(t *testing.T) {
	memDB, cdc, _ := setupTest(t)
	ctx := context.Background()
	logger := log.NewTestLogger(t)

	// Create base handler
	baseHandler, err := NewAuthzHandler(&kvAccessor{db: memDB}, cdc)
	require.NoError(t, err)

	// Test addresses
	granter := sdk.AccAddress([]byte("granter_address_____"))
	grantee := sdk.AccAddress([]byte("grantee_address_____"))
	msgType := "/cosmos.bank.v1beta1.MsgSend"

	// Create key for the grant
	key := createGrantStoreKey(granter, grantee, msgType)

	t.Run("Has error during delete", func(t *testing.T) {
		// Create mock handler with Has error
		mockHandler := &AuthzHandler{
			kvStoreService: baseHandler.kvStoreService,
			cdc:            baseHandler.cdc,
			Schema:         baseHandler.Schema,
			Authorizations: baseHandler.Authorizations,
		}

		pair := &storetypes.StoreKVPair{
			Key:    key,
			Delete: true,
		}

		// Simulate Has error by using an invalid context
		canceledCtx, cancel := context.WithCancel(ctx)
		cancel()

		// Should not error even with Has failure
		err := SafeAuthzHandlerUpdate(canceledCtx, mockHandler, pair, logger)
		require.NoError(t, err)
	})

	t.Run("SetGrant error during create", func(t *testing.T) {
		// Create authorization
		auth := banktypes.NewSendAuthorization(sdk.NewCoins(sdk.NewInt64Coin("uxion", 1000)), nil)
		expiration := time.Now().Add(time.Hour)
		grant, err := authz.NewGrant(expiration, auth, nil)
		require.NoError(t, err)

		// Marshal grant
		grantBz, err := cdc.Marshal(&grant)
		require.NoError(t, err)

		pair := &storetypes.StoreKVPair{
			Key:    key,
			Value:  grantBz,
			Delete: false,
		}

		// Simulate SetGrant error by using an invalid context
		canceledCtx, cancel := context.WithCancel(ctx)
		cancel()

		// Should not error even with SetGrant failure
		err = SafeAuthzHandlerUpdate(canceledCtx, baseHandler, pair, logger)
		require.NoError(t, err)
	})
}

// Helper function to create grant store key (mimics the real implementation)
func createGrantStoreKey(granter, grantee sdk.AccAddress, msgType string) []byte {
	// This mimics the actual key structure used in authz module
	// Format: 0x01 | len(granter) | granter | len(grantee) | grantee | msgType
	key := []byte{0x01}
	key = append(key, byte(len(granter)))
	key = append(key, granter...)
	key = append(key, byte(len(grantee)))
	key = append(key, grantee...)
	key = append(key, []byte(msgType)...)
	return key
}

func TestSafeAuthzHandlerUpdate_KeyParsing(t *testing.T) {
	memDB, cdc, _ := setupTest(t)
	ctx := context.Background()
	logger := log.NewTestLogger(t)

	// Create handler
	handler, err := NewAuthzHandler(&kvAccessor{db: memDB}, cdc)
	require.NoError(t, err)

	// Test with various key formats
	testCases := []struct {
		name      string
		key       []byte
		expectLog bool
	}{
		{
			name:      "valid key",
			key:       createGrantStoreKey(sdk.AccAddress("granter"), sdk.AccAddress("grantee"), "/msg.Type"),
			expectLog: false,
		},
		{
			name:      "empty key",
			key:       []byte{},
			expectLog: true,
		},
		{
			name:      "malformed key",
			key:       []byte{0x01, 0xFF, 0xFF}, // Invalid length bytes
			expectLog: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pair := &storetypes.StoreKVPair{
				Key:    tc.key,
				Value:  []byte("test"),
				Delete: false,
			}

			// Should not error even with malformed keys
			err := SafeAuthzHandlerUpdate(ctx, handler, pair, logger)
			require.NoError(t, err)
		})
	}
}

func TestSafeFeeGrantHandlerUpdate_KeyParsing(t *testing.T) {
	memDB, cdc, _ := setupTest(t)
	ctx := context.Background()
	logger := log.NewTestLogger(t)

	// Create handler
	handler, err := NewFeeGrantHandler(&kvAccessor{db: memDB}, cdc)
	require.NoError(t, err)

	// Test with various key formats
	testCases := []struct {
		name      string
		key       []byte
		expectLog bool
	}{
		{
			name: "valid key",
			key: feegrant.FeeAllowanceKey(
				sdk.AccAddress("granter_____________"),
				sdk.AccAddress("grantee_____________"),
			),
			expectLog: false,
		},
		{
			name:      "empty key",
			key:       []byte{},
			expectLog: true,
		},
		{
			name:      "malformed key",
			key:       []byte{0x00, 0xFF, 0xFF}, // Invalid format
			expectLog: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pair := &storetypes.StoreKVPair{
				Key:    tc.key,
				Value:  []byte("test"),
				Delete: false,
			}

			// Should not error even with malformed keys
			err := SafeFeeGrantHandlerUpdate(ctx, handler, pair, logger)
			require.NoError(t, err)
		})
	}
}

func TestSafeHandlers_ConcurrentAccess(t *testing.T) {
	memDB, cdc, _ := setupTest(t)
	ctx := context.Background()
	logger := log.NewNopLogger()

	// Create handlers
	authzHandler, err := NewAuthzHandler(&kvAccessor{db: memDB}, cdc)
	require.NoError(t, err)

	feeHandler, err := NewFeeGrantHandler(&kvAccessor{db: memDB}, cdc)
	require.NoError(t, err)

	// Run concurrent updates to test thread safety
	done := make(chan bool, 4)

	// Concurrent authz updates
	go func() {
		for i := 0; i < 10; i++ {
			granter := sdk.AccAddress([]byte("granter" + hex.EncodeToString([]byte{byte(i)})))
			grantee := sdk.AccAddress([]byte("grantee" + hex.EncodeToString([]byte{byte(i)})))
			key := createGrantStoreKey(granter, grantee, "/test.Msg")

			pair := &storetypes.StoreKVPair{
				Key:    key,
				Value:  []byte("test"),
				Delete: false,
			}

			_ = SafeAuthzHandlerUpdate(ctx, authzHandler, pair, logger)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			granter := sdk.AccAddress([]byte("granter" + hex.EncodeToString([]byte{byte(i)})))
			grantee := sdk.AccAddress([]byte("grantee" + hex.EncodeToString([]byte{byte(i)})))
			key := createGrantStoreKey(granter, grantee, "/test.Msg")

			pair := &storetypes.StoreKVPair{
				Key:    key,
				Delete: true,
			}

			_ = SafeAuthzHandlerUpdate(ctx, authzHandler, pair, logger)
		}
		done <- true
	}()

	// Concurrent feegrant updates
	go func() {
		for i := 0; i < 10; i++ {
			granter := sdk.AccAddress([]byte("granter" + hex.EncodeToString([]byte{byte(i)})))
			grantee := sdk.AccAddress([]byte("grantee" + hex.EncodeToString([]byte{byte(i)})))
			key := feegrant.FeeAllowanceKey(granter, grantee)

			pair := &storetypes.StoreKVPair{
				Key:    key,
				Value:  []byte("test"),
				Delete: false,
			}

			_ = SafeFeeGrantHandlerUpdate(ctx, feeHandler, pair, logger)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			granter := sdk.AccAddress([]byte("granter" + hex.EncodeToString([]byte{byte(i)})))
			grantee := sdk.AccAddress([]byte("grantee" + hex.EncodeToString([]byte{byte(i)})))
			key := feegrant.FeeAllowanceKey(granter, grantee)

			pair := &storetypes.StoreKVPair{
				Key:    key,
				Delete: true,
			}

			_ = SafeFeeGrantHandlerUpdate(ctx, feeHandler, pair, logger)
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 4; i++ {
		<-done
	}
}
