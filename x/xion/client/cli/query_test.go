package cli_test

import (
	"context"
	"io"
	"testing"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/burnt-labs/xion/x/xion/client/cli"
)

func (s *CLITestSuite) TestGetQueryCmd() {
	cmd := cli.GetQueryCmd()
	require.NotNil(s.T(), cmd)
	require.Equal(s.T(), "xion", cmd.Use)
	require.Equal(s.T(), "Querying commands for the xion module", cmd.Short)

	// Check that it has subcommands
	require.True(s.T(), cmd.HasSubCommands())
}

func (s *CLITestSuite) TestCmdPlatformPercentage() {
	cmd := cli.CmdPlatformPercentage()
	require.NotNil(s.T(), cmd)
	require.Equal(s.T(), "platform-percentage", cmd.Use)
	require.Equal(s.T(), "Get Platform Percentage", cmd.Short)

	// Test that command has default args validator (which allows any args)
	require.Nil(s.T(), cmd.Args)
}

func (s *CLITestSuite) TestCmdPlatformMinimum() {
	cmd := cli.CmdPlatformMinimum()
	require.NotNil(s.T(), cmd)
	require.Equal(s.T(), "platform-minimum", cmd.Use)
	require.Equal(s.T(), "Get Platform Minimum", cmd.Short)

	// Test that command has default args validator (which allows any args)
	require.Nil(s.T(), cmd.Args)
}

// TestQueryCommandsFullCoverage tests the remaining coverage gaps in query commands
func TestQueryCommandsFullCoverage(t *testing.T) {
	testCases := []struct {
		name    string
		cmdFunc func() *cobra.Command
		args    []string
		setup   func(*cobra.Command)
	}{
		{
			name:    "platform-percentage with actual query call",
			cmdFunc: cli.CmdPlatformPercentage,
			args:    []string{},
			setup: func(cmd *cobra.Command) {
				// Set up flags that will be used in the query
				cmd.Flags().Set(flags.FlagOutput, "json")
				cmd.Flags().Set(flags.FlagHeight, "0")
			},
		},
		{
			name:    "platform-minimum with actual query call",
			cmdFunc: cli.CmdPlatformMinimum,
			args:    []string{},
			setup: func(cmd *cobra.Command) {
				// Set up flags that will be used in the query
				cmd.Flags().Set(flags.FlagOutput, "text")
				cmd.Flags().Set(flags.FlagHeight, "1")
			},
		},
		{
			name:    "webauthn-register with query setup",
			cmdFunc: cli.CmdWebAuthNVerifyRegister,
			args:    []string{"xion1addr", "challenge123", "localhost", "testdata"},
			setup: func(cmd *cobra.Command) {
				cmd.Flags().Set(flags.FlagOutput, "json")
			},
		},
		{
			name:    "webauthn-authenticate with query setup",
			cmdFunc: cli.CmdWebAuthNVerifyAuthenticate,
			args:    []string{"xion1addr", "challenge123", "localhost", "credential", "testdata"},
			setup: func(cmd *cobra.Command) {
				cmd.Flags().Set(flags.FlagOutput, "text")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := tc.cmdFunc()
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs(tc.args)

			// Setup flags
			if tc.setup != nil {
				tc.setup(cmd)
			}

			ctx := svrcmd.CreateExecuteContext(context.Background())
			cmd.SetContext(ctx)

			// Set up a basic client context to exercise more paths
			baseCtx := client.Context{}
			err := client.SetCmdClientContextHandler(baseCtx, cmd)
			require.NoError(t, err)

			// This will likely error due to no actual server, but will exercise more code paths
			err = cmd.Execute()
			require.Error(t, err) // Expected to error without real server/setup
		})
	}
}

// TestQueryCommandsWithDifferentContexts tests query commands with various context setups
func TestQueryCommandsWithDifferentContexts(t *testing.T) {
	t.Run("platform_percentage_with_empty_context", func(t *testing.T) {
		cmd := cli.CmdPlatformPercentage()
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)

		// Test with context but no client setup
		ctx := svrcmd.CreateExecuteContext(context.Background())
		cmd.SetContext(ctx)

		err := cmd.RunE(cmd, []string{})
		require.Error(t, err) // Should error when trying to get client context
	})

	t.Run("platform_minimum_with_empty_context", func(t *testing.T) {
		cmd := cli.CmdPlatformMinimum()
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)

		// Test with context but no client setup
		ctx := svrcmd.CreateExecuteContext(context.Background())
		cmd.SetContext(ctx)

		err := cmd.RunE(cmd, []string{})
		require.Error(t, err) // Should error when trying to get client context
	})

	t.Run("webauthn_register_with_empty_context", func(t *testing.T) {
		cmd := cli.CmdWebAuthNVerifyRegister()
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)

		// Test with context but no client setup
		ctx := svrcmd.CreateExecuteContext(context.Background())
		cmd.SetContext(ctx)

		err := cmd.RunE(cmd, []string{"addr", "challenge", "rp", "data"})
		require.Error(t, err) // Should error when trying to get client context
	})

	t.Run("webauthn_authenticate_with_empty_context", func(t *testing.T) {
		cmd := cli.CmdWebAuthNVerifyAuthenticate()
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)

		// Test with context but no client setup
		ctx := svrcmd.CreateExecuteContext(context.Background())
		cmd.SetContext(ctx)

		err := cmd.RunE(cmd, []string{"addr", "challenge", "rp", "credential", "data"})
		require.Error(t, err) // Should error when trying to get client context
	})
}

// TestQueryCommandArgParsing tests argument parsing in query commands
func TestQueryCommandArgParsing(t *testing.T) {
	t.Run("webauthn_register_args_slice_access", func(t *testing.T) {
		cmd := cli.CmdWebAuthNVerifyRegister()
		cmd.SetContext(context.Background())

		// Test that args are properly accessed in RunE
		err := cmd.RunE(cmd, []string{"addr", "challenge", "rp", "data"})
		require.Error(t, err) // Will error on client context but should access all args
	})

	t.Run("webauthn_authenticate_args_slice_access", func(t *testing.T) {
		cmd := cli.CmdWebAuthNVerifyAuthenticate()
		cmd.SetContext(context.Background())

		// Test that args are properly accessed in RunE
		err := cmd.RunE(cmd, []string{"addr", "challenge", "rp", "credential", "data"})
		require.Error(t, err) // Will error on client context but should access all args
	})
}
