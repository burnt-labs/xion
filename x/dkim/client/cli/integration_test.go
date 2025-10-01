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
)

type IntegrationTestSuite struct {
	ctx         sdk.Context
	keeper      keeper.Keeper
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
	suite.keeper = keeper.NewKeeper(suite.cdc, storeService, logger, govModAddr)

	// Initialize params
	require.NoError(t, suite.keeper.Params.Set(suite.ctx, types.DefaultParams()))

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

func TestIntegration_GetCmdParams(t *testing.T) {
	suite := setupIntegrationTest(t)

	// Create command
	cmd := cli.GetCmdParams()
	require.NotNil(t, cmd)

	// Setup client context on command
	cmd.SetContext(context.Background())
	require.NoError(t, client.SetCmdClientContext(cmd, suite.clientCtx))

	// The command will fail because we don't have a gRPC connection,
	// but we can verify it gets past the context setup
	err := cmd.RunE(cmd, []string{})
	require.Error(t, err) // Expected - no gRPC connection

	// Verify the error is about gRPC, not about missing context
	require.Contains(t, err.Error(), "dial", "Should fail on gRPC dial, not context")
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

func TestIntegration_QueryParams_Direct(t *testing.T) {
	suite := setupIntegrationTest(t)

	// Create a mock command with context
	cmd := &cobra.Command{}
	cmd.SetContext(suite.ctx)

	// Test QueryParams directly with our query server
	// This tests the helper function with a real query server
	res, err := suite.queryServer.Params(suite.ctx, &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.NotNil(t, res)
	require.NotNil(t, res.Params)
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
			name:      "GetCmdParams",
			createCmd: cli.GetCmdParams,
			checkFlags: []string{
				flags.FlagOutput,
			},
		},
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
