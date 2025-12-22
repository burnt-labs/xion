package cli_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/burnt-labs/xion/x/dkim/client/cli"
	"github.com/burnt-labs/xion/x/dkim/keeper"
	"github.com/burnt-labs/xion/x/dkim/types"
	zkkeeper "github.com/burnt-labs/xion/x/zk/keeper"
)

type IntegrationTestSuite struct {
	ctx         sdk.Context
	keeper      keeper.Keeper
	zkeeper     zkkeeper.Keeper
	queryServer types.QueryServer
	clientCtx   client.Context
	cdc         codec.Codec
}

func setupIntegrationTest(t *testing.T) *IntegrationTestSuite {
	t.Helper()

	suite := &IntegrationTestSuite{}

	// Setup interface registry
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(interfaceRegistry)

	// Create codec
	suite.cdc = codec.NewProtoCodec(interfaceRegistry)

	// Setup store
	key := storetypes.NewKVStoreKey(types.ModuleName)
	storeService := runtime.NewKVStoreService(key)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))
	suite.ctx = testCtx.Ctx

	// Create keeper
	govModAddr := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	logger := log.NewTestLogger(t)
	suite.zkeeper = zkkeeper.NewKeeper(suite.cdc, storeService, logger, govModAddr)
	suite.keeper = keeper.NewKeeper(suite.cdc, storeService, logger, govModAddr, suite.zkeeper)

	// Create query server
	suite.queryServer = keeper.NewQuerier(suite.keeper)

	// Setup client context
	out := new(bytes.Buffer)
	suite.clientCtx = client.Context{}.
		WithCodec(suite.cdc).
		WithInterfaceRegistry(interfaceRegistry).
		WithOutput(out).
		WithOutputFormat(flags.OutputFormatJSON)

	return suite
}

func TestIntegration_ParseDkimPubKeysFlags_WithCommand(t *testing.T) {
	suite := setupIntegrationTest(t)

	// Create the full command to test flag parsing with proper setup
	cmd := cli.GetDkimPublicKeys()
	require.NotNil(t, cmd)

	// Setup client context
	cmd.SetContext(context.Background())
	require.NoError(t, client.SetCmdClientContext(cmd, suite.clientCtx))

	// Set flags
	require.NoError(t, cmd.Flags().Set("domain", "example.com"))
	require.NoError(t, cmd.Flags().Set("selector", "selector1"))
	require.NoError(t, cmd.Flags().Set("hash", "hash123"))

	// Parse flags using our helper
	domain, selector, hash, err := cli.ParseDkimPubKeysFlags(cmd)
	require.NoError(t, err)
	require.Equal(t, "example.com", domain)
	require.Equal(t, "selector1", selector)
	require.Equal(t, "hash123", hash)

	// Verify the command itself can start executing
	// It will fail at gRPC connection, but that's expected
	err = cmd.RunE(cmd, []string{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "dial", "Should fail on gRPC dial")
}

func TestIntegration_GetDkimPublicKey(t *testing.T) {
	suite := setupIntegrationTest(t)

	// Create command
	cmd := cli.GetDkimPublicKey()
	require.NotNil(t, cmd)

	// Setup client context on command
	cmd.SetContext(context.Background())
	require.NoError(t, client.SetCmdClientContext(cmd, suite.clientCtx))

	// Test with valid arguments (domain and selector)
	// The command will fail at gRPC connection, but we'll execute more of the RunE
	err := cmd.RunE(cmd, []string{"example.com", "default"})
	require.Error(t, err) // Expected - no gRPC connection

	// Verify the error is about gRPC/connection, not about argument parsing or context
	require.Contains(t, err.Error(), "dial", "Should fail on gRPC dial, not earlier")
}

func TestIntegration_GetDkimPublicKey_ArgValidation(t *testing.T) {
	// Test argument validation
	cmd := cli.GetDkimPublicKey()
	require.NotNil(t, cmd)

	// cobra.ExactArgs(2) should require exactly 2 arguments
	t.Run("with 0 args should fail", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		require.Error(t, err)
	})

	t.Run("with 1 arg should fail", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"arg1"})
		require.Error(t, err)
	})

	t.Run("with 2 args should pass", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"domain", "selector"})
		require.NoError(t, err)
	})

	t.Run("with 3 args should fail", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"arg1", "arg2", "arg3"})
		require.Error(t, err)
	})
}

func TestIntegration_CommandStructure(t *testing.T) {
	// Test that commands are properly structured
	tests := []struct {
		name       string
		createCmd  func() *cobra.Command
		checkFlags []string
	}{
		{
			name:      "GetDkimPublicKey",
			createCmd: cli.GetDkimPublicKey,
			checkFlags: []string{
				flags.FlagOutput,
			},
		},
		{
			name:      "GetDkimPublicKeys",
			createCmd: cli.GetDkimPublicKeys,
			checkFlags: []string{
				flags.FlagOutput,
				"domain",
				"selector",
				"hash",
			},
		},
		{
			name:      "GetParams",
			createCmd: cli.GetParams,
			checkFlags: []string{
				flags.FlagOutput,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.createCmd()
			require.NotNil(t, cmd)
			require.NotNil(t, cmd.RunE)

			// Verify flags exist
			for _, flagName := range tt.checkFlags {
				flag := cmd.Flags().Lookup(flagName)
				require.NotNil(t, flag, fmt.Sprintf("Flag %s should exist", flagName))
			}
		})
	}
}

func TestIntegration_GetParams(t *testing.T) {
	suite := setupIntegrationTest(t)

	// Create command
	cmd := cli.GetParams()
	require.NotNil(t, cmd)

	// Setup client context on command
	cmd.SetContext(context.Background())
	require.NoError(t, client.SetCmdClientContext(cmd, suite.clientCtx))

	// Test with no arguments (params takes no args)
	// The command will fail at gRPC connection, but we'll execute more of the RunE
	err := cmd.RunE(cmd, []string{})
	require.Error(t, err) // Expected - no gRPC connection

	// Verify the error is about gRPC/connection, not about argument parsing or context
	require.Contains(t, err.Error(), "dial", "Should fail on gRPC dial, not earlier")
}

func TestIntegration_GetParams_ArgValidation(t *testing.T) {
	// Test argument validation
	cmd := cli.GetParams()
	require.NotNil(t, cmd)

	// cobra.NoArgs should reject any arguments
	t.Run("with 0 args should pass", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		require.NoError(t, err)
	})

	t.Run("with 1 arg should fail", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"arg1"})
		require.Error(t, err)
	})

	t.Run("with 2 args should fail", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"arg1", "arg2"})
		require.Error(t, err)
	})
}

func TestIntegration_GenerateDkimPublicKey(t *testing.T) {
	suite := setupIntegrationTest(t)

	// Create command
	cmd := cli.GenerateDkimPublicKey()
	require.NotNil(t, cmd)

	// Setup client context on command
	cmd.SetContext(context.Background())
	require.NoError(t, client.SetCmdClientContext(cmd, suite.clientCtx))

	// Test with valid arguments (domain and selector)
	// The command will fail at DNS lookup, but we'll execute more of the RunE
	err := cmd.RunE(cmd, []string{"example.com", "selector1"})
	require.Error(t, err) // Expected - DNS lookup failure for fake domain
}

func TestIntegration_GenerateDkimPublicKey_ArgValidation(t *testing.T) {
	// Test argument validation
	cmd := cli.GenerateDkimPublicKey()
	require.NotNil(t, cmd)

	// cobra.ExactArgs(2) should require exactly 2 arguments
	t.Run("with 0 args should fail", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		require.Error(t, err)
	})

	t.Run("with 1 arg should fail", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"domain"})
		require.Error(t, err)
	})

	t.Run("with 2 args should pass", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"domain", "selector"})
		require.NoError(t, err)
	})

	t.Run("with 3 args should fail", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"domain", "selector", "extra"})
		require.Error(t, err)
	})
}

func TestIntegration_MsgRevokeDkimPubKey(t *testing.T) {
	suite := setupIntegrationTest(t)

	// Create command
	cmd := cli.MsgRevokeDkimPubKey()
	require.NotNil(t, cmd)

	// Setup client context on command
	cmd.SetContext(context.Background())
	require.NoError(t, client.SetCmdClientContext(cmd, suite.clientCtx))

	// Test with valid arguments (domain and private key)
	// The command will fail at getting tx context, but we'll execute the RunE
	err := cmd.RunE(cmd, []string{"example.com", "MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQC..."})
	require.Error(t, err) // Expected - no keyring/from address
}

func TestIntegration_MsgRevokeDkimPubKey_ArgValidation(t *testing.T) {
	// Test argument validation
	cmd := cli.MsgRevokeDkimPubKey()
	require.NotNil(t, cmd)

	// cobra.ExactArgs(2) should require exactly 2 arguments
	t.Run("with 0 args should fail", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		require.Error(t, err)
	})

	t.Run("with 1 arg should fail", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"domain"})
		require.Error(t, err)
	})

	t.Run("with 2 args should pass", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"domain", "privkey"})
		require.NoError(t, err)
	})

	t.Run("with 3 args should fail", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"domain", "privkey", "extra"})
		require.Error(t, err)
	})
}

func TestIntegration_NewTxDecodeCmd(t *testing.T) {
	suite := setupIntegrationTest(t)

	// Create command
	cmd := cli.NewTxDecodeCmd()
	require.NotNil(t, cmd)

	// Setup client context on command
	cmd.SetContext(context.Background())
	require.NoError(t, client.SetCmdClientContext(cmd, suite.clientCtx))

	// Test with valid base64 argument
	// The command will fail because TxConfig is not set in the test context
	err := cmd.RunE(cmd, []string{"SGVsbG8gV29ybGQ="})
	require.Error(t, err) // Expected - TxConfig not initialized
}

func TestIntegration_NewTxDecodeCmd_ArgValidation(t *testing.T) {
	// Test argument validation
	cmd := cli.NewTxDecodeCmd()
	require.NotNil(t, cmd)

	// cobra.ExactArgs(1) should require exactly 1 argument
	t.Run("with 0 args should fail", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		require.Error(t, err)
	})

	t.Run("with 1 arg should pass", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"base64string"})
		require.NoError(t, err)
	})

	t.Run("with 2 args should fail", func(t *testing.T) {
		err := cmd.Args(cmd, []string{"arg1", "arg2"})
		require.Error(t, err)
	})
}

func TestIntegration_TxCommandStructure(t *testing.T) {
	// Test that tx commands are properly structured
	tests := []struct {
		name       string
		createCmd  func() *cobra.Command
		checkFlags []string
	}{
		{
			name:      "MsgRevokeDkimPubKey",
			createCmd: cli.MsgRevokeDkimPubKey,
			checkFlags: []string{
				flags.FlagFrom,
			},
		},
		{
			name:      "NewTxDecodeCmd",
			createCmd: cli.NewTxDecodeCmd,
			checkFlags: []string{
				flags.FlagOutput,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.createCmd()
			require.NotNil(t, cmd)
			require.NotNil(t, cmd.RunE)

			// Verify flags exist
			for _, flagName := range tt.checkFlags {
				flag := cmd.Flags().Lookup(flagName)
				require.NotNil(t, flag, fmt.Sprintf("Flag %s should exist", flagName))
			}
		})
	}
}

func TestIntegration_NewTxCmd(t *testing.T) {
	// Test the parent tx command structure
	cmd := cli.NewTxCmd()
	require.NotNil(t, cmd)

	// Verify subcommands exist
	subCmds := cmd.Commands()
	require.GreaterOrEqual(t, len(subCmds), 2, "Should have at least 2 subcommands")

	// Check for expected subcommands
	cmdNames := make([]string, len(subCmds))
	for i, c := range subCmds {
		cmdNames[i] = c.Name()
	}

	require.Contains(t, cmdNames, "revoke-dkim", "Should have revoke-dkim subcommand")
	require.Contains(t, cmdNames, "decode", "Should have decode subcommand")
}
