package e2e_app

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/burnt-labs/xion/e2e_tests/testlib"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/types"
	ibctest "github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/testutil"
	"github.com/stretchr/testify/require"
)

func TestAppUpdateTreasuryParams(t *testing.T) {
	ctx := t.Context()
	// Setup Xion chain
	xion := testlib.BuildXionChain(t)

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

	codeIDStr, err := xion.StoreContract(ctx, xionUser.FormattedAddress(),
		testlib.IntegrationTestPath("testdata", "contracts", "treasury-aarch64.wasm"))
	require.NoError(t, err)

	// Instantiate contract
	t.Log("Instantiating contract")
	accAddrStr := xionUser.FormattedAddress()
	instantiateMsg := TreasuryInstantiateMsg{
		Admin:        &accAddrStr,
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

	// CLI command to update params
	t.Log("Updating contract parameters")
	cmd := []string{
		"xion", "update-params", contractAddr,
		"https://example.com/redirect",
		"https://example.com/icon.png",
		"--chain-id", xion.Config().ChainID,
		"--from", xionUser.KeyName(),
		"--gas-prices", "1uxion",
		"--gas-adjustment", "1.4",
		"--gas", "400000",
		"-y",
	}

	_, err = testlib.ExecTx(t, ctx, xion.GetNode(), xionUser.KeyName(), cmd...)
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
	queryMsg := map[string]any{
		"params": struct{}{},
	}
	queryMsgStr, err := json.Marshal(queryMsg)
	require.NoError(t, err)

	queryRaw, err := testlib.ExecQuery(t, ctx, xion.GetNode(), "wasm", "contract-state", "smart", contractAddr, string(queryMsgStr))
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
	require.Equal(t, "https://example.com/redirect", queriedParams.RedirectURL)
	require.Equal(t, "https://example.com/icon.png", queriedParams.IconURL)
}
