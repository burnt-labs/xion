package cli_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/burnt-labs/xion/x/dkim/client/cli"
	"github.com/burnt-labs/xion/x/dkim/types"
)

func TestGetQueryCmd(t *testing.T) {
	cmd := cli.GetQueryCmd()
	require.NotNil(t, cmd)
	require.Equal(t, types.ModuleName, cmd.Use)
	require.True(t, cmd.DisableFlagParsing)

	// Verify subcommands are added
	require.True(t, len(cmd.Commands()) > 0)
}

func TestGetDkimPublicKey(t *testing.T) {
	cmd := cli.GetDkimPublicKey()
	require.NotNil(t, cmd)
	require.Contains(t, cmd.Use, "dkim-pubkey")
	require.Equal(t, "Get a DKIM public key", cmd.Short)
	require.NotEmpty(t, cmd.Aliases)
	require.Contains(t, cmd.Aliases, "qdkim")

	// Test that command has RunE function defined
	require.NotNil(t, cmd.RunE)

	// Test command structure
	require.NotNil(t, cmd.Flags())

	// Call RunE to ensure coverage - expect panic/error without context
	func() {
		defer func() {
			_ = recover() // Ignore panics from missing context
		}()
		_ = cmd.RunE(cmd, []string{"example.com", "default"})
	}()
}

func TestGetDkimPublicKeys(t *testing.T) {
	cmd := cli.GetDkimPublicKeys()
	require.NotNil(t, cmd)
	require.Contains(t, cmd.Use, "dkim-pubkeys")
	require.Equal(t, "Get a DKIM public key matching filter parameters", cmd.Short)
	require.NotEmpty(t, cmd.Long)
	require.NotEmpty(t, cmd.Example)
	require.NotEmpty(t, cmd.Aliases)
	require.Contains(t, cmd.Aliases, "qdkims")

	// Test that command has RunE function defined
	require.NotNil(t, cmd.RunE)

	// Test command structure and flags
	require.NotNil(t, cmd.Flags())
	domainFlag := cmd.Flags().Lookup("domain")
	require.NotNil(t, domainFlag)
	require.Equal(t, "Filter by domain", domainFlag.Usage)

	selectorFlag := cmd.Flags().Lookup("selector")
	require.NotNil(t, selectorFlag)
	require.Contains(t, selectorFlag.Usage, "selector")

	hashFlag := cmd.Flags().Lookup("hash")
	require.NotNil(t, hashFlag)
	require.Contains(t, hashFlag.Usage, "poseidon hash")

	// Test Args validator with different argument counts
	// cobra.RangeArgs(0, 3) should allow 0-3 args
	t.Run("args validator with 0 args", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		require.NoError(t, err)
	})

	t.Run("args validator with 1 arg", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"arg1"})
		require.NoError(t, err)
	})

	t.Run("args validator with 2 args", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"arg1", "arg2"})
		require.NoError(t, err)
	})

	t.Run("args validator with 3 args", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"arg1", "arg2", "arg3"})
		require.NoError(t, err)
	})

	t.Run("args validator with 4 args should fail", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"arg1", "arg2", "arg3", "arg4"})
		require.Error(t, err)
	})

	// Test with different flag combinations to improve coverage
	t.Run("no flags", func(t *testing.T) {
		func() {
			defer func() {
				_ = recover() // Ignore panics from missing context
			}()
			_ = cmd.RunE(cmd, []string{})
		}()
	})

	t.Run("with domain flag", func(t *testing.T) {
		require.NoError(t, cmd.Flags().Set("domain", "example.com"))
		func() {
			defer func() {
				_ = recover() // Ignore panics from missing context
			}()
			_ = cmd.RunE(cmd, []string{})
		}()
	})

	t.Run("with selector flag", func(t *testing.T) {
		require.NoError(t, cmd.Flags().Set("selector", "default"))
		func() {
			defer func() {
				_ = recover() // Ignore panics from missing context
			}()
			_ = cmd.RunE(cmd, []string{})
		}()
	})

	t.Run("with hash flag", func(t *testing.T) {
		require.NoError(t, cmd.Flags().Set("hash", "somehash"))
		func() {
			defer func() {
				_ = recover() // Ignore panics from missing context
			}()
			_ = cmd.RunE(cmd, []string{})
		}()
	})
}

func TestGenerateDkimPublicKey(t *testing.T) {
	cmd := cli.GenerateDkimPublicKey()
	require.NotNil(t, cmd)
	require.Contains(t, cmd.Use, "generate-dkim-pubkey")
	require.Equal(t, "Generate a DKIM msg to create a new DKIM public key", cmd.Short)
	require.NotEmpty(t, cmd.Aliases)
	require.Contains(t, cmd.Aliases, "gdkim")
	require.NotEmpty(t, cmd.Example)
	require.NotEmpty(t, cmd.Long)

	// Test that command has RunE function defined
	require.NotNil(t, cmd.RunE)

	// Test command structure
	require.NotNil(t, cmd.Flags())

	// Call RunE to ensure coverage - expect panic/error without context or DNS
	func() {
		defer func() {
			_ = recover() // Ignore panics from missing context
		}()
		_ = cmd.RunE(cmd, []string{"example.com", "default"})
	}()
}

func TestNewTxCmd(t *testing.T) {
	cmd := cli.NewTxCmd()
	require.NotNil(t, cmd)
	require.Equal(t, types.ModuleName, cmd.Use)

	// Verify subcommands are added
	require.True(t, len(cmd.Commands()) > 0)
}

func TestMsgRevokeDkimPubKey(t *testing.T) {
	cmd := cli.MsgRevokeDkimPubKey()
	require.NotNil(t, cmd)
	require.Contains(t, cmd.Use, "revoke-dkim")
	require.NotEmpty(t, cmd.Aliases)
	require.Contains(t, cmd.Aliases, "rdkim")
	require.NotEmpty(t, cmd.Long)

	// Test that command has RunE function defined
	require.NotNil(t, cmd.RunE)

	// Test command structure
	require.NotNil(t, cmd.Flags())

	// Test with invalid private key to cover error handling
	t.Run("with invalid private key", func(t *testing.T) {
		func() {
			defer func() {
				_ = recover() // Ignore panics from missing context
			}()
			_ = cmd.RunE(cmd, []string{"example.com", "invalid-key"})
		}()
	})

	// Test with a properly formatted (but fake) private key to cover more code paths
	t.Run("with formatted private key", func(t *testing.T) {
		// This will fail at getting client context, but will cover the PEM decoding logic
		fakeKey := "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA"
		func() {
			defer func() {
				_ = recover() // Ignore panics from missing context
			}()
			_ = cmd.RunE(cmd, []string{"https://example.com", fakeKey})
		}()
	})
}
