package cli_test

import (
	"context"
	"io"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client/flags"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"

	"github.com/burnt-labs/xion/x/xion/client/cli"
)

// Consolidated tests for platform fee queries (percentage & minimum) and webauthn verify commands.

func (s *CLITestSuite) TestPlatformFeeCommands_MetadataFlagsHelpAndErrorPaths() {
	type entry struct {
		name     string
		new      func() *cobra.Command
		use      string
		short    string
		helpArgs []string
		runArgs  []string
	}
	cases := []entry{
		{"platform-percentage", cli.CmdPlatformPercentage, "platform-percentage", "Get Platform Percentage", []string{"--help"}, []string{}},
		{"platform-minimum", cli.CmdPlatformMinimum, "platform-minimum", "Get Platform Minimum", []string{"--help"}, []string{}},
	}
	for _, c := range cases {
		s.Run(c.name, func() {
			// metadata & flags
			cmd := c.new()
			s.Require().Equal(c.use, cmd.Use)
			s.Require().Equal(c.short, cmd.Short)
			s.Require().NotNil(cmd.RunE)
			s.Require().NotNil(cmd.Flag(flags.FlagOutput))
			s.Require().NotNil(cmd.Flag(flags.FlagHeight))

			// help path (no error)
			helpCmd := c.new()
			helpCmd.SetOut(io.Discard)
			helpCmd.SetErr(io.Discard)
			helpCmd.SetArgs(c.helpArgs)
			s.Require().NoError(helpCmd.Execute())

			// execution path without proper client context should error
			runCmd := c.new()
			runCmd.SetOut(io.Discard)
			runCmd.SetErr(io.Discard)
			runCmd.SetArgs(c.runArgs)
			runCmd.SetContext(svrcmd.CreateExecuteContext(context.Background()))
			err := runCmd.Execute()
			s.Require().Error(err)

			// direct RunE invocation error path
			direct := c.new()
			direct.SetContext(context.Background())
			s.Require().Error(direct.RunE(direct, []string{}))
		})
	}
}

func (s *CLITestSuite) TestWebAuthNCommands_MetadataArgsAndRunE() {
	type waCase struct {
		name      string
		new       func() *cobra.Command
		goodArgs  []string
		badArgs   [][]string
		use       string
		shortDesc string
	}
	cases := []waCase{
		{
			name:      "webauthn-register",
			new:       cli.CmdWebAuthNVerifyRegister,
			goodArgs:  []string{"addr", "challenge", "rp", "data"},
			badArgs:   [][]string{{}, {"addr"}, {"addr", "challenge"}},
			use:       "webauthn-register [addr] [challenge] [rp] [data]",
			shortDesc: "Test Webauthn Registration",
		},
		{
			name:      "webauthn-authenticate",
			new:       cli.CmdWebAuthNVerifyAuthenticate,
			goodArgs:  []string{"addr", "challenge", "rp", "credential", "data"},
			badArgs:   [][]string{{}, {"addr"}, {"addr", "challenge"}, {"addr", "challenge", "rp"}},
			use:       "webauthn-authenticate [addr] [challenge] [rp] [credential] [data]",
			shortDesc: "Test Webauthn Authentication",
		},
	}
	for _, c := range cases {
		s.Run(c.name, func() {
			cmd := c.new()
			s.Require().Equal(c.use, cmd.Use)
			s.Require().Equal(c.shortDesc, cmd.Short)
			s.Require().NotNil(cmd.RunE)

			// arg validation
			for _, ba := range c.badArgs {
				s.Require().Error(cmd.Args(cmd, ba))
			}
			s.Require().NoError(cmd.Args(cmd, c.goodArgs))

			// execution (will error due to missing client context)
			cmdExec := c.new()
			cmdExec.SetOut(io.Discard)
			cmdExec.SetErr(io.Discard)
			cmdExec.SetArgs(c.goodArgs)
			cmdExec.SetContext(svrcmd.CreateExecuteContext(context.Background()))
			s.Require().Error(cmdExec.Execute())

			// direct RunE invocation
			runEDirect := c.new()
			runEDirect.SetContext(context.Background())
			s.Require().Error(runEDirect.RunE(runEDirect, c.goodArgs))
		})
	}
}
