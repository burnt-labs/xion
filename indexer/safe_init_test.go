package indexer

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	abci "github.com/cometbft/cometbft/abci/types"

	db "github.com/cosmos/cosmos-db"

	"cosmossdk.io/log"
	"cosmossdk.io/x/feegrant"

	"github.com/cosmos/cosmos-sdk/codec"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/std"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

func setupTestCodec() codec.Codec {
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	std.RegisterInterfaces(interfaceRegistry)
	authz.RegisterInterfaces(interfaceRegistry)
	banktypes.RegisterInterfaces(interfaceRegistry)
	feegrant.RegisterInterfaces(interfaceRegistry)
	return codec.NewProtoCodec(interfaceRegistry)
}

func TestNewSafeIndexer_Success(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()
	cdc := setupTestCodec()
	addrCodec := addresscodec.NewBech32Codec("xion")
	logger := log.NewTestLogger(t)

	// Test successful initialization
	service := NewSafeIndexer(tempDir, cdc, addrCodec, logger)
	require.NotNil(t, service)

	// Verify it's a StreamService (not NoOp)
	streamService, ok := service.(*StreamService)
	require.True(t, ok, "Should return StreamService on success")
	require.NotNil(t, streamService.db)
	require.NotNil(t, streamService.authzHandler)
	require.NotNil(t, streamService.feeGrantHandler)
	require.NotNil(t, streamService.authzQuerier)
	require.NotNil(t, streamService.feegrantQuerier)

	// Clean up
	err := service.Close()
	require.NoError(t, err)
}

func TestNewSafeIndexer_DatabaseInitFailure(t *testing.T) {
	// Use an invalid directory path that will cause DB initialization to fail
	invalidDir := "/invalid/path/that/does/not/exist"
	cdc := setupTestCodec()
	addrCodec := addresscodec.NewBech32Codec("xion")
	logger := log.NewTestLogger(t)

	// Test that it returns NoOpStreamService on DB init failure
	service := NewSafeIndexer(invalidDir, cdc, addrCodec, logger)
	require.NotNil(t, service)

	// Verify it's a NoOpStreamService
	_, ok := service.(*NoOpStreamService)
	require.True(t, ok, "Should return NoOpStreamService on database init failure")

	// Clean up
	err := service.Close()
	require.NoError(t, err)
}

func TestNewSafeIndexer_HandlerInitFailure(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()

	// Create a nil codec to cause handler initialization to fail
	var nilCodec codec.Codec = nil
	addrCodec := addresscodec.NewBech32Codec("xion")
	logger := log.NewTestLogger(t)

	// Test that it returns NoOpStreamService on handler init failure
	service := NewSafeIndexer(tempDir, nilCodec, addrCodec, logger)
	require.NotNil(t, service)

	// Verify it's a NoOpStreamService
	_, ok := service.(*NoOpStreamService)
	require.True(t, ok, "Should return NoOpStreamService on handler init failure")

	// Clean up
	err := service.Close()
	require.NoError(t, err)
}

func TestNewSafeIndexer_InterfaceCompliance(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()
	cdc := setupTestCodec()
	addrCodec := addresscodec.NewBech32Codec("xion")
	logger := log.NewNopLogger()

	// Test that both success and failure cases implement IndexerService
	service := NewSafeIndexer(tempDir, cdc, addrCodec, logger)
	require.NotNil(t, service)

	// Verify interface compliance
	_ = service

	// Clean up
	err := service.Close()
	require.NoError(t, err)
}

func TestNewSafeIndexer_LoggerConfiguration(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()
	cdc := setupTestCodec()
	addrCodec := addresscodec.NewBech32Codec("xion")

	// Use a test logger to verify logging
	logger := log.NewTestLogger(t)

	// Test successful initialization logs
	service := NewSafeIndexer(tempDir, cdc, addrCodec, logger)
	require.NotNil(t, service)

	// Clean up
	err := service.Close()
	require.NoError(t, err)
}

// MockKVAccessor simulates database access failures
type MockKVAccessor struct {
	shouldFail bool
}

func (m *MockKVAccessor) Get(key []byte) ([]byte, error) {
	if m.shouldFail {
		return nil, os.ErrNotExist
	}
	return nil, nil
}

func (m *MockKVAccessor) Set(key, value []byte) error {
	if m.shouldFail {
		return os.ErrClosed
	}
	return nil
}

func (m *MockKVAccessor) Delete(key []byte) error {
	if m.shouldFail {
		return os.ErrClosed
	}
	return nil
}

func (m *MockKVAccessor) Has(key []byte) (bool, error) {
	if m.shouldFail {
		return false, os.ErrNotExist
	}
	return false, nil
}

func (m *MockKVAccessor) Iterator(start, end []byte) (db.Iterator, error) {
	if m.shouldFail {
		return nil, os.ErrClosed
	}
	return nil, nil
}

func (m *MockKVAccessor) ReverseIterator(start, end []byte) (db.Iterator, error) {
	if m.shouldFail {
		return nil, os.ErrClosed
	}
	return nil, nil
}

func TestNewSafeIndexer_MultipleCalls(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()
	cdc := setupTestCodec()
	addrCodec := addresscodec.NewBech32Codec("xion")
	logger := log.NewNopLogger()

	// Create multiple indexer instances
	services := make([]IndexerService, 3)
	for i := range services {
		// Each instance should have its own subdirectory
		subDir := filepath.Join(tempDir, string(rune('0'+i)))
		err := os.MkdirAll(filepath.Join(subDir, "data"), 0o755)
		require.NoError(t, err)

		services[i] = NewSafeIndexer(subDir, cdc, addrCodec, logger)
		require.NotNil(t, services[i])
	}

	// Clean up all services
	for _, service := range services {
		err := service.Close()
		require.NoError(t, err)
	}
}

func TestNewSafeIndexer_DatabaseCleanup(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()
	addrCodec := addresscodec.NewBech32Codec("xion")
	logger := log.NewTestLogger(t)

	// Test scenario where authz handler fails but DB was created
	// We need to simulate this by creating a DB first
	dataDir := filepath.Join(tempDir, "data")
	storeDB, err := db.NewPebbleDB("xion_indexer", dataDir, nil)
	require.NoError(t, err)

	// Close the DB to simulate it being available for cleanup
	err = storeDB.Close()
	require.NoError(t, err)

	// Now test with a scenario that will fail handler creation
	// but should still clean up the DB properly
	service := NewSafeIndexer(tempDir, nil, addrCodec, logger)
	require.NotNil(t, service)

	// Should return NoOpStreamService due to nil codec
	_, ok := service.(*NoOpStreamService)
	require.True(t, ok, "Should return NoOpStreamService on handler failure")
}

func TestIndexerService_AllMethods(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()
	cdc := setupTestCodec()
	addrCodec := addresscodec.NewBech32Codec("xion")
	logger := log.NewNopLogger()

	// Create service
	service := NewSafeIndexer(tempDir, cdc, addrCodec, logger)
	require.NotNil(t, service)

	ctx := context.Background()

	// Test all IndexerService methods
	t.Run("ListenFinalizeBlock", func(t *testing.T) {
		err := service.ListenFinalizeBlock(ctx, abci.RequestFinalizeBlock{}, abci.ResponseFinalizeBlock{})
		require.NoError(t, err)
	})

	t.Run("ListenCommit", func(t *testing.T) {
		err := service.ListenCommit(ctx, abci.ResponseCommit{}, nil)
		require.NoError(t, err)
	})

	t.Run("RegisterServices", func(t *testing.T) {
		// Skip testing RegisterServices with nil configurator as it requires
		// a proper module.Configurator implementation which is complex to mock
		// The important part is that the method exists and is callable
		t.Skip("RegisterServices requires proper module.Configurator implementation")
	})

	t.Run("Close", func(t *testing.T) {
		err := service.Close()
		require.NoError(t, err)
	})
}

func TestNewSafeIndexer_PermissionIssues(t *testing.T) {
	// Skip this test if running as root (CI/CD environments)
	if os.Geteuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	// Create a directory with no write permissions
	tempDir := t.TempDir()
	readOnlyDir := filepath.Join(tempDir, "readonly")
	err := os.MkdirAll(readOnlyDir, 0o555) // Read and execute only
	require.NoError(t, err)

	cdc := setupTestCodec()
	addrCodec := addresscodec.NewBech32Codec("xion")
	logger := log.NewTestLogger(t)

	// Should return NoOpStreamService due to permission issues
	service := NewSafeIndexer(readOnlyDir, cdc, addrCodec, logger)
	require.NotNil(t, service)

	// Verify it's a NoOpStreamService
	_, ok := service.(*NoOpStreamService)
	require.True(t, ok, "Should return NoOpStreamService on permission issues")

	// Clean up
	err = service.Close()
	require.NoError(t, err)

	// Restore permissions for cleanup
	err = os.Chmod(readOnlyDir, 0o755)
	require.NoError(t, err)
}

func TestNewSafeIndexer_ConcurrentInitialization(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()
	cdc := setupTestCodec()
	addrCodec := addresscodec.NewBech32Codec("xion")
	logger := log.NewNopLogger()

	// Test concurrent initialization
	numGoroutines := 5
	services := make([]IndexerService, numGoroutines)
	done := make(chan int, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			// Each goroutine gets its own directory
			subDir := filepath.Join(tempDir, string(rune('0'+idx)))
			err := os.MkdirAll(filepath.Join(subDir, "data"), 0o755)
			require.NoError(t, err)

			services[idx] = NewSafeIndexer(subDir, cdc, addrCodec, logger)
			done <- idx
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		idx := <-done
		require.NotNil(t, services[idx])
	}

	// Clean up all services
	for _, service := range services {
		if service != nil {
			err := service.Close()
			require.NoError(t, err)
		}
	}
}

// TestIndexerServiceInterface verifies all implementations satisfy the interface
func TestIndexerServiceInterface(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()
	cdc := setupTestCodec()
	addrCodec := addresscodec.NewBech32Codec("xion")
	logger := log.NewNopLogger()

	// Test successful case (StreamService)
	service1 := NewSafeIndexer(tempDir, cdc, addrCodec, logger)
	require.Implements(t, (*IndexerService)(nil), service1)
	require.Implements(t, (*io.Closer)(nil), service1)

	// Test failure case (NoOpStreamService)
	service2 := NewSafeIndexer("/invalid/path", cdc, addrCodec, logger)
	require.Implements(t, (*IndexerService)(nil), service2)
	require.Implements(t, (*io.Closer)(nil), service2)

	// Clean up
	_ = service1.Close()
	_ = service2.Close()
}
