package cli_test

import (
	"context"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"

	"github.com/burnt-labs/xion/x/zk/client/cli"
	"github.com/burnt-labs/xion/x/zk/types"
)

// txClientContext returns a client context able to build and print an
// unsigned transaction without a node or a keyring.
func txClientContext(t *testing.T) client.Context {
	t.Helper()
	encCfg := moduletestutil.MakeTestEncodingConfig()
	types.RegisterInterfaces(encCfg.InterfaceRegistry)

	addr := sdk.AccAddress("test_from_addr__________")
	return client.Context{}.
		WithCodec(encCfg.Codec).
		WithInterfaceRegistry(encCfg.InterfaceRegistry).
		WithTxConfig(encCfg.TxConfig).
		WithFromAddress(addr).
		WithFrom(addr.String())
}

func generateOnlyFlags(t *testing.T) []string {
	t.Helper()
	return []string{
		"--" + flags.FlagGenerateOnly,
		"--" + flags.FlagChainID, "zk-test",
		"--" + flags.FlagKeyringBackend, "test",
		"--" + flags.FlagKeyringDir, t.TempDir(),
	}
}

func TestTxCommandsGenerateUnsignedTx(t *testing.T) {
	groth16File := writeTempFile(t, "vkey.json", types.CreateDefaultVKeyBytes())
	gnarkFile := writeTempFile(t, "vkey.bin", keeperFixture(t, "gnark", "vkey.bin"))
	honkFile := writeTempFile(t, "vk", keeperFixture(t, "barretenberg", "vk"))

	tests := []struct {
		name      string
		cmd       func() *cobra.Command
		args      []string
		ultrahonk bool
		typeURL   string
	}{
		{"add-vkey groth16", cli.GetCmdAddVKey, []string{"email", groth16File, "desc", "groth16"}, false, sdk.MsgTypeURL(&types.MsgAddVKey{})},
		{"add-vkey gnark", cli.GetCmdAddVKey, []string{"model", gnarkFile, "desc", "gnark"}, false, sdk.MsgTypeURL(&types.MsgAddVKey{})},
		{"add-vkey ultrahonk", cli.GetCmdAddVKey, []string{"rollup", honkFile, "desc", "ultrahonk"}, true, sdk.MsgTypeURL(&types.MsgAddVKey{})},
		{"update-vkey groth16", cli.GetCmdUpdateVKey, []string{"email", groth16File, "desc", "groth16"}, false, sdk.MsgTypeURL(&types.MsgUpdateVKey{})},
		{"update-vkey gnark", cli.GetCmdUpdateVKey, []string{"model", gnarkFile, "desc", "gnark"}, false, sdk.MsgTypeURL(&types.MsgUpdateVKey{})},
		{"update-vkey ultrahonk", cli.GetCmdUpdateVKey, []string{"rollup", honkFile, "desc", "ultrahonk"}, true, sdk.MsgTypeURL(&types.MsgUpdateVKey{})},
		{"remove-vkey", cli.GetCmdRemoveVKey, []string{"email"}, false, sdk.MsgTypeURL(&types.MsgRemoveVKey{})},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.ultrahonk {
				requireRealBarretenberg(t)
			}
			out, err := clitestutil.ExecTestCLICmd(txClientContext(t), tc.cmd(), append(tc.args, generateOnlyFlags(t)...))
			require.NoError(t, err)
			require.Contains(t, out.String(), tc.typeURL)
		})
	}
}

func TestTxCommandsRejectInvalidFeePayer(t *testing.T) {
	vkeyFile := writeTempFile(t, "vkey.json", types.CreateDefaultVKeyBytes())

	tests := []struct {
		name string
		cmd  func() *cobra.Command
		args []string
	}{
		{"add-vkey", cli.GetCmdAddVKey, []string{"email", vkeyFile, "desc", "groth16"}},
		{"update-vkey", cli.GetCmdUpdateVKey, []string{"email", vkeyFile, "desc", "groth16"}},
		{"remove-vkey", cli.GetCmdRemoveVKey, []string{"email"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := tc.cmd()
			cmd.SetArgs(append(tc.args, "--"+flags.FlagFeePayer, "not-an-address"))
			clientCtx := txClientContext(t)
			ctx := context.WithValue(context.Background(), client.ClientContextKey, &clientCtx)
			err := cmd.ExecuteContext(ctx)
			require.ErrorContains(t, err, "decoding bech32 failed")
		})
	}
}
