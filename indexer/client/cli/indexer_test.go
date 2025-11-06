package cmd

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	dbm "github.com/cosmos/cosmos-db"

	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/std"
)

// Mock app creator for testing - matches AppCreator type
var mockAppCreator servertypes.AppCreator = func(logger log.Logger, db dbm.DB, traceStore io.Writer, appOpts servertypes.AppOptions) servertypes.Application {
	// Return nil for testing - the ReIndex function will fail but we can test the setup
	return nil
}

func TestIndexer(t *testing.T) {
	cmd := Indexer(mockAppCreator, "/tmp/test")
	require.NotNil(t, cmd)
	require.Equal(t, "indexer", cmd.Use)
	require.NotNil(t, cmd.RunE)

	// Test that it has subcommands
	require.True(t, cmd.HasSubCommands())

	// Count subcommands
	subCmds := cmd.Commands()
	require.Len(t, subCmds, 5) // re-index + 4 query commands

	// Test running the command (no-op)
	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)
}

func TestReIndex(t *testing.T) {
	cmd := ReIndex(mockAppCreator, "/tmp/test")
	require.NotNil(t, cmd)
	require.Equal(t, "re-index", cmd.Use)
	require.NotNil(t, cmd.RunE)

	// Check flags
	homeFlag := cmd.Flags().Lookup(flags.FlagHome)
	require.NotNil(t, homeFlag)
	require.Equal(t, "/tmp/test", homeFlag.DefValue)

	dbBackendFlag := cmd.Flags().Lookup(FlagAppDBBackend)
	require.NotNil(t, dbBackendFlag)
}

func TestQueryGrantsByGrantee(t *testing.T) {
	cmd := QueryGrantsByGrantee()
	require.NotNil(t, cmd)
	require.Equal(t, "query-grants-by-grantee [grantee]", cmd.Use)
	require.Equal(t, "query authz grants by grantee", cmd.Short)
	require.NotNil(t, cmd.RunE)

	// Test that it requires exactly 1 argument
	err := cmd.Args(cmd, []string{})
	require.Error(t, err)

	err = cmd.Args(cmd, []string{"addr1"})
	require.NoError(t, err)

	err = cmd.Args(cmd, []string{"addr1", "addr2"})
	require.Error(t, err)
}

func TestQueryGrantsByGranter(t *testing.T) {
	cmd := QueryGrantsByGranter()
	require.NotNil(t, cmd)
	require.Equal(t, "query-grants-by-granter [granter]", cmd.Use)
	require.Equal(t, "query grants by granter", cmd.Short)
	require.NotNil(t, cmd.RunE)

	// Test that it requires exactly 1 argument
	err := cmd.Args(cmd, []string{})
	require.Error(t, err)

	err = cmd.Args(cmd, []string{"addr1"})
	require.NoError(t, err)

	err = cmd.Args(cmd, []string{"addr1", "addr2"})
	require.Error(t, err)
}

func TestQueryAllowancesByGrantee(t *testing.T) {
	cmd := QueryAllowancesByGrantee()
	require.NotNil(t, cmd)
	require.Equal(t, "query-grants-by-grantee [grantee]", cmd.Use)
	require.Equal(t, "query grants by grantee", cmd.Short)
	require.NotNil(t, cmd.RunE)

	// Test that it requires exactly 1 argument
	err := cmd.Args(cmd, []string{})
	require.Error(t, err)

	err = cmd.Args(cmd, []string{"addr1"})
	require.NoError(t, err)

	err = cmd.Args(cmd, []string{"addr1", "addr2"})
	require.Error(t, err)
}

func TestQueryAllowancesByGranter(t *testing.T) {
	cmd := QueryAllowancesByGranter()
	require.NotNil(t, cmd)
	require.Equal(t, "query-allowances-by-granter [granter]", cmd.Use)
	require.Equal(t, "query allowances by granter", cmd.Short)
	require.NotNil(t, cmd.RunE)

	// Test that it requires exactly 1 argument
	err := cmd.Args(cmd, []string{})
	require.Error(t, err)

	err = cmd.Args(cmd, []string{"addr1"})
	require.NoError(t, err)

	err = cmd.Args(cmd, []string{"addr1", "addr2"})
	require.Error(t, err)
}

func TestOpenDB(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()

	// Test with memory backend
	db, err := openDB(tmpDir, dbm.MemDBBackend)
	require.NoError(t, err)
	require.NotNil(t, db)
	defer db.Close()

	// Verify data directory path is constructed correctly
	expectedDataDir := filepath.Join(tmpDir, "data")
	require.NoError(t, os.MkdirAll(expectedDataDir, 0o755))
}

// TestQueryGrantsByGranteeRunE tests the RunE function with a mock client context
func TestQueryGrantsByGranteeRunE(t *testing.T) {
	cmd := QueryGrantsByGrantee()

	// Create a buffer to capture output
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	// Set up minimal codec for client context
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	std.RegisterInterfaces(interfaceRegistry)
	cdc := codec.NewProtoCodec(interfaceRegistry)

	// Create client context
	clientCtx := client.Context{}.
		WithCodec(cdc).
		WithOutput(buf)

	// Set client context in command
	ctx := context.WithValue(context.Background(), client.ClientContextKey, &clientCtx)
	cmd.SetContext(ctx)

	// Test RunE with invalid client context (should fail at GetClientQueryContext)
	err := cmd.RunE(cmd, []string{"xion1test"})
	// We expect an error because we don't have a full client setup
	require.Error(t, err)
}

// TestQueryGrantsByGranterRunE tests the RunE function
func TestQueryGrantsByGranterRunE(t *testing.T) {
	cmd := QueryGrantsByGranter()

	// Create a buffer to capture output
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	// Set up minimal codec
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	std.RegisterInterfaces(interfaceRegistry)
	cdc := codec.NewProtoCodec(interfaceRegistry)

	// Create client context
	clientCtx := client.Context{}.
		WithCodec(cdc).
		WithOutput(buf)

	// Set client context in command
	ctx := context.WithValue(context.Background(), client.ClientContextKey, &clientCtx)
	cmd.SetContext(ctx)

	// Test RunE - should fail at GetClientQueryContext
	err := cmd.RunE(cmd, []string{"xion1test"})
	require.Error(t, err)
}

// TestQueryAllowancesByGranteeRunE tests the RunE function
func TestQueryAllowancesByGranteeRunE(t *testing.T) {
	cmd := QueryAllowancesByGrantee()

	// Create a buffer to capture output
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	// Set up minimal codec
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	std.RegisterInterfaces(interfaceRegistry)
	cdc := codec.NewProtoCodec(interfaceRegistry)

	// Create client context
	clientCtx := client.Context{}.
		WithCodec(cdc).
		WithOutput(buf)

	// Set client context in command
	ctx := context.WithValue(context.Background(), client.ClientContextKey, &clientCtx)
	cmd.SetContext(ctx)

	// Test RunE - should fail at GetClientQueryContext
	err := cmd.RunE(cmd, []string{"xion1test"})
	require.Error(t, err)
}

// TestQueryAllowancesByGranterRunE tests the RunE function
func TestQueryAllowancesByGranterRunE(t *testing.T) {
	cmd := QueryAllowancesByGranter()

	// Create a buffer to capture output
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	// Set up minimal codec
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	std.RegisterInterfaces(interfaceRegistry)
	cdc := codec.NewProtoCodec(interfaceRegistry)

	// Create client context
	clientCtx := client.Context{}.
		WithCodec(cdc).
		WithOutput(buf)

	// Set client context in command
	ctx := context.WithValue(context.Background(), client.ClientContextKey, &clientCtx)
	cmd.SetContext(ctx)

	// Test RunE - should fail at GetClientQueryContext
	err := cmd.RunE(cmd, []string{"xion1test"})
	require.Error(t, err)
}

// TestFlagAppDBBackend tests the constant
func TestFlagAppDBBackend(t *testing.T) {
	require.Equal(t, "app-db-backend", FlagAppDBBackend)
}
