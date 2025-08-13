package cli

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/client/flags"
)

func TestGetQueryCmd(t *testing.T) {
	cmd := GetQueryCmd()
	require.NotNil(t, cmd)
	require.Equal(t, "globalfee", cmd.Use)
	require.Equal(t, "Querying commands for the global fee module", cmd.Short)
	require.True(t, cmd.DisableFlagParsing)
	require.Equal(t, 2, cmd.SuggestionsMinimumDistance)
	require.NotNil(t, cmd.RunE)

	// Check that it has the params subcommand
	subcommands := cmd.Commands()
	require.Len(t, subcommands, 1)
	require.Equal(t, "params", subcommands[0].Use)
}

func TestGetCmdShowGlobalFeeParams(t *testing.T) {
	cmd := GetCmdShowGlobalFeeParams()
	require.NotNil(t, cmd)
	require.Equal(t, "params", cmd.Use)
	require.Equal(t, "Show globalfee params", cmd.Short)
	require.Equal(t, "Show globalfee requirement: minimum_gas_prices, bypass_min_fee_msg_types, max_total_bypass_minFee_msg_gas_usage", cmd.Long)
	require.NotNil(t, cmd.RunE)

	// Test that it expects exactly 0 args by testing the function behavior
	require.NotNil(t, cmd.Args)
	// Test with 0 args (should pass)
	err := cmd.Args(cmd, []string{})
	require.NoError(t, err)
	// Test with 1 arg (should fail)
	err = cmd.Args(cmd, []string{"extra"})
	require.Error(t, err)

	// Test that flags are added
	flagSet := cmd.Flags()
	require.NotNil(t, flagSet)

	// Check that query flags are added
	flag := flagSet.Lookup(flags.FlagOutput)
	require.NotNil(t, flag)
}

func TestGetCmdShowGlobalFeeParamsExecution(t *testing.T) {
	cmd := GetCmdShowGlobalFeeParams()

	// Test that command has no arguments validator
	require.NotNil(t, cmd.Args)

	// Test execution with args (should return error about no arguments)
	err := cmd.Args(cmd, []string{"extra"})
	require.Error(t, err)
}
