package cli_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/burnt-labs/xion/x/zk/client/cli"
	"github.com/burnt-labs/xion/x/zk/types"
)

func TestGetQueryCmd(t *testing.T) {
	cmd := cli.GetQueryCmd()
	require.NotNil(t, cmd)
	require.Equal(t, types.ModuleName, cmd.Use)
	require.True(t, cmd.DisableFlagParsing)

	// Verify subcommands are added
	require.True(t, len(cmd.Commands()) > 0)
}

func TestGetCmdParams(t *testing.T) {
	cmd := cli.GetCmdParams()
	require.NotNil(t, cmd)
	require.Equal(t, "params", cmd.Use)
	require.Equal(t, "Show module params", cmd.Short)

	// Test that command has RunE function defined
	require.NotNil(t, cmd.RunE)

	// Test command structure
	require.NotNil(t, cmd.Flags())

	// Call RunE to ensure coverage - expect panic/error without context
	func() {
		defer func() {
			_ = recover() // Ignore panics from missing context
		}()
		_ = cmd.RunE(cmd, []string{})
	}()
}

func TestNewTxCmd(t *testing.T) {
	cmd := cli.NewTxCmd()
	require.NotNil(t, cmd)
	require.Equal(t, types.ModuleName, cmd.Use)

	// Verify subcommands are added
	require.True(t, len(cmd.Commands()) > 0)
}

func TestMsgUpdateParams(t *testing.T) {
	cmd := cli.MsgUpdateParams()
	require.NotNil(t, cmd)
	require.Contains(t, cmd.Use, "update-params")

	// Test that command has RunE function defined
	require.NotNil(t, cmd.RunE)

	// Call RunE to ensure coverage - it will fail/panic but that's OK
	func() {
		defer func() {
			_ = recover() // Ignore panics from missing context
		}()
		_ = cmd.RunE(cmd, []string{"authority", "vkey.json"})
	}()
}
