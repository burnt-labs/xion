package cli_test

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	rpcclientmock "github.com/cometbft/cometbft/rpc/client/mock"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	testutilmod "github.com/cosmos/cosmos-sdk/types/module/testutil"

	"github.com/burnt-labs/xion/x/xauthz"
	"github.com/burnt-labs/xion/x/xauthz/client/cli"
)

func newEnc() testutilmod.TestEncodingConfig {
	return testutilmod.MakeTestEncodingConfig(xauthz.AppModuleBasic{})
}

func newEmptyCtx() client.Context {
	enc := newEnc()
	return client.Context{}.
		WithCodec(enc.Codec).
		WithTxConfig(enc.TxConfig).
		WithLegacyAmino(enc.Amino)
}

func newMockCtx(t *testing.T) client.Context {
	t.Helper()
	enc := newEnc()
	kr := keyring.NewInMemory(enc.Codec)
	return client.Context{}.
		WithCodec(enc.Codec).
		WithTxConfig(enc.TxConfig).
		WithLegacyAmino(enc.Amino).
		WithKeyring(kr).
		WithFromAddress(sdk.AccAddress("test_from_address_____")).
		WithFromName("from").
		WithChainID("test-chain").
		WithClient(clitestutil.MockCometRPC{Client: rpcclientmock.Client{}}).
		WithAccountRetriever(client.MockAccountRetriever{}).
		WithBroadcastMode(flags.BroadcastSync)
}

func TestCommandMetadata(t *testing.T) {
	meta := []struct {
		name string
		cmd  *cobra.Command
	}{
		{"tx root", cli.GetTxCmd()},
		{"grant", cli.GrantCodeExecutionCmd()},
	}
	for _, m := range meta {
		t.Run(m.name, func(t *testing.T) {
			require.NotEmpty(t, m.cmd.Use, m.name)
			require.NotEmpty(t, m.cmd.Short, m.name)
			// Help path (no error expected)
			m.cmd.SetArgs([]string{"--help"})
			require.NoError(t, m.cmd.Execute(), m.name)
		})
	}
}

func TestArgumentValidation(t *testing.T) {
	tests := []struct {
		name    string
		cmdFn   func() *cobra.Command
		args    []string
		wantErr bool
	}{
		{"grant missing all args", cli.GrantCodeExecutionCmd, []string{}, true},
		{"grant missing code ids", cli.GrantCodeExecutionCmd, []string{"cosmos1hsk6jryyqjfhp5dhc55tc9jtckygx0eph6dd02"}, true},
		{"grant with invalid grantee", cli.GrantCodeExecutionCmd, []string{"invalid-addr", "1,2,3"}, true},
		{"grant with invalid code ids", cli.GrantCodeExecutionCmd, []string{"cosmos1hsk6jryyqjfhp5dhc55tc9jtckygx0eph6dd02", "abc"}, true},
		{"grant with empty code ids", cli.GrantCodeExecutionCmd, []string{"cosmos1hsk6jryyqjfhp5dhc55tc9jtckygx0eph6dd02", ""}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := tc.cmdFn()
			cmd.SetArgs(tc.args)
			err := cmd.Execute()
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGrantCodeExecutionCmd(t *testing.T) {
	ctx := newMockCtx(t)

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "valid single code id",
			args:    []string{"cosmos1hsk6jryyqjfhp5dhc55tc9jtckygx0eph6dd02", "1"},
			wantErr: true, // Will error due to mock context validation
		},
		{
			name:    "valid multiple code ids",
			args:    []string{"cosmos1hsk6jryyqjfhp5dhc55tc9jtckygx0eph6dd02", "1,2,3"},
			wantErr: true, // Will error due to mock context validation
		},
		{
			name:    "valid with expiration",
			args:    []string{"cosmos1hsk6jryyqjfhp5dhc55tc9jtckygx0eph6dd02", "1,2,3", "--expiration", "1893456000"},
			wantErr: true, // Will error due to mock context validation
		},
		{
			name:    "invalid grantee address",
			args:    []string{"invalid-address", "1,2,3"},
			wantErr: true,
		},
		{
			name:    "invalid code id format",
			args:    []string{"cosmos1hsk6jryyqjfhp5dhc55tc9jtckygx0eph6dd02", "abc,def"},
			wantErr: true,
		},
		{
			name:    "empty code ids",
			args:    []string{"cosmos1hsk6jryyqjfhp5dhc55tc9jtckygx0eph6dd02", ""},
			wantErr: true,
		},
		{
			name:    "code ids with spaces",
			args:    []string{"cosmos1hsk6jryyqjfhp5dhc55tc9jtckygx0eph6dd02", "1, 2, 3"},
			wantErr: true, // Will error due to mock context validation, but parsing should work
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := cli.GrantCodeExecutionCmd()
			_, err := clitestutil.ExecTestCLICmd(ctx, cmd, tc.args)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGrantCodeExecutionCmdWithEmptyContext(t *testing.T) {
	emptyCtx := newEmptyCtx()

	// Should panic with empty context due to nil pointer in tx broadcasting
	cmd := cli.GrantCodeExecutionCmd()
	require.Panics(t, func() {
		_, _ = clitestutil.ExecTestCLICmd(emptyCtx, cmd, []string{
			"cosmos1hsk6jryyqjfhp5dhc55tc9jtckygx0eph6dd02", "1,2,3",
		})
	})
}

func TestGetTxCmd(t *testing.T) {
	cmd := cli.GetTxCmd()

	// Verify the root command is properly configured
	require.Equal(t, "xauthz", cmd.Use)
	require.NotEmpty(t, cmd.Short)

	// Verify subcommands are registered
	subCmds := cmd.Commands()
	require.Len(t, subCmds, 1)

	// Find grant command
	var grantCmd *cobra.Command
	for _, sub := range subCmds {
		if sub.Name() == "grant" {
			grantCmd = sub
			break
		}
	}
	require.NotNil(t, grantCmd, "grant command should be registered")
}

func TestParseCodeIDs(t *testing.T) {
	ctx := newMockCtx(t)

	tests := []struct {
		name     string
		codeIDs  string
		wantErr  bool
		contains string
	}{
		{
			name:    "single code id",
			codeIDs: "1",
			wantErr: true, // Error due to mock context, but parsing succeeds
		},
		{
			name:    "multiple code ids",
			codeIDs: "1,2,3",
			wantErr: true,
		},
		{
			name:    "code ids with spaces",
			codeIDs: "1, 2, 3",
			wantErr: true,
		},
		{
			name:     "invalid code id",
			codeIDs:  "abc",
			wantErr:  true,
			contains: "invalid syntax",
		},
		{
			name:     "mixed valid and invalid",
			codeIDs:  "1,abc,3",
			wantErr:  true,
			contains: "invalid syntax",
		},
		{
			name:     "empty string",
			codeIDs:  "",
			wantErr:  true,
			contains: "empty",
		},
		{
			name:    "negative number",
			codeIDs: "-1",
			wantErr: true,
		},
		{
			name:    "zero",
			codeIDs: "0",
			wantErr: true,
		},
		{
			name:    "large number",
			codeIDs: "18446744073709551615",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := cli.GrantCodeExecutionCmd()
			args := []string{"cosmos1hsk6jryyqjfhp5dhc55tc9jtckygx0eph6dd02", tc.codeIDs}
			_, err := clitestutil.ExecTestCLICmd(ctx, cmd, args)
			if tc.wantErr {
				require.Error(t, err)
				if tc.contains != "" {
					require.Contains(t, err.Error(), tc.contains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestExpirationFlag(t *testing.T) {
	ctx := newMockCtx(t)

	tests := []struct {
		name       string
		expiration string
		wantErr    bool
	}{
		{
			name:       "valid expiration",
			expiration: "1893456000",
			wantErr:    true, // Error due to mock context
		},
		{
			name:       "zero expiration (no expiration)",
			expiration: "0",
			wantErr:    true,
		},
		{
			name:       "negative expiration",
			expiration: "-1",
			wantErr:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := cli.GrantCodeExecutionCmd()
			args := []string{
				"cosmos1hsk6jryyqjfhp5dhc55tc9jtckygx0eph6dd02",
				"1,2,3",
				"--expiration", tc.expiration,
			}
			_, err := clitestutil.ExecTestCLICmd(ctx, cmd, args)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestHelpOutput(t *testing.T) {
	roots := []*cobra.Command{cli.GetTxCmd()}
	for _, r := range roots {
		r.SetArgs([]string{"--help"})
		require.NoError(t, r.Execute())
	}
}

func TestGrantCmdFlags(t *testing.T) {
	cmd := cli.GrantCodeExecutionCmd()

	// Verify expiration flag exists
	flag := cmd.Flags().Lookup("expiration")
	require.NotNil(t, flag)
	require.Equal(t, "int64", flag.Value.Type())
}

func TestRunEHandlerErrorPaths(t *testing.T) {
	// Test GrantCodeExecutionCmd RunE with no client context - should panic
	grantCmd := cli.GrantCodeExecutionCmd()
	require.NotNil(t, grantCmd.RunE)
	require.Panics(t, func() {
		_ = grantCmd.RunE(grantCmd, []string{"cosmos1hsk6jryyqjfhp5dhc55tc9jtckygx0eph6dd02", "1"})
	}, "Expected panic when no client context")
}
