package integration_tests

import (
	"context"
	"encoding/json"
	"os"
	"path"
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/types"
	ibctest "github.com/strangelove-ventures/interchaintest/v10"
	"github.com/strangelove-ventures/interchaintest/v10/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v10/testutil"
	"github.com/stretchr/testify/require"
)

func TestUpdateTreasuryContractParams(t *testing.T) {
	ctx := t.Context()
	// Setup Xion chain
	xion := BuildXionChain(t)

	config := types.GetConfig()
	config.SetBech32PrefixForAccount("xion", "xionpub")

	// Create and fund test user
	t.Log("Creating and funding user accounts")
	fundAmount := math.NewInt(10_000_000)
	users := ibctest.GetAndFundTestUsers(t, ctx, "default", fundAmount, xion)
	xionUser := users[0]

	err := testutil.WaitForBlocks(ctx, 2, xion)
	require.NoError(t, err)

	// Deploy contract
	t.Log("Deploying contract")
	fp, err := os.Getwd()
	require.NoError(t, err)

	codeIDStr, err := xion.StoreContract(ctx, xionUser.FormattedAddress(),
		path.Join(fp, "integration_tests", "testdata", "contracts", "treasury-aarch64.wasm"))
	require.NoError(t, err)

	// Instantiate contract
	t.Log("Instantiating contract")
	instantiateMsg := TreasuryInstantiateMsg{
		TypeUrls:     []string{},
		GrantConfigs: []GrantConfig{},
		FeeConfig: &FeeConfig{
			Description: "test fee grant",
		},
	}
	instantiateMsgStr, err := json.Marshal(instantiateMsg)
	require.NoError(t, err)

	contractAddr, err := xion.InstantiateContract(ctx, xionUser.KeyName(), codeIDStr, string(instantiateMsgStr), true)
	require.NoError(t, err)
	t.Logf("Deployed and instantiated contract at address: %s", contractAddr)

	// CLI command to update params
	t.Log("Updating contract parameters")
	cmd := []string{
		"xion", "update-params", contractAddr,
		"https://example.com/display",
		"https://example.com/redirect",
		"https://example.com/icon.png",
		"--chain-id", xion.Config().ChainID,
		"--from", xionUser.KeyName(),
		"--gas-prices", "1uxion",
		"--gas-adjustment", "1.4",
		"--gas", "400000",
		"-y",
	}

	_, err = ExecTx(t, ctx, xion.GetNode(), xionUser.KeyName(), cmd...)
	require.NoError(t, err)

	// Wait for the transaction to be included in a block
	err = testutil.WaitForBlocks(ctx, 2, xion)
	require.NoError(t, err)

	// Query and validate contract state
	validateUpdatedParams(t, ctx, xion, contractAddr)
}

// **Validation Function**
func validateUpdatedParams(t *testing.T, ctx context.Context, xion *cosmos.CosmosChain, contractAddr string) {
	t.Log("Querying updated contract parameters")
	queryMsg := map[string]interface{}{
		"params": struct{}{},
	}
	queryMsgStr, err := json.Marshal(queryMsg)
	require.NoError(t, err)

	queryRaw, err := ExecQuery(t, ctx, xion.GetNode(), "wasm", "contract-state", "smart", contractAddr, string(queryMsgStr))
	require.NoError(t, err)

	queryBytes, err := json.Marshal(queryRaw["data"])
	require.NoError(t, err)

	var queriedParams struct {
		DisplayURL  string `json:"display_url"`
		RedirectURL string `json:"redirect_url"`
		IconURL     string `json:"icon_url"`
	}
	err = json.Unmarshal(queryBytes, &queriedParams)
	require.NoError(t, err)

	// Validate the updated contract state
	require.Equal(t, "https://example.com/display", queriedParams.DisplayURL)
	require.Equal(t, "https://example.com/redirect", queriedParams.RedirectURL)
	require.Equal(t, "https://example.com/icon.png", queriedParams.IconURL)
}
