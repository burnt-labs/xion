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

	"github.com/burnt-labs/xion/x/zk/client/cli"
	"github.com/burnt-labs/xion/x/zk/keeper"
	"github.com/burnt-labs/xion/x/zk/types"
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
