package cli_test

import (
	"context"
	"io"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/client/flags"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"

	"github.com/burnt-labs/xion/x/xion/client/cli"
)

// Consolidated command metadata & arg validation (reduces multiple redundant tests)
func TestCommandMetadataAndArgs(t *testing.T) {
	cases := []struct {
		name        string
		newCmd      func() *cobra.Command
		useContains string
		short       string
		validArgs   [][]string // arg sets expected to pass
		invalidArgs [][]string // arg sets expected to fail
	}{
		{
			name:        "register",
			newCmd:      cli.NewRegisterCmd,
			useContains: "register",
			short:       "Register an abstract account",
			validArgs:   [][]string{{}, {"1"}, {"1", "key"}},
			invalidArgs: [][]string{{"1", "key", "extra"}},
		},
		{
			name:        "add-authenticator",
			newCmd:      cli.NewAddAuthenticatorCmd,
			useContains: "add-authenticator",
			short:       "Add the signing key as an authenticator to an abstract account",
			validArgs:   [][]string{{"addr"}},
			invalidArgs: [][]string{{}, {"a", "b"}},
		},
		{
			name:        "sign",
			newCmd:      cli.NewSignCmd,
			useContains: "sign",
			short:       "sign a transaction",
			validArgs:   [][]string{{"k", "acct", "file"}},
			invalidArgs: [][]string{{}, {"k"}, {"k", "a"}, {"k", "a", "b", "c"}},
		},
		{
			name:        "emit",
			newCmd:      cli.NewEmitArbitraryDataCmd,
			useContains: "emit",
			short:       "Emit an arbitrary data from the chain",
			validArgs:   [][]string{{"data", "contract"}},
			invalidArgs: [][]string{{}, {"only"}, {"a", "b", "c"}},
		},
		{
			name:        "update-params",
			newCmd:      cli.NewUpdateParamsCmd,
			useContains: "update-params",
			short:       "Update treasury contract parameters",
			validArgs:   [][]string{{"c", "d", "r", "i"}},
			invalidArgs: [][]string{{}, {"c"}, {"c", "d", "r"}, {"c", "d", "r", "i", "x"}},
		},
		{
			name:        "update-configs",
			newCmd:      cli.NewUpdateConfigsCmd,
			useContains: "update-configs",
			short:       "Batch update grant configs and fee config for the treasury",
			validArgs:   [][]string{{"c", "path"}},
			invalidArgs: [][]string{{}, {"c"}, {"c", "p", "x"}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := tc.newCmd()
			require.Contains(t, cmd.Use, tc.useContains)
			require.Equal(t, tc.short, cmd.Short)
			for _, a := range tc.validArgs {
				require.NoError(t, cmd.Args(cmd, a), "args should be valid: %v", a)
			}
			for _, a := range tc.invalidArgs {
				require.Error(t, cmd.Args(cmd, a), "args should be invalid: %v", a)
			}
			// smoke help
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs([]string{"--help"})
			ctx := svrcmd.CreateExecuteContext(context.Background())
			cmd.SetContext(ctx)
			require.NoError(t, cmd.Execute())
		})
	}
}

// Simplified flag presence test
func TestCommandCommonFlags(t *testing.T) {
	cmds := []struct {
		name string
		cmd  func() *cobra.Command
	}{
		{"register", cli.NewRegisterCmd},
		{"add-authenticator", cli.NewAddAuthenticatorCmd},
		{"sign", cli.NewSignCmd},
		{"emit", cli.NewEmitArbitraryDataCmd},
		{"update-params", cli.NewUpdateParamsCmd},
		{"update-configs", cli.NewUpdateConfigsCmd},
	}
	for _, c := range cmds {
		t.Run(c.name, func(t *testing.T) {
			cmd := c.cmd()
			require.NotNil(t, cmd.Flag(flags.FlagChainID))
			require.NotNil(t, cmd.Flag(flags.FlagFrom))
			require.NotNil(t, cmd.Flag(flags.FlagGas))
		})
	}
}
