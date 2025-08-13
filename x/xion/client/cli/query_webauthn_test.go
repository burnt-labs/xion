package cli_test

import (
	"context"
	"io"

	"github.com/burnt-labs/xion/x/xion/client/cli"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// TestCmdWebAuthNVerifyRegister tests the webauthn verify register command structure
func (s *CLITestSuite) TestCmdWebAuthNVerifyRegister() {
	cmd := cli.CmdWebAuthNVerifyRegister()
	require.NotNil(s.T(), cmd)
	require.Equal(s.T(), "webauthn-register [addr] [challenge] [rp] [data]", cmd.Use)
	require.Equal(s.T(), "Test Webauthn Registration", cmd.Short)

	// Test that command requires exactly four arguments
	require.NotNil(s.T(), cmd.Args)

	// Test execution with no args (should return error about missing argument)
	err := cmd.Args(cmd, []string{})
	require.Error(s.T(), err)

	// Test execution with correct number of args (should pass)
	err = cmd.Args(cmd, []string{"addr", "challenge", "rp", "data"})
	require.NoError(s.T(), err)
}

// TestCmdWebAuthNVerifyRegisterExecution tests webauthn register command execution
func (s *CLITestSuite) TestCmdWebAuthNVerifyRegisterExecution() {
	testCases := []struct {
		name      string
		args      []string
		expectErr bool
	}{
		{
			"help flag execution",
			[]string{"--help"},
			false,
		},
		{
			"insufficient args",
			[]string{},
			true,
		},
		{
			"too many args",
			[]string{"arg1", "arg2", "arg3", "arg4", "arg5"},
			true,
		},
		{
			"with required args but no server",
			[]string{"addr", "challenge", "rp", "data"},
			true, // Should error because no actual server
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			cmd := cli.CmdWebAuthNVerifyRegister()
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)

			ctx := svrcmd.CreateExecuteContext(context.Background())
			cmd.SetContext(ctx)
			cmd.SetArgs(tc.args)

			err := cmd.Execute()
			if tc.expectErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

// TestCmdWebAuthNVerifyRegisterFlags tests the flags and options
func (s *CLITestSuite) TestCmdWebAuthNVerifyRegisterFlags() {
	cmd := cli.CmdWebAuthNVerifyRegister()
	require.NotNil(s.T(), cmd)

	// Test that query flags are added
	require.True(s.T(), cmd.Flags().HasAvailableFlags())

	// Test setting context with flags
	cmd.SetArgs([]string{"addr", "challenge", "rp", "data", "--node", "tcp://localhost:26657"})
	err := cmd.Execute()
	require.Error(s.T(), err) // Will fail on actual query but flags should parse

	// Test output format flag
	cmd.SetArgs([]string{"addr", "challenge", "rp", "data", "--output", "json"})
	err = cmd.Execute()
	require.Error(s.T(), err) // Will fail on actual query but flags should parse
}

// TestCmdWebAuthNVerifyRegisterRunE tests the RunE function properties
func (s *CLITestSuite) TestCmdWebAuthNVerifyRegisterRunE() {
	cmd := cli.CmdWebAuthNVerifyRegister()
	require.NotNil(s.T(), cmd.RunE)

	// Test that RunE exists and is a function
	require.IsType(s.T(), func(*cobra.Command, []string) error { return nil }, cmd.RunE)
}

// TestCmdWebAuthNVerifyAuthenticate tests the webauthn verify authenticate command structure
func (s *CLITestSuite) TestCmdWebAuthNVerifyAuthenticate() {
	cmd := cli.CmdWebAuthNVerifyAuthenticate()
	require.NotNil(s.T(), cmd)
	require.Equal(s.T(), "webauthn-authenticate [addr] [challenge] [rp] [credential] [data]", cmd.Use)
	require.Equal(s.T(), "Test Webauthn Authentication", cmd.Short)

	// Test that command requires exactly five arguments
	require.NotNil(s.T(), cmd.Args)

	// Test execution with no args (should return error about missing argument)
	err := cmd.Args(cmd, []string{})
	require.Error(s.T(), err)

	// Test execution with correct number of args (should pass)
	err = cmd.Args(cmd, []string{"addr", "challenge", "rp", "credential", "data"})
	require.NoError(s.T(), err)
}

// TestCmdWebAuthNVerifyAuthenticateExecution tests webauthn authenticate command execution
func (s *CLITestSuite) TestCmdWebAuthNVerifyAuthenticateExecution() {
	testCases := []struct {
		name      string
		args      []string
		expectErr bool
	}{
		{
			"help flag execution",
			[]string{"--help"},
			false,
		},
		{
			"insufficient args",
			[]string{},
			true,
		},
		{
			"too many args",
			[]string{"arg1", "arg2", "arg3", "arg4", "arg5", "arg6"},
			true,
		},
		{
			"with required args but no server",
			[]string{"addr", "challenge", "rp", "credential", "data"},
			true, // Should error because no actual server
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			cmd := cli.CmdWebAuthNVerifyAuthenticate()
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)

			ctx := svrcmd.CreateExecuteContext(context.Background())
			cmd.SetContext(ctx)
			cmd.SetArgs(tc.args)

			err := cmd.Execute()
			if tc.expectErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

// TestCmdWebAuthNVerifyAuthenticateFlags tests the flags and options
func (s *CLITestSuite) TestCmdWebAuthNVerifyAuthenticateFlags() {
	cmd := cli.CmdWebAuthNVerifyAuthenticate()
	require.NotNil(s.T(), cmd)

	// Test that query flags are added
	require.True(s.T(), cmd.Flags().HasAvailableFlags())

	// Test setting context with flags
	cmd.SetArgs([]string{"addr", "challenge", "rp", "credential", "data", "--node", "tcp://localhost:26657"})
	err := cmd.Execute()
	require.Error(s.T(), err) // Will fail on actual query but flags should parse

	// Test output format flag
	cmd.SetArgs([]string{"addr", "challenge", "rp", "credential", "data", "--output", "json"})
	err = cmd.Execute()
	require.Error(s.T(), err) // Will fail on actual query but flags should parse
}

// TestCmdWebAuthNVerifyAuthenticateRunE tests the RunE function properties
func (s *CLITestSuite) TestCmdWebAuthNVerifyAuthenticateRunE() {
	cmd := cli.CmdWebAuthNVerifyAuthenticate()
	require.NotNil(s.T(), cmd.RunE)

	// Test that RunE exists and is a function
	require.IsType(s.T(), func(*cobra.Command, []string) error { return nil }, cmd.RunE)
}
