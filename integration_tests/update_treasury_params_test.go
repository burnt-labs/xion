package integration_tests

import (
	"context"
	"encoding/json"
	"os"
	"path"
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/types"
	ibctest "github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
)

func TestUpdateTreasuryContractParams(t *testing.T) {
	// Setup Xion chain
	td := BuildXionChain(t, "0.0uxion", nil)
	xion, ctx := td.xionChain, td.ctx

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
	userAddrStr := xionUser.FormattedAddress()
	instantiateMsg := TreasuryInstantiateMsg{
		Admin:        &userAddrStr, // Set the user as admin (pointer)
		TypeUrls:     []string{},
		GrantConfigs: []GrantConfig{},
		FeeConfig: &FeeConfig{
			Description: "test fee grant",
		},
		Params: &Params{
			RedirectURL: "https://example.com",
			IconURL:     "https://example.com/icon.png",
			Metadata:    "{}",
		},
	}
	instantiateMsgStr, err := json.Marshal(instantiateMsg)
	require.NoError(t, err)

	contractAddr, err := xion.InstantiateContract(ctx, xionUser.KeyName(), codeIDStr, string(instantiateMsgStr), true)
	require.NoError(t, err)
	t.Logf("Deployed and instantiated contract at address: %s", contractAddr)

	// Update params using direct contract execution
	t.Log("Updating contract parameters")
	updateParamsMsg := map[string]interface{}{
		"update_params": map[string]interface{}{
			"params": Params{
				RedirectURL: "https://example.com/redirect",
				IconURL:     "https://example.com/icon.png",
				Metadata:    `{"updated": "true"}`,
			},
		},
	}
	updateParamsMsgBz, err := json.Marshal(updateParamsMsg)
	require.NoError(t, err)

	// Execute the update params message
	_, err = xion.ExecuteContract(ctx, xionUser.KeyName(), contractAddr, string(updateParamsMsgBz))
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

	var queriedParams Params
	err = json.Unmarshal(queryBytes, &queriedParams)
	require.NoError(t, err)

	// Validate the updated contract state
	require.Equal(t, "https://example.com/redirect", queriedParams.RedirectURL)
	require.Equal(t, "https://example.com/icon.png", queriedParams.IconURL)
	require.Equal(t, `{"updated": "true"}`, queriedParams.Metadata)
}
