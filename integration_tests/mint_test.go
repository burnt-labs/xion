package integration_tests

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/icza/dyno"
	"github.com/strangelove-ventures/interchaintest/v7/testutil"
	"github.com/stretchr/testify/require"
)

func TestMintModuleNoInflationNoFees(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()

	xion, ctx := BuildXionChain(t)
	// Get the distribution module account address
	var moduleAccount map[string]interface{}
	// Query the distribution module account
	queryRes, _, err := xion.FullNodes[0].ExecQuery(ctx, "auth", "module-account", "distribution")
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(queryRes, &moduleAccount))
	moduleAccountAddress, err := dyno.GetString(moduleAccount, "account", "base_account", "address")
	require.NoError(t, err)
	// Get the distribution module account balance
	initialModuleAccountBalance, err := xion.GetBalance(ctx, moduleAccountAddress, xion.Config().Denom)
	require.NoError(t, err)
	t.Logf("Initial distribution address balance: %d", initialModuleAccountBalance)

	// Query the mint module for the current inflation
	var inflation json.Number
	queryRes, _, err = xion.FullNodes[0].ExecQuery(ctx, "mint", "inflation")
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(queryRes, &inflation))
	inflationValue, err := inflation.Float64()
	t.Logf("Current inflation: %f", inflationValue)
	require.NoError(t, err, "inflation should be a float")
	// Make sure inflation is 0
	require.Equal(t, 0.0, inflationValue)

	// Query the mint module for inflation rate change
	var params = make(map[string]interface{})
	queryRes, _, err = xion.FullNodes[0].ExecQuery(ctx, "mint", "params")
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(queryRes, &params))
	inflationRateChange, err := dyno.GetString(params, "inflation_rate_change")
	require.NoError(t, err, "inflation_rate_change should be a string")
	inflationRateChangeValue, err := strconv.ParseFloat(inflationRateChange, 64)
	require.NoError(t, err, "inflation_rate_change should be convertible to float")
	t.Logf("Current inflation rate change: %f", inflationRateChangeValue)
	// Make sure inflation rate change is 0
	require.Equal(t, 0.0, inflationRateChangeValue)

	// Get the total bank supply
	jsonRes := make(map[string]interface{})
	queryRes, _, err = xion.FullNodes[0].ExecQuery(ctx, "bank", "total")
	require.NoError(t, err)

	require.NoError(t, json.Unmarshal(queryRes, &jsonRes))

	// Presuming we are the only denom on the chain
	totalSupply, err := dyno.GetSlice(jsonRes, "supply")
	require.NoError(t, err)
	xionCoin := totalSupply[0]
	require.NotEmpty(t, xionCoin)
	// Make sure we selected the uxion denom
	xionCoinDenom, err := dyno.GetString(xionCoin, "denom")
	require.NoError(t, err)
	require.Equal(t, xionCoinDenom, xion.Config().Denom)
	initialXionSupply, err := dyno.GetString(xionCoin, "amount")
	require.NoError(t, err)
	t.Logf("Initial Xion supply: %s", initialXionSupply)

	// Wait for some blocks and check if that supply stays the same
	chainHeight, _ := xion.Height(ctx)
	err = testutil.WaitForBlocks(ctx, int(chainHeight)+10, xion)
	require.NoError(t, err)

	// Get the distribution module account balance
	currentModuleAccountBalance, err := xion.GetBalance(ctx, moduleAccountAddress, xion.Config().Denom)
	require.NoError(t, err)
	t.Logf("Current distribution address balance: %d", currentModuleAccountBalance)

	// Get the total bank supply
	currentResJson := make(map[string]interface{})
	currentSupplyRes, _, queryErr := xion.FullNodes[0].ExecQuery(ctx, "bank", "total")
	require.NoError(t, queryErr)
	require.NoError(t, json.Unmarshal(currentSupplyRes, &currentResJson))

	newTotalSupply, err := dyno.GetSlice(currentResJson, "supply")
	require.NoError(t, err)
	currentXionCoin := newTotalSupply[0]

	currentXionSupply, err := dyno.GetString(currentXionCoin, "amount")
	require.NoError(t, err)
	t.Logf("Current Xion supply: %s", currentXionSupply)

	require.Equal(t, initialXionSupply, currentXionSupply)
	require.Equal(t, initialModuleAccountBalance, currentModuleAccountBalance)
}
