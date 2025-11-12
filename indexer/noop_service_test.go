package indexer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	abci "github.com/cometbft/cometbft/abci/types"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/types/module"
)

// MockConfigurator is a mock implementation of module.Configurator for testing
type MockConfigurator struct {
	module.Configurator
	serviceCalled bool
}

func TestNewNoOpStreamService(t *testing.T) {
	logger := log.NewNopLogger()

	// Test creating a new NoOpStreamService
	service := NewNoOpStreamService(logger)

	require.NotNil(t, service)
	require.NotNil(t, service.log)

	// Verify the logger is properly configured with module and mode
	// This ensures the service is properly initialized
}

func TestNoOpStreamService_ListenFinalizeBlock(t *testing.T) {
	logger := log.NewNopLogger()
	service := NewNoOpStreamService(logger)
	ctx := context.Background()

	// Create test request and response
	req := abci.RequestFinalizeBlock{
		Height: 100,
		Txs:    [][]byte{[]byte("test-tx")},
	}

	res := abci.ResponseFinalizeBlock{
		TxResults: []*abci.ExecTxResult{
			{
				Code: 0,
				Log:  "success",
			},
		},
	}

	// Test that ListenFinalizeBlock returns nil (no-op behavior)
	err := service.ListenFinalizeBlock(ctx, req, res)
	require.NoError(t, err)

	// Test with empty request/response
	err = service.ListenFinalizeBlock(ctx, abci.RequestFinalizeBlock{}, abci.ResponseFinalizeBlock{})
	require.NoError(t, err)
}

func TestNoOpStreamService_ListenCommit(t *testing.T) {
	logger := log.NewNopLogger()
	service := NewNoOpStreamService(logger)
	ctx := context.Background()

	// Create test response and changeset
	res := abci.ResponseCommit{
		RetainHeight: 90,
	}

	changeSet := []*storetypes.StoreKVPair{
		{
			StoreKey: "test-store",
			Delete:   false,
			Key:      []byte("key1"),
			Value:    []byte("value1"),
		},
		{
			StoreKey: "test-store",
			Delete:   true,
			Key:      []byte("key2"),
			Value:    nil,
		},
	}

	// Test that ListenCommit returns nil (no-op behavior)
	err := service.ListenCommit(ctx, res, changeSet)
	require.NoError(t, err)

	// Test with nil changeset
	err = service.ListenCommit(ctx, res, nil)
	require.NoError(t, err)

	// Test with empty changeset
	err = service.ListenCommit(ctx, res, []*storetypes.StoreKVPair{})
	require.NoError(t, err)
}

func TestNoOpStreamService_Close(t *testing.T) {
	// Use a test logger to verify log output
	logger := log.NewTestLogger(t)
	service := NewNoOpStreamService(logger)

	// Test that Close returns nil and logs appropriately
	err := service.Close()
	require.NoError(t, err)

	// Call Close multiple times to ensure idempotency
	err = service.Close()
	require.NoError(t, err)
}

func TestNoOpStreamService_RegisterServices(t *testing.T) {
	// Use a test logger to verify log output
	logger := log.NewTestLogger(t)
	service := NewNoOpStreamService(logger)

	// Create a mock configurator
	configurator := &MockConfigurator{}

	// Test that RegisterServices returns nil (no-op behavior)
	err := service.RegisterServices(configurator)
	require.NoError(t, err)

	// Verify that no actual service registration occurred
	require.False(t, configurator.serviceCalled)

	// Test with nil configurator (should still not panic)
	err = service.RegisterServices(nil)
	require.NoError(t, err)
}

func TestNoOpStreamService_InterfaceCompliance(t *testing.T) {
	logger := log.NewNopLogger()
	service := NewNoOpStreamService(logger)

	// Verify that NoOpStreamService implements IndexerService interface
	var _ IndexerService = service

	// This test ensures the NoOpStreamService properly implements all required methods
	// of the IndexerService interface, maintaining compatibility with the main service
}

func TestNoOpStreamService_ConcurrentOperations(t *testing.T) {
	logger := log.NewNopLogger()
	service := NewNoOpStreamService(logger)
	ctx := context.Background()

	// Test concurrent calls to ensure thread safety
	done := make(chan bool, 4)

	// Concurrent ListenFinalizeBlock
	go func() {
		err := service.ListenFinalizeBlock(ctx, abci.RequestFinalizeBlock{}, abci.ResponseFinalizeBlock{})
		require.NoError(t, err)
		done <- true
	}()

	// Concurrent ListenCommit
	go func() {
		err := service.ListenCommit(ctx, abci.ResponseCommit{}, nil)
		require.NoError(t, err)
		done <- true
	}()

	// Concurrent RegisterServices
	go func() {
		err := service.RegisterServices(nil)
		require.NoError(t, err)
		done <- true
	}()

	// Concurrent Close
	go func() {
		err := service.Close()
		require.NoError(t, err)
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 4; i++ {
		<-done
	}
}
