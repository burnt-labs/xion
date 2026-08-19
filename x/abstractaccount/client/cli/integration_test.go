package cli_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/burnt-labs/xion/x/abstractaccount/simapp"
	simapptesting "github.com/burnt-labs/xion/x/abstractaccount/simapp/testing"
	cli "github.com/burnt-labs/xion/x/abstractaccount/client/cli"
)

// TestQueryParamsIntegration tests the queryParams function with a real client context
func TestQueryParamsIntegration(t *testing.T) {
	// Create mock app with cleanup
	_, cleanup := simapptesting.MakeMockAppWithCleanup([]banktypes.Balance{})
	defer cleanup()

	// Create encoding config
	encCfg := simapp.MakeEncodingConfig()

	// Create client context
	clientCtx := client.Context{}.
		WithCodec(encCfg.Codec).
		WithTxConfig(encCfg.TxConfig).
		WithLegacyAmino(encCfg.Amino).
		WithInterfaceRegistry(encCfg.InterfaceRegistry).
		WithBroadcastMode(flags.BroadcastSync).
		WithHomeDir("/tmp").
		WithChainID("test-chain")

	t.Run("successful queryParams execution with client context", func(t *testing.T) {
		// Create the params command to test the actual function assignment
		paramsCmd, _, findErr := cli.GetQueryCmd().Find([]string{"params"})
		require.NoError(t, findErr)
		require.NotNil(t, paramsCmd.RunE)

		// Set proper context with client context
		paramsCmd.SetContext(context.WithValue(context.Background(), client.ClientContextKey, &clientCtx))

		// This will still fail because we need a running GRPC server, but it tests more of the execution path
		err := paramsCmd.RunE(paramsCmd, []string{})
		require.Error(t, err)
		// The error should be about connection/transport, not about missing client context
		require.Contains(t, err.Error(), "connection")
	})

	t.Run("queryParams with properly structured command", func(t *testing.T) {
		// Test the actual queryParams function indirectly through the command structure
		queryCmd := cli.GetQueryCmd()
		require.NotNil(t, queryCmd)

		subCommands := queryCmd.Commands()
		require.Len(t, subCommands, 2)

		paramsSubCmd, _, findErr := queryCmd.Find([]string{"params"})
		require.NoError(t, findErr)
		require.Equal(t, "params", paramsSubCmd.Use)
		require.NotNil(t, paramsSubCmd.RunE)

		// Set proper context
		paramsSubCmd.SetContext(context.WithValue(context.Background(), client.ClientContextKey, &clientCtx))

		// Execute with proper setup but expect connection error
		err := paramsSubCmd.RunE(paramsSubCmd, []string{})
		require.Error(t, err)
		// Should fail on connection, not on client context setup
		require.NotContains(t, err.Error(), "nil pointer")
	})
}

// TestQueryCmdStructureIntegration tests the full command structure with proper client context
func TestQueryCmdStructureIntegration(t *testing.T) {
	// Create mock app
	_, cleanup := simapptesting.MakeMockAppWithCleanup([]banktypes.Balance{})
	defer cleanup()

	// Test that the command structure works with proper client context
	queryCmd := cli.GetQueryCmd()
	require.NotNil(t, queryCmd)

	// Test that subcommands are created with proper structure
	subCommands := queryCmd.Commands()
	require.Len(t, subCommands, 2)

	paramsCmd, _, findErr := queryCmd.Find([]string{"params"})
	require.NoError(t, findErr)
	require.Equal(t, "params", paramsCmd.Use)
	require.Equal(t, "Query the module's parameters", paramsCmd.Short)
	require.NotNil(t, paramsCmd.RunE)
	require.NotNil(t, paramsCmd.Args)

	// Test Args validation
	err := paramsCmd.Args(paramsCmd, []string{})
	require.NoError(t, err)

	err = paramsCmd.Args(paramsCmd, []string{"extra"})
	require.Error(t, err)

	// Test flags are properly added
	flagSet := paramsCmd.Flags()
	outputFlag := flagSet.Lookup(flags.FlagOutput)
	require.NotNil(t, outputFlag)
}

// TestParamsCmdExecutionWithClientContext tests params command execution with client context
func TestParamsCmdExecutionWithClientContext(t *testing.T) {
	// Create mock app
	_, cleanup := simapptesting.MakeMockAppWithCleanup([]banktypes.Balance{})
	defer cleanup()

	// Create encoding config
	encCfg := simapp.MakeEncodingConfig()

	// Create client context
	clientCtx := client.Context{}.
		WithCodec(encCfg.Codec).
		WithTxConfig(encCfg.TxConfig).
		WithLegacyAmino(encCfg.Amino).
		WithInterfaceRegistry(encCfg.InterfaceRegistry).
		WithBroadcastMode(flags.BroadcastSync).
		WithHomeDir("/tmp").
		WithChainID("test-chain")

	// Get the params command
	queryCmd := cli.GetQueryCmd()
	paramsCmd, _, findErr := queryCmd.Find([]string{"params"})
	require.NoError(t, findErr)

	// Set proper context
	paramsCmd.SetContext(context.WithValue(context.Background(), client.ClientContextKey, &clientCtx))

	// Execute the command - this should get further in execution than before
	err := paramsCmd.RunE(paramsCmd, []string{})
	require.Error(t, err)
	// Should fail on gRPC connection, not on client context creation
	require.Contains(t, err.Error(), "connection")
}
