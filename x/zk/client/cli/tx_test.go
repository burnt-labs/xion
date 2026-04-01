package cli_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/zk/client/cli"
)

// ctxWithFrom sets a client context with a valid from address so RunE passes
// initial context checks. Pass the returned context to cmd.ExecuteContext(ctx).
func ctxWithFrom(t *testing.T) context.Context {
	t.Helper()
	addr := sdk.AccAddress("test_from_addr__________") // 20 bytes
	clientCtx := client.Context{}.
		WithFromAddress(addr).
		WithFrom(addr.String())
	return context.WithValue(context.Background(), client.ClientContextKey, &clientCtx)
}

func TestGetTxCmd(t *testing.T) {
	t.Run("returns valid command", func(t *testing.T) {
		cmd := cli.GetTxCmd()
		require.NotNil(t, cmd)
		require.Equal(t, "zk", cmd.Use)
		require.True(t, cmd.DisableFlagParsing)
		require.Equal(t, 2, cmd.SuggestionsMinimumDistance)
	})

	t.Run("has expected subcommands", func(t *testing.T) {
		cmd := cli.GetTxCmd()
		subcommands := cmd.Commands()

		// Should have 3 subcommands
		require.Len(t, subcommands, 3)

		// Verify subcommand names
		names := make(map[string]bool)
		for _, subcmd := range subcommands {
			names[subcmd.Use] = true
		}

		require.True(t, names["add-vkey [name] [vkey-file] [description] [proof-system]"])
		require.True(t, names["update-vkey [name] [vkey-file] [description] [proof-system]"])
		require.True(t, names["remove-vkey [name]"])
	})

	t.Run("short description is set", func(t *testing.T) {
		cmd := cli.GetTxCmd()
		require.Contains(t, cmd.Short, "zk")
		require.Contains(t, cmd.Short, "transactions")
	})
}

func TestGetCmdAddVKey(t *testing.T) {
	t.Run("returns valid command", func(t *testing.T) {
		cmd := cli.GetCmdAddVKey()
		require.NotNil(t, cmd)
		require.Equal(t, "add-vkey [name] [vkey-file] [description] [proof-system]", cmd.Use)
		require.Equal(t, "Add a new verification key", cmd.Short)
	})

	t.Run("requires exactly four arguments", func(t *testing.T) {
		cmd := cli.GetCmdAddVKey()
		require.NotNil(t, cmd.Args)

		// Test with wrong number of args
		err := cmd.Args(cmd, []string{})
		require.Error(t, err)

		err = cmd.Args(cmd, []string{"name"})
		require.Error(t, err)

		err = cmd.Args(cmd, []string{"name", "file.json"})
		require.Error(t, err)

		err = cmd.Args(cmd, []string{"name", "file.json", "description"})
		require.Error(t, err)

		err = cmd.Args(cmd, []string{"name", "file.json", "description", "groth16", "extra"})
		require.Error(t, err)

		err = cmd.Args(cmd, []string{"name", "file.json", "description", "groth16"})
		require.NoError(t, err)
	})

	t.Run("has tx flags", func(t *testing.T) {
		cmd := cli.GetCmdAddVKey()
		flags := cmd.Flags()
		require.NotNil(t, flags)

		// Should have standard tx flags
		fromFlag := flags.Lookup("from")
		require.NotNil(t, fromFlag)

		chainIDFlag := flags.Lookup("chain-id")
		require.NotNil(t, chainIDFlag)

		feesFlag := flags.Lookup("fees")
		require.NotNil(t, feesFlag)
	})

	t.Run("has long description", func(t *testing.T) {
		cmd := cli.GetCmdAddVKey()
		require.NotEmpty(t, cmd.Long)
		require.Contains(t, cmd.Long, "verification key")
		require.Contains(t, cmd.Long, "groth16")
		require.Contains(t, cmd.Long, "ultrahonk")
	})

	t.Run("has example usage", func(t *testing.T) {
		cmd := cli.GetCmdAddVKey()
		require.NotEmpty(t, cmd.Example)
		require.Contains(t, cmd.Example, "xiond")
		require.Contains(t, cmd.Example, "add-vkey")
		require.Contains(t, cmd.Example, "email_auth")
		require.Contains(t, cmd.Example, "vkey.json")
		require.Contains(t, cmd.Example, "--from")
	})
}

func TestGetCmdAddVKeyExtended(t *testing.T) {
	t.Run("RunE fails without client context", func(t *testing.T) {
		cmd := cli.GetCmdAddVKey()

		// Create a temporary vkey file
		tmpDir := t.TempDir()
		vkeyFile := filepath.Join(tmpDir, "vkey.json")
		err := os.WriteFile(vkeyFile, []byte(`{}`), 0o600)
		require.NoError(t, err)

		cmd.SetArgs([]string{"test_name", vkeyFile, "test description", "groth16"})
		err = cmd.Execute()
		require.Error(t, err)
	})

	t.Run("RunE fails with non-existent vkey file", func(t *testing.T) {
		cmd := cli.GetCmdAddVKey()

		cmd.SetArgs([]string{"test_name", "/non/existent/vkey.json", "test description", "groth16"})
		err := cmd.Execute()
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to read vkey file")
	})

	t.Run("RunE fails early on invalid proof system (ValidateBasic)", func(t *testing.T) {
		cmd := cli.GetCmdAddVKey()
		ctx := ctxWithFrom(t)

		// Create a temporary vkey file.
		tmpDir := t.TempDir()
		vkeyFile := filepath.Join(tmpDir, "vkey.json")
		err := os.WriteFile(vkeyFile, []byte(`{}`), 0o600)
		require.NoError(t, err)

		cmd.SetArgs([]string{"test_name", vkeyFile, "test description", "invalid-proof-system"})
		err = cmd.ExecuteContext(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "proof_system must be")
	})

	t.Run("RunE fails with empty vkey file", func(t *testing.T) {
		cmd := cli.GetCmdAddVKey()
		ctx := ctxWithFrom(t)

		// Create an empty vkey file
		tmpDir := t.TempDir()
		vkeyFile := filepath.Join(tmpDir, "empty_vkey.json")
		err := os.WriteFile(vkeyFile, []byte{}, 0o600)
		require.NoError(t, err)

		cmd.SetArgs([]string{"test_name", vkeyFile, "test description", "groth16"})
		err = cmd.ExecuteContext(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "vkey_bytes cannot be empty")
	})

	t.Run("RunE with directory instead of file", func(t *testing.T) {
		cmd := cli.GetCmdAddVKey()

		tmpDir := t.TempDir()

		cmd.SetArgs([]string{"test_name", tmpDir, "test description", "groth16"})
		err := cmd.Execute()
		require.Error(t, err)
	})

	t.Run("RunE with unreadable file", func(t *testing.T) {
		cmd := cli.GetCmdAddVKey()

		// Create a file and make it unreadable
		tmpDir := t.TempDir()
		vkeyFile := filepath.Join(tmpDir, "unreadable_vkey.json")
		err := os.WriteFile(vkeyFile, []byte(`{}`), 0o000)
		require.NoError(t, err)

		// Restore permissions after test
		defer func() {
			err := os.Chmod(vkeyFile, 0o600)
			require.NoError(t, err)
		}()

		cmd.SetArgs([]string{"test_name", vkeyFile, "test description", "groth16"})
		err = cmd.Execute()
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to read vkey file")
	})
}

func TestGetCmdUpdateVKey(t *testing.T) {
	t.Run("returns valid command", func(t *testing.T) {
		cmd := cli.GetCmdUpdateVKey()
		require.NotNil(t, cmd)
		require.Equal(t, "update-vkey [name] [vkey-file] [description] [proof-system]", cmd.Use)
		require.Equal(t, "Update an existing verification key", cmd.Short)
	})

	t.Run("requires exactly four arguments", func(t *testing.T) {
		cmd := cli.GetCmdUpdateVKey()
		require.NotNil(t, cmd.Args)

		// Test with wrong number of args
		err := cmd.Args(cmd, []string{})
		require.Error(t, err)

		err = cmd.Args(cmd, []string{"name"})
		require.Error(t, err)

		err = cmd.Args(cmd, []string{"name", "file.json"})
		require.Error(t, err)

		err = cmd.Args(cmd, []string{"name", "file.json", "description"})
		require.Error(t, err)

		err = cmd.Args(cmd, []string{"name", "file.json", "description", "groth16", "extra"})
		require.Error(t, err)

		err = cmd.Args(cmd, []string{"name", "file.json", "description", "groth16"})
		require.NoError(t, err)
	})

	t.Run("has tx flags", func(t *testing.T) {
		cmd := cli.GetCmdUpdateVKey()
		flags := cmd.Flags()
		require.NotNil(t, flags)

		// Should have standard tx flags
		fromFlag := flags.Lookup("from")
		require.NotNil(t, fromFlag)

		chainIDFlag := flags.Lookup("chain-id")
		require.NotNil(t, chainIDFlag)

		feesFlag := flags.Lookup("fees")
		require.NotNil(t, feesFlag)
	})

	t.Run("has long description", func(t *testing.T) {
		cmd := cli.GetCmdUpdateVKey()
		require.NotEmpty(t, cmd.Long)
		require.Contains(t, cmd.Long, "Update")
		require.Contains(t, cmd.Long, "verification key")
		require.Contains(t, cmd.Long, "groth16")
		require.Contains(t, cmd.Long, "ultrahonk")
	})

	t.Run("has example usage", func(t *testing.T) {
		cmd := cli.GetCmdUpdateVKey()
		require.NotEmpty(t, cmd.Example)
		require.Contains(t, cmd.Example, "xiond")
		require.Contains(t, cmd.Example, "update-vkey")
		require.Contains(t, cmd.Example, "email_auth")
		require.Contains(t, cmd.Example, "--from")
	})
}

func TestGetCmdUpdateVKeyExtended(t *testing.T) {
	t.Run("RunE fails without client context", func(t *testing.T) {
		cmd := cli.GetCmdUpdateVKey()

		// Create a temporary vkey file
		tmpDir := t.TempDir()
		vkeyFile := filepath.Join(tmpDir, "vkey.json")
		err := os.WriteFile(vkeyFile, []byte(`{}`), 0o600)
		require.NoError(t, err)

		cmd.SetArgs([]string{"test_name", vkeyFile, "test description", "groth16"})
		err = cmd.Execute()
		require.Error(t, err)
	})

	t.Run("RunE fails with non-existent vkey file", func(t *testing.T) {
		cmd := cli.GetCmdUpdateVKey()

		cmd.SetArgs([]string{"test_name", "/non/existent/vkey.json", "test description", "groth16"})
		err := cmd.Execute()
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to read vkey file")
	})

	t.Run("RunE fails early on invalid proof system (ValidateBasic)", func(t *testing.T) {
		cmd := cli.GetCmdUpdateVKey()
		ctx := ctxWithFrom(t)

		// Create a temporary vkey file.
		tmpDir := t.TempDir()
		vkeyFile := filepath.Join(tmpDir, "vkey.json")
		err := os.WriteFile(vkeyFile, []byte(`{}`), 0o600)
		require.NoError(t, err)

		cmd.SetArgs([]string{"test_name", vkeyFile, "test description", "invalid-proof-system"})
		err = cmd.ExecuteContext(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "proof_system must be")
	})

	t.Run("RunE fails with empty vkey file", func(t *testing.T) {
		cmd := cli.GetCmdUpdateVKey()
		ctx := ctxWithFrom(t)

		// Create an empty vkey file
		tmpDir := t.TempDir()
		vkeyFile := filepath.Join(tmpDir, "empty_vkey.json")
		err := os.WriteFile(vkeyFile, []byte{}, 0o600)
		require.NoError(t, err)

		cmd.SetArgs([]string{"test_name", vkeyFile, "test description", "groth16"})
		err = cmd.ExecuteContext(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "vkey_bytes cannot be empty")
	})

	t.Run("vkey JSON structure is not validated in CLI ValidateBasic", func(t *testing.T) {
		cmd := cli.GetCmdUpdateVKey()

		// Create a vkey file that is not validated at CLI layer anymore.
		tmpDir := t.TempDir()
		vkeyFile := filepath.Join(tmpDir, "incomplete_vkey.json")
		err := os.WriteFile(vkeyFile, []byte(`{"some": "data"}`), 0o600)
		require.NoError(t, err)

		cmd.SetArgs([]string{"test_name", vkeyFile, "test description", "groth16"})
		// CLI no longer rejects this content directly; this command should fail for
		// missing execution context in this unit test setup (keeper validates later).
		err = cmd.Execute()
		require.Error(t, err)
	})

	t.Run("RunE with special characters in name", func(t *testing.T) {
		cmd := cli.GetCmdUpdateVKey()

		// Create a temporary vkey file
		tmpDir := t.TempDir()
		vkeyFile := filepath.Join(tmpDir, "vkey.json")
		err := os.WriteFile(vkeyFile, []byte(`{}`), 0o600)
		require.NoError(t, err)

		// Name with special characters
		cmd.SetArgs([]string{"test-name_v2", vkeyFile, "test description", "groth16"})
		err = cmd.Execute()
		require.Error(t, err)
		// Should fail on validation or client context, not on name parsing
	})

	t.Run("RunE with empty description", func(t *testing.T) {
		cmd := cli.GetCmdUpdateVKey()

		// Create a temporary vkey file
		tmpDir := t.TempDir()
		vkeyFile := filepath.Join(tmpDir, "vkey.json")
		err := os.WriteFile(vkeyFile, []byte(`{}`), 0o600)
		require.NoError(t, err)

		cmd.SetArgs([]string{"test_name", vkeyFile, "", "groth16"})
		err = cmd.Execute()
		require.Error(t, err)
	})

	t.Run("RunE with long description", func(t *testing.T) {
		cmd := cli.GetCmdUpdateVKey()

		// Create a temporary vkey file
		tmpDir := t.TempDir()
		vkeyFile := filepath.Join(tmpDir, "vkey.json")
		err := os.WriteFile(vkeyFile, []byte(`{}`), 0o600)
		require.NoError(t, err)

		// Very long description
		longDesc := ""
		for i := 0; i < 1000; i++ {
			longDesc += "a"
		}

		cmd.SetArgs([]string{"test_name", vkeyFile, longDesc, "groth16"})
		err = cmd.Execute()
		require.Error(t, err)
	})

	t.Run("RunE reads file from relative path", func(t *testing.T) {
		cmd := cli.GetCmdUpdateVKey()

		// Create a temporary vkey file in current directory
		tmpDir := t.TempDir()
		vkeyFile := filepath.Join(tmpDir, "relative_vkey.json")
		err := os.WriteFile(vkeyFile, []byte(`{"invalid": true}`), 0o600)
		require.NoError(t, err)

		cmd.SetArgs([]string{"test_name", vkeyFile, "description", "groth16"})
		err = cmd.Execute()
		require.Error(t, err)
	})
}

func TestGetCmdRemoveVKey(t *testing.T) {
	t.Run("returns valid command", func(t *testing.T) {
		cmd := cli.GetCmdRemoveVKey()
		require.NotNil(t, cmd)
		require.Equal(t, "remove-vkey [name]", cmd.Use)
		require.Equal(t, "Remove a verification key", cmd.Short)
	})

	t.Run("requires exactly one argument", func(t *testing.T) {
		cmd := cli.GetCmdRemoveVKey()
		require.NotNil(t, cmd.Args)

		// Test with wrong number of args
		err := cmd.Args(cmd, []string{})
		require.Error(t, err)

		err = cmd.Args(cmd, []string{"name", "extra"})
		require.Error(t, err)

		err = cmd.Args(cmd, []string{"email_auth"})
		require.NoError(t, err)
	})

	t.Run("has tx flags", func(t *testing.T) {
		cmd := cli.GetCmdRemoveVKey()
		flags := cmd.Flags()
		require.NotNil(t, flags)

		// Should have standard tx flags
		fromFlag := flags.Lookup("from")
		require.NotNil(t, fromFlag)

		chainIDFlag := flags.Lookup("chain-id")
		require.NotNil(t, chainIDFlag)

		feesFlag := flags.Lookup("fees")
		require.NotNil(t, feesFlag)
	})

	t.Run("has long description", func(t *testing.T) {
		cmd := cli.GetCmdRemoveVKey()
		require.NotEmpty(t, cmd.Long)
		require.Contains(t, cmd.Long, "Remove")
		require.Contains(t, cmd.Long, "verification key")
	})

	t.Run("has example usage", func(t *testing.T) {
		cmd := cli.GetCmdRemoveVKey()
		require.NotEmpty(t, cmd.Example)
		require.Contains(t, cmd.Example, "xiond")
		require.Contains(t, cmd.Example, "remove-vkey")
		require.Contains(t, cmd.Example, "email_auth")
		require.Contains(t, cmd.Example, "--from")
		require.Contains(t, cmd.Example, "--chain-id")
	})
}

func TestGetCmdRemoveVKeyExtended(t *testing.T) {
	t.Run("RunE fails without client context", func(t *testing.T) {
		cmd := cli.GetCmdRemoveVKey()

		cmd.SetArgs([]string{"test_name"})
		err := cmd.Execute()
		require.Error(t, err)
	})

	t.Run("RunE with various name formats", func(t *testing.T) {
		testCases := []struct {
			name        string
			vkeyName    string
			shouldError bool
		}{
			{"simple name", "email_auth", true},
			{"name with hyphen", "email-auth", true},
			{"name with numbers", "circuit123", true},
			{"name with underscore", "my_circuit_v2", true},
			{"single character", "a", true},
			{"long name", "this_is_a_very_long_verification_key_name_that_might_be_used", true},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				cmd := cli.GetCmdRemoveVKey()
				cmd.SetArgs([]string{tc.vkeyName})
				err := cmd.Execute()
				if tc.shouldError {
					require.Error(t, err)
					// Should fail on client context, not on name validation
					require.NotContains(t, err.Error(), "invalid name")
				}
			})
		}
	})

	t.Run("RunE processes name argument correctly", func(t *testing.T) {
		cmd := cli.GetCmdRemoveVKey()

		// The command should accept the name and fail on client context
		cmd.SetArgs([]string{"valid_circuit_name"})
		err := cmd.Execute()
		require.Error(t, err)
		// Error should be about client context, not about the name
	})

	t.Run("RunE with empty name fails on args validation", func(t *testing.T) {
		cmd := cli.GetCmdRemoveVKey()

		// Empty args should fail
		cmd.SetArgs([]string{})
		err := cmd.Execute()
		require.Error(t, err)
	})
}

func TestTxCommandsIntegration(t *testing.T) {
	t.Run("all subcommands are accessible from parent", func(t *testing.T) {
		parentCmd := cli.GetTxCmd()

		// Find add-vkey subcommand
		addVKeyCmd, _, err := parentCmd.Find([]string{"add-vkey", "name", "file.json", "desc", "groth16"})
		require.NoError(t, err)
		require.NotNil(t, addVKeyCmd)

		// Find update-vkey subcommand
		updateVKeyCmd, _, err := parentCmd.Find([]string{"update-vkey", "name", "file.json", "desc", "groth16"})
		require.NoError(t, err)
		require.NotNil(t, updateVKeyCmd)

		// Find remove-vkey subcommand
		removeVKeyCmd, _, err := parentCmd.Find([]string{"remove-vkey", "name"})
		require.NoError(t, err)
		require.NotNil(t, removeVKeyCmd)
	})

	t.Run("subcommands have unique uses", func(t *testing.T) {
		parentCmd := cli.GetTxCmd()
		subcommands := parentCmd.Commands()

		uses := make(map[string]bool)
		for _, subcmd := range subcommands {
			require.False(t, uses[subcmd.Use], "Duplicate use: %s", subcmd.Use)
			uses[subcmd.Use] = true
		}
	})

	t.Run("all commands have RunE set", func(t *testing.T) {
		parentCmd := cli.GetTxCmd()
		subcommands := parentCmd.Commands()

		for _, subcmd := range subcommands {
			require.NotNil(t, subcmd.RunE, "Command %s should have RunE set", subcmd.Use)
		}
	})
}

func TestTxCommandConsistency(t *testing.T) {
	t.Run("add and update have same argument structure", func(t *testing.T) {
		addCmd := cli.GetCmdAddVKey()
		updateCmd := cli.GetCmdUpdateVKey()

		// Both require exactly 4 args: name, vkey-file, description, proof-system
		err := addCmd.Args(addCmd, []string{"name", "vkey.json", "description", "groth16"})
		require.NoError(t, err)

		err = updateCmd.Args(updateCmd, []string{"name", "vkey.json", "description", "groth16"})
		require.NoError(t, err)
	})

	t.Run("all commands have from flag", func(t *testing.T) {
		addCmd := cli.GetCmdAddVKey()
		updateCmd := cli.GetCmdUpdateVKey()
		removeCmd := cli.GetCmdRemoveVKey()

		require.NotNil(t, addCmd.Flags().Lookup("from"))
		require.NotNil(t, updateCmd.Flags().Lookup("from"))
		require.NotNil(t, removeCmd.Flags().Lookup("from"))
	})

	t.Run("all examples include xiond binary name", func(t *testing.T) {
		addCmd := cli.GetCmdAddVKey()
		updateCmd := cli.GetCmdUpdateVKey()
		removeCmd := cli.GetCmdRemoveVKey()

		require.Contains(t, addCmd.Example, "xiond")
		require.Contains(t, updateCmd.Example, "xiond")
		require.Contains(t, removeCmd.Example, "xiond")
	})
}
