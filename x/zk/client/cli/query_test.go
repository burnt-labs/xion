package cli_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/burnt-labs/xion/x/zk/client/cli"
)

func TestGetQueryCmd(t *testing.T) {
	t.Run("returns valid command", func(t *testing.T) {
		cmd := cli.GetQueryCmd()
		require.NotNil(t, cmd)
		require.Equal(t, "zk", cmd.Use)
		require.True(t, cmd.DisableFlagParsing)
		require.Equal(t, 2, cmd.SuggestionsMinimumDistance)
	})

	t.Run("has expected subcommands", func(t *testing.T) {
		cmd := cli.GetQueryCmd()
		subcommands := cmd.Commands()

		// Should have 6 subcommands
		require.Len(t, subcommands, 6)

		// Verify subcommand names
		names := make(map[string]bool)
		for _, subcmd := range subcommands {
			names[subcmd.Use] = true
		}

		require.True(t, names["vkey [id]"])
		require.True(t, names["vkey-by-name [name]"])
		require.True(t, names["vkeys"])
		require.True(t, names["has-vkey [name]"])
		require.True(t, names["verify-proof [proof-file]"])
		require.True(t, names["params"])
	})

	t.Run("short description is set", func(t *testing.T) {
		cmd := cli.GetQueryCmd()
		require.Contains(t, cmd.Short, "zk")
		require.Contains(t, cmd.Short, "Querying")
	})
}

// ============================================================================
// GetCmdQueryVKey Tests
// ============================================================================

func TestGetCmdQueryVKey(t *testing.T) {
	t.Run("returns valid command", func(t *testing.T) {
		cmd := cli.GetCmdQueryVKey()
		require.NotNil(t, cmd)
		require.Equal(t, "vkey [id]", cmd.Use)
		require.Equal(t, "Query a verification key by ID", cmd.Short)
	})

	t.Run("requires exactly one argument", func(t *testing.T) {
		cmd := cli.GetCmdQueryVKey()
		require.NotNil(t, cmd.Args)

		// Test with wrong number of args
		err := cmd.Args(cmd, []string{})
		require.Error(t, err)

		err = cmd.Args(cmd, []string{"1", "2"})
		require.Error(t, err)

		err = cmd.Args(cmd, []string{"1"})
		require.NoError(t, err)
	})

	t.Run("has query flags", func(t *testing.T) {
		cmd := cli.GetCmdQueryVKey()
		flags := cmd.Flags()
		require.NotNil(t, flags)

		// Should have standard query flags like --node, --height, etc.
		nodeFlag := flags.Lookup("node")
		require.NotNil(t, nodeFlag)
	})

	t.Run("has example usage", func(t *testing.T) {
		cmd := cli.GetCmdQueryVKey()
		require.NotEmpty(t, cmd.Example)
		require.Contains(t, cmd.Example, "xiond")
		require.Contains(t, cmd.Example, "vkey")
	})
}

// ============================================================================
// GetCmdQueryVKeyByName Tests
// ============================================================================

func TestGetCmdQueryVKeyByName(t *testing.T) {
	t.Run("returns valid command", func(t *testing.T) {
		cmd := cli.GetCmdQueryVKeyByName()
		require.NotNil(t, cmd)
		require.Equal(t, "vkey-by-name [name]", cmd.Use)
		require.Equal(t, "Query a verification key by name", cmd.Short)
	})

	t.Run("requires exactly one argument", func(t *testing.T) {
		cmd := cli.GetCmdQueryVKeyByName()
		require.NotNil(t, cmd.Args)

		err := cmd.Args(cmd, []string{})
		require.Error(t, err)

		err = cmd.Args(cmd, []string{"name1", "name2"})
		require.Error(t, err)

		err = cmd.Args(cmd, []string{"email_auth"})
		require.NoError(t, err)
	})

	t.Run("has query flags", func(t *testing.T) {
		cmd := cli.GetCmdQueryVKeyByName()
		flags := cmd.Flags()
		require.NotNil(t, flags)

		nodeFlag := flags.Lookup("node")
		require.NotNil(t, nodeFlag)
	})

	t.Run("has example usage", func(t *testing.T) {
		cmd := cli.GetCmdQueryVKeyByName()
		require.NotEmpty(t, cmd.Example)
		require.Contains(t, cmd.Example, "email_auth")
		require.Contains(t, cmd.Example, "rollup_circuit")
	})
}

// ============================================================================
// GetCmdQueryVKeys Tests
// ============================================================================

func TestGetCmdQueryVKeys(t *testing.T) {
	t.Run("returns valid command", func(t *testing.T) {
		cmd := cli.GetCmdQueryVKeys()
		require.NotNil(t, cmd)
		require.Equal(t, "vkeys", cmd.Use)
		require.Equal(t, "Query all verification keys with pagination", cmd.Short)
	})

	t.Run("has query flags", func(t *testing.T) {
		cmd := cli.GetCmdQueryVKeys()
		flags := cmd.Flags()
		require.NotNil(t, flags)

		nodeFlag := flags.Lookup("node")
		require.NotNil(t, nodeFlag)
	})

	t.Run("has pagination flags", func(t *testing.T) {
		cmd := cli.GetCmdQueryVKeys()
		flags := cmd.Flags()

		// Should have pagination flags
		limitFlag := flags.Lookup("limit")
		require.NotNil(t, limitFlag)

		offsetFlag := flags.Lookup("offset")
		require.NotNil(t, offsetFlag)
	})

	t.Run("has example usage with pagination", func(t *testing.T) {
		cmd := cli.GetCmdQueryVKeys()
		require.NotEmpty(t, cmd.Example)
		require.Contains(t, cmd.Example, "--limit")
		require.Contains(t, cmd.Example, "--offset")
		require.Contains(t, cmd.Example, "--page")
	})
}

// ============================================================================
// GetCmdQueryHasVKey Tests
// ============================================================================

func TestGetCmdQueryHasVKey(t *testing.T) {
	t.Run("returns valid command", func(t *testing.T) {
		cmd := cli.GetCmdQueryHasVKey()
		require.NotNil(t, cmd)
		require.Equal(t, "has-vkey [name]", cmd.Use)
		require.Equal(t, "Check if a verification key exists by name", cmd.Short)
	})

	t.Run("requires exactly one argument", func(t *testing.T) {
		cmd := cli.GetCmdQueryHasVKey()
		require.NotNil(t, cmd.Args)

		err := cmd.Args(cmd, []string{})
		require.Error(t, err)

		err = cmd.Args(cmd, []string{"name1", "name2"})
		require.Error(t, err)

		err = cmd.Args(cmd, []string{"email_auth"})
		require.NoError(t, err)
	})

	t.Run("has query flags", func(t *testing.T) {
		cmd := cli.GetCmdQueryHasVKey()
		flags := cmd.Flags()
		require.NotNil(t, flags)

		nodeFlag := flags.Lookup("node")
		require.NotNil(t, nodeFlag)
	})

	t.Run("has example usage", func(t *testing.T) {
		cmd := cli.GetCmdQueryHasVKey()
		require.NotEmpty(t, cmd.Example)
		require.Contains(t, cmd.Example, "email_auth")
		require.Contains(t, cmd.Example, "rollup_circuit")
	})
}

// ============================================================================
// GetCmdQueryParams Tests
// ============================================================================

func TestGetCmdQueryParams(t *testing.T) {
	t.Run("returns valid command", func(t *testing.T) {
		cmd := cli.GetCmdQueryParams()
		require.NotNil(t, cmd)
		require.Equal(t, "params", cmd.Use)
		require.Contains(t, cmd.Short, "parameters")
	})

	t.Run("has query flags", func(t *testing.T) {
		cmd := cli.GetCmdQueryParams()
		flags := cmd.Flags()
		require.NotNil(t, flags)

		nodeFlag := flags.Lookup("node")
		require.NotNil(t, nodeFlag)
	})

	t.Run("RunE fails without client context", func(t *testing.T) {
		cmd := cli.GetCmdQueryParams()

		err := cmd.Execute()
		require.Error(t, err)
	})
}

// ============================================================================
// GetCmdQueryVerifyProof Tests
// ============================================================================

func TestGetCmdQueryVerifyProof(t *testing.T) {
	t.Run("returns valid command", func(t *testing.T) {
		cmd := cli.GetCmdQueryVerifyProof()
		require.NotNil(t, cmd)
		require.Equal(t, "verify-proof [proof-file]", cmd.Use)
		require.Contains(t, cmd.Short, "Verify")
		require.Contains(t, cmd.Short, "zero-knowledge proof")
	})

	t.Run("requires exactly one argument", func(t *testing.T) {
		cmd := cli.GetCmdQueryVerifyProof()
		require.NotNil(t, cmd.Args)

		err := cmd.Args(cmd, []string{})
		require.Error(t, err)

		err = cmd.Args(cmd, []string{"proof1.json", "proof2.json"})
		require.Error(t, err)

		err = cmd.Args(cmd, []string{"proof.json"})
		require.NoError(t, err)
	})

	t.Run("has vkey-name flag", func(t *testing.T) {
		cmd := cli.GetCmdQueryVerifyProof()
		flags := cmd.Flags()

		vkeyNameFlag := flags.Lookup("vkey-name")
		require.NotNil(t, vkeyNameFlag)
		require.Equal(t, "Name of the verification key to use", vkeyNameFlag.Usage)
	})

	t.Run("has vkey-id flag", func(t *testing.T) {
		cmd := cli.GetCmdQueryVerifyProof()
		flags := cmd.Flags()

		vkeyIDFlag := flags.Lookup("vkey-id")
		require.NotNil(t, vkeyIDFlag)
		require.Equal(t, "ID of the verification key to use", vkeyIDFlag.Usage)
	})

	t.Run("has public-inputs flag", func(t *testing.T) {
		cmd := cli.GetCmdQueryVerifyProof()
		flags := cmd.Flags()

		publicInputsFlag := flags.Lookup("public-inputs")
		require.NotNil(t, publicInputsFlag)
		require.Equal(t, "Comma-separated list of public inputs", publicInputsFlag.Usage)
	})

	t.Run("has query flags", func(t *testing.T) {
		cmd := cli.GetCmdQueryVerifyProof()
		flags := cmd.Flags()

		nodeFlag := flags.Lookup("node")
		require.NotNil(t, nodeFlag)
	})

	t.Run("has long description", func(t *testing.T) {
		cmd := cli.GetCmdQueryVerifyProof()
		require.NotEmpty(t, cmd.Long)
		require.Contains(t, cmd.Long, "JSON-encoded")
		require.Contains(t, cmd.Long, "--vkey-name")
		require.Contains(t, cmd.Long, "--vkey-id")
	})

	t.Run("has example usage", func(t *testing.T) {
		cmd := cli.GetCmdQueryVerifyProof()
		require.NotEmpty(t, cmd.Example)
		require.Contains(t, cmd.Example, "proof.json")
		require.Contains(t, cmd.Example, "--vkey-name")
		require.Contains(t, cmd.Example, "--vkey-id")
		require.Contains(t, cmd.Example, "--public-inputs")
	})
}

// ============================================================================
// GetCmdQueryVerifyProof Extended Tests
// ============================================================================

func TestGetCmdQueryVerifyProofExtended(t *testing.T) {
	t.Run("RunE fails without client context", func(t *testing.T) {
		cmd := cli.GetCmdQueryVerifyProof()

		// Create a temporary proof file
		tmpDir := t.TempDir()
		proofFile := filepath.Join(tmpDir, "proof.json")
		err := os.WriteFile(proofFile, []byte(`{"pi_a": ["1", "2", "1"]}`), 0o600)
		require.NoError(t, err)

		// Set flags
		require.NoError(t, cmd.Flags().Set("vkey-name", "test"))
		require.NoError(t, cmd.Flags().Set("public-inputs", "1,2,3"))

		// Execute without proper client context - should fail
		cmd.SetArgs([]string{proofFile})
		err = cmd.Execute()
		require.Error(t, err)
	})

	t.Run("RunE fails with non-existent proof file", func(t *testing.T) {
		cmd := cli.GetCmdQueryVerifyProof()

		// Set flags
		require.NoError(t, cmd.Flags().Set("vkey-name", "test"))
		require.NoError(t, cmd.Flags().Set("public-inputs", "1,2,3"))

		// Execute with non-existent file
		cmd.SetArgs([]string{"/non/existent/proof.json"})
		err := cmd.Execute()
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to read proof file")
	})

	t.Run("RunE fails without vkey-name or vkey-id", func(t *testing.T) {
		cmd := cli.GetCmdQueryVerifyProof()

		// Create a temporary proof file
		tmpDir := t.TempDir()
		proofFile := filepath.Join(tmpDir, "proof.json")
		err := os.WriteFile(proofFile, []byte(`{"pi_a": ["1", "2", "1"]}`), 0o600)
		require.NoError(t, err)

		// Set public-inputs but not vkey-name or vkey-id
		require.NoError(t, cmd.Flags().Set("public-inputs", "1,2,3"))

		cmd.SetArgs([]string{proofFile})
		err = cmd.Execute()
		require.Error(t, err)
		require.Contains(t, err.Error(), "either --vkey-name or --vkey-id must be specified")
	})

	t.Run("RunE fails without public-inputs", func(t *testing.T) {
		cmd := cli.GetCmdQueryVerifyProof()

		// Create a temporary proof file
		tmpDir := t.TempDir()
		proofFile := filepath.Join(tmpDir, "proof.json")
		err := os.WriteFile(proofFile, []byte(`{"pi_a": ["1", "2", "1"]}`), 0o600)
		require.NoError(t, err)

		// Set vkey-name but not public-inputs
		require.NoError(t, cmd.Flags().Set("vkey-name", "test"))

		cmd.SetArgs([]string{proofFile})
		err = cmd.Execute()
		require.Error(t, err)
		require.Contains(t, err.Error(), "--public-inputs must be specified")
	})

	t.Run("RunE with vkey-id instead of vkey-name", func(t *testing.T) {
		cmd := cli.GetCmdQueryVerifyProof()

		// Create a temporary proof file
		tmpDir := t.TempDir()
		proofFile := filepath.Join(tmpDir, "proof.json")
		err := os.WriteFile(proofFile, []byte(`{"pi_a": ["1", "2", "1"]}`), 0o600)
		require.NoError(t, err)

		// Set vkey-id and public-inputs (should fail on client context, not validation)
		require.NoError(t, cmd.Flags().Set("vkey-id", "1"))
		require.NoError(t, cmd.Flags().Set("public-inputs", "1,2,3"))

		cmd.SetArgs([]string{proofFile})
		err = cmd.Execute()
		require.Error(t, err)
		// Should fail on client context, not on validation
		require.NotContains(t, err.Error(), "either --vkey-name or --vkey-id must be specified")
	})

	t.Run("RunE reads proof file successfully before failing on context", func(t *testing.T) {
		cmd := cli.GetCmdQueryVerifyProof()

		// Create a temporary proof file with valid JSON
		tmpDir := t.TempDir()
		proofFile := filepath.Join(tmpDir, "proof.json")
		proofContent := `{
			"pi_a": ["6043643433140642569280898259541128431907635878547614935681440820683038963792", "9992132192779112865958667381915120532497401445863381693125708878412867819429", "1"],
			"pi_b": [["857150703036151009004130834885577860944545321105272581149620288148902385440", "3313419972466342030467701882126850537491115446681093222335468857323210697295"], ["21712445344172795956102361993647268776674729003569584506047190630474625887295", "13180126619787644952475441454844294991198251669191962852459355269881478597074"], ["1", "0"]],
			"pi_c": ["5608874530415768909531379297509258028398465201351680955270584280524807563327", "12825389375859294537236568763270506206901646432644007343954893485864905401313", "1"],
			"protocol": "groth16",
			"curve": "bn128"
		}`
		err := os.WriteFile(proofFile, []byte(proofContent), 0o600)
		require.NoError(t, err)

		// Set all required flags
		require.NoError(t, cmd.Flags().Set("vkey-name", "email_auth"))
		require.NoError(t, cmd.Flags().Set("public-inputs", "1,2,3,4,5"))

		cmd.SetArgs([]string{proofFile})
		err = cmd.Execute()
		// Should fail on client context, but proof file was read successfully
		require.Error(t, err)
		require.NotContains(t, err.Error(), "failed to read proof file")
	})

	t.Run("public-inputs parsing with various formats", func(t *testing.T) {
		cmd := cli.GetCmdQueryVerifyProof()

		// Create a temporary proof file
		tmpDir := t.TempDir()
		proofFile := filepath.Join(tmpDir, "proof.json")
		err := os.WriteFile(proofFile, []byte(`{}`), 0o600)
		require.NoError(t, err)

		// Test with spaces in public inputs
		require.NoError(t, cmd.Flags().Set("vkey-name", "test"))
		require.NoError(t, cmd.Flags().Set("public-inputs", "1, 2, 3, 4"))

		cmd.SetArgs([]string{proofFile})
		err = cmd.Execute()
		// Should fail on client context, not on parsing
		require.Error(t, err)
	})

	t.Run("empty proof file", func(t *testing.T) {
		cmd := cli.GetCmdQueryVerifyProof()

		// Create an empty proof file
		tmpDir := t.TempDir()
		proofFile := filepath.Join(tmpDir, "empty.json")
		err := os.WriteFile(proofFile, []byte{}, 0o600)
		require.NoError(t, err)

		require.NoError(t, cmd.Flags().Set("vkey-name", "test"))
		require.NoError(t, cmd.Flags().Set("public-inputs", "1,2,3"))

		cmd.SetArgs([]string{proofFile})
		err = cmd.Execute()
		// Should fail (either on parsing or client context)
		require.Error(t, err)
	})
}

// ============================================================================
// ParsePublicInputs Tests
// ============================================================================

func TestParsePublicInputs(t *testing.T) {
	t.Run("empty string returns empty slice", func(t *testing.T) {
		result := cli.ParsePublicInputs("")
		require.Empty(t, result)
		require.NotNil(t, result)
	})

	t.Run("single value", func(t *testing.T) {
		result := cli.ParsePublicInputs("123")
		require.Len(t, result, 1)
		require.Equal(t, "123", result[0])
	})

	t.Run("multiple comma-separated values", func(t *testing.T) {
		result := cli.ParsePublicInputs("1,2,3,4")
		require.Len(t, result, 4)
		require.Equal(t, "1", result[0])
		require.Equal(t, "2", result[1])
		require.Equal(t, "3", result[2])
		require.Equal(t, "4", result[3])
	})

	t.Run("strips spaces", func(t *testing.T) {
		result := cli.ParsePublicInputs("1, 2, 3")
		require.Len(t, result, 3)
		require.Equal(t, "1", result[0])
		require.Equal(t, "2", result[1])
		require.Equal(t, "3", result[2])
	})

	t.Run("strips tabs", func(t *testing.T) {
		result := cli.ParsePublicInputs("1,\t2,\t3")
		require.Len(t, result, 3)
		require.Equal(t, "1", result[0])
		require.Equal(t, "2", result[1])
		require.Equal(t, "3", result[2])
	})

	t.Run("strips newlines", func(t *testing.T) {
		result := cli.ParsePublicInputs("1,\n2,\n3")
		require.Len(t, result, 3)
		require.Equal(t, "1", result[0])
		require.Equal(t, "2", result[1])
		require.Equal(t, "3", result[2])
	})

	t.Run("handles empty values between commas", func(t *testing.T) {
		result := cli.ParsePublicInputs("1,,2")
		require.Len(t, result, 2)
		require.Equal(t, "1", result[0])
		require.Equal(t, "2", result[1])
	})

	t.Run("handles trailing comma", func(t *testing.T) {
		result := cli.ParsePublicInputs("1,2,3,")
		require.Len(t, result, 3)
		require.Equal(t, "1", result[0])
		require.Equal(t, "2", result[1])
		require.Equal(t, "3", result[2])
	})

	t.Run("handles leading comma", func(t *testing.T) {
		result := cli.ParsePublicInputs(",1,2,3")
		require.Len(t, result, 3)
		require.Equal(t, "1", result[0])
		require.Equal(t, "2", result[1])
		require.Equal(t, "3", result[2])
	})

	t.Run("handles large numbers", func(t *testing.T) {
		result := cli.ParsePublicInputs("123456789012345678901234567890,987654321098765432109876543210")
		require.Len(t, result, 2)
		require.Equal(t, "123456789012345678901234567890", result[0])
		require.Equal(t, "987654321098765432109876543210", result[1])
	})

	t.Run("handles mixed whitespace", func(t *testing.T) {
		result := cli.ParsePublicInputs(" 1 , 2 , 3 ")
		require.Len(t, result, 3)
		require.Equal(t, "1", result[0])
		require.Equal(t, "2", result[1])
		require.Equal(t, "3", result[2])
	})

	t.Run("handles only whitespace between commas", func(t *testing.T) {
		result := cli.ParsePublicInputs("1,   ,2")
		require.Len(t, result, 2)
		require.Equal(t, "1", result[0])
		require.Equal(t, "2", result[1])
	})

	t.Run("preserves non-numeric characters", func(t *testing.T) {
		result := cli.ParsePublicInputs("abc,def,123")
		require.Len(t, result, 3)
		require.Equal(t, "abc", result[0])
		require.Equal(t, "def", result[1])
		require.Equal(t, "123", result[2])
	})

	t.Run("handles hex-like values", func(t *testing.T) {
		result := cli.ParsePublicInputs("0x123,0xabc,0xDEF")
		require.Len(t, result, 3)
		require.Equal(t, "0x123", result[0])
		require.Equal(t, "0xabc", result[1])
		require.Equal(t, "0xDEF", result[2])
	})

	t.Run("handles negative numbers", func(t *testing.T) {
		result := cli.ParsePublicInputs("-1,-2,-3")
		require.Len(t, result, 3)
		require.Equal(t, "-1", result[0])
		require.Equal(t, "-2", result[1])
		require.Equal(t, "-3", result[2])
	})

	t.Run("only whitespace returns empty", func(t *testing.T) {
		result := cli.ParsePublicInputs("   \t\n  ")
		require.Empty(t, result)
	})

	t.Run("only commas returns empty", func(t *testing.T) {
		result := cli.ParsePublicInputs(",,,")
		require.Empty(t, result)
	})

	t.Run("single character values", func(t *testing.T) {
		result := cli.ParsePublicInputs("a,b,c")
		require.Len(t, result, 3)
		require.Equal(t, "a", result[0])
		require.Equal(t, "b", result[1])
		require.Equal(t, "c", result[2])
	})

	t.Run("realistic public inputs", func(t *testing.T) {
		// Test with realistic ZK proof public inputs
		input := "6632353713085157925504008443078919716322386156160602218536961028046468237192,12057794547485210516928817874827048708844252651510875086257455163416697746512,0"
		result := cli.ParsePublicInputs(input)
		require.Len(t, result, 3)
		require.Equal(t, "6632353713085157925504008443078919716322386156160602218536961028046468237192", result[0])
		require.Equal(t, "12057794547485210516928817874827048708844252651510875086257455163416697746512", result[1])
		require.Equal(t, "0", result[2])
	})

	t.Run("multiple consecutive commas", func(t *testing.T) {
		result := cli.ParsePublicInputs("1,,,,,2")
		require.Len(t, result, 2)
		require.Equal(t, "1", result[0])
		require.Equal(t, "2", result[1])
	})

	t.Run("value with underscore", func(t *testing.T) {
		result := cli.ParsePublicInputs("email_auth,rollup_circuit")
		require.Len(t, result, 2)
		require.Equal(t, "email_auth", result[0])
		require.Equal(t, "rollup_circuit", result[1])
	})

	t.Run("value with dots", func(t *testing.T) {
		result := cli.ParsePublicInputs("1.234,5.678")
		require.Len(t, result, 2)
		require.Equal(t, "1.234", result[0])
		require.Equal(t, "5.678", result[1])
	})
}

// ============================================================================
// Flag Interaction Tests
// ============================================================================

func TestVerifyProofFlagInteractions(t *testing.T) {
	t.Run("vkey-id of 0 with no vkey-name fails", func(t *testing.T) {
		cmd := cli.GetCmdQueryVerifyProof()

		tmpDir := t.TempDir()
		proofFile := filepath.Join(tmpDir, "proof.json")
		err := os.WriteFile(proofFile, []byte(`{}`), 0o600)
		require.NoError(t, err)

		// vkey-id defaults to 0, and vkey-name is empty
		require.NoError(t, cmd.Flags().Set("public-inputs", "1,2,3"))

		cmd.SetArgs([]string{proofFile})
		err = cmd.Execute()
		require.Error(t, err)
		require.Contains(t, err.Error(), "either --vkey-name or --vkey-id must be specified")
	})

	t.Run("vkey-id non-zero without vkey-name succeeds validation", func(t *testing.T) {
		cmd := cli.GetCmdQueryVerifyProof()

		tmpDir := t.TempDir()
		proofFile := filepath.Join(tmpDir, "proof.json")
		err := os.WriteFile(proofFile, []byte(`{}`), 0o600)
		require.NoError(t, err)

		require.NoError(t, cmd.Flags().Set("vkey-id", "5"))
		require.NoError(t, cmd.Flags().Set("public-inputs", "1,2,3"))

		cmd.SetArgs([]string{proofFile})
		err = cmd.Execute()
		require.Error(t, err)
		// Should not fail on validation, but on client context
		require.NotContains(t, err.Error(), "either --vkey-name or --vkey-id must be specified")
	})

	t.Run("both vkey-name and vkey-id set", func(t *testing.T) {
		cmd := cli.GetCmdQueryVerifyProof()

		tmpDir := t.TempDir()
		proofFile := filepath.Join(tmpDir, "proof.json")
		err := os.WriteFile(proofFile, []byte(`{}`), 0o600)
		require.NoError(t, err)

		require.NoError(t, cmd.Flags().Set("vkey-name", "test"))
		require.NoError(t, cmd.Flags().Set("vkey-id", "5"))
		require.NoError(t, cmd.Flags().Set("public-inputs", "1,2,3"))

		cmd.SetArgs([]string{proofFile})
		err = cmd.Execute()
		// Should proceed (both set is valid)
		require.Error(t, err)
		require.NotContains(t, err.Error(), "either --vkey-name or --vkey-id must be specified")
	})
}

// ============================================================================
// Query Commands Integration Tests
// ============================================================================

func TestQueryCommandsIntegration(t *testing.T) {
	t.Run("all subcommands are accessible from parent", func(t *testing.T) {
		parentCmd := cli.GetQueryCmd()

		// Find vkey subcommand
		vkeyCmd, _, err := parentCmd.Find([]string{"vkey", "1"})
		require.NoError(t, err)
		require.NotNil(t, vkeyCmd)

		// Find vkey-by-name subcommand
		vkeyByNameCmd, _, err := parentCmd.Find([]string{"vkey-by-name", "test"})
		require.NoError(t, err)
		require.NotNil(t, vkeyByNameCmd)

		// Find vkeys subcommand
		vkeysCmd, _, err := parentCmd.Find([]string{"vkeys"})
		require.NoError(t, err)
		require.NotNil(t, vkeysCmd)

		// Find has-vkey subcommand
		hasVKeyCmd, _, err := parentCmd.Find([]string{"has-vkey", "test"})
		require.NoError(t, err)
		require.NotNil(t, hasVKeyCmd)

		// Find verify-proof subcommand
		verifyProofCmd, _, err := parentCmd.Find([]string{"verify-proof", "proof.json"})
		require.NoError(t, err)
		require.NotNil(t, verifyProofCmd)
	})

	t.Run("subcommands have unique uses", func(t *testing.T) {
		parentCmd := cli.GetQueryCmd()
		subcommands := parentCmd.Commands()

		uses := make(map[string]bool)
		for _, subcmd := range subcommands {
			require.False(t, uses[subcmd.Use], "Duplicate use: %s", subcmd.Use)
			uses[subcmd.Use] = true
		}
	})
}
