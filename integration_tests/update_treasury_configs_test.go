package integration_tests

import (
	"encoding/json"
	"os"
	"path"
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/types"
	ibctest "github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
)

func TestUpdateTreasuryConfigs(t *testing.T) {
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
	instantiateMsg := TreasuryInstantiateMsg{
		TypeUrls:     []string{},
		GrantConfigs: []GrantConfig{},
		FeeConfig: &FeeConfig{
			Description: "test fee grant",
		},
	}
	instantiateMsgStr, err := json.Marshal(instantiateMsg)
	require.NoError(t, err)

	treasuryAddr, err := xion.InstantiateContract(ctx, xionUser.KeyName(), codeIDStr, string(instantiateMsgStr), true)
	require.NoError(t, err)

	// Read JSON files for grants and fee configs
	t.Log("Reading configuration JSON files")
	grantsFile, err := os.ReadFile(path.Join(fp, "integration_tests", "testdata", "unsigned_msgs", "grants.json"))
	require.NoError(t, err)

	feeConfigsFile, err := os.ReadFile(path.Join(fp, "integration_tests", "testdata", "unsigned_msgs", "fee_configs.json"))
	require.NoError(t, err)

	var grants []map[string]interface{}
	var feeConfigs []FeeConfig
	err = json.Unmarshal(grantsFile, &grants)
	require.NoError(t, err)

	err = json.Unmarshal(feeConfigsFile, &feeConfigs)
	require.NoError(t, err)

	// Construct update_configs execute message
	t.Log("Constructing update_configs message")
	executeMsg := map[string]interface{}{
		"update_configs": map[string]interface{}{
			"grants":      grants,
			"fee_configs": feeConfigs,
		},
	}
	executeMsgStr, err := json.Marshal(executeMsg)
	require.NoError(t, err)

	executeCmd := []string{
		"wasm", "execute", treasuryAddr, string(executeMsgStr), "--chain-id", xion.Config().ChainID,
	}

	_, err = ExecTx(t, ctx, xion.GetNode(), xionUser.KeyName(), executeCmd...)
	require.NoError(t, err)

	// Query and validate Grant Config URLs
	t.Log("Querying grant config type URLs")
	grantQueryMsg := map[string]interface{}{
		"grant_config_type_urls": struct{}{},
	}
	grantQueryMsgStr, err := json.Marshal(grantQueryMsg)
	require.NoError(t, err)

	grantQueryRaw, err := ExecQuery(t, ctx, xion.GetNode(), "wasm", "contract-state", "smart", treasuryAddr, string(grantQueryMsgStr))
	require.NoError(t, err)

	grantQueryBytes, err := json.Marshal(grantQueryRaw["data"])
	require.NoError(t, err)

	var queriedGrantConfigUrls []string
	err = json.Unmarshal(grantQueryBytes, &queriedGrantConfigUrls)
	require.NoError(t, err)

	// Validate that all grants are in the contract state
	require.Equal(t, len(grants), len(queriedGrantConfigUrls))
	for _, grant := range grants {
		msgTypeURL := grant["msg_type_url"].(string)
		require.Contains(t, queriedGrantConfigUrls, msgTypeURL)
	}

	// Query and validate Fee Config
	t.Log("Querying fee config")
	feeQueryMsg := map[string]interface{}{
		"fee_config": struct{}{},
	}
	feeQueryMsgStr, err := json.Marshal(feeQueryMsg)
	require.NoError(t, err)

	feeQueryRaw, err := ExecQuery(t, ctx, xion.GetNode(), "wasm", "contract-state", "smart", treasuryAddr, string(feeQueryMsgStr))
	require.NoError(t, err)

	feeQueryBytes, err := json.Marshal(feeQueryRaw["data"])
	require.NoError(t, err)

	var queriedFeeConfig FeeConfig
	err = json.Unmarshal(feeQueryBytes, &queriedFeeConfig)
	require.NoError(t, err)

	// Validate Fee Config
	require.Equal(t, len(feeConfigs), 1)
	require.Equal(t, feeConfigs[0].Description, queriedFeeConfig.Description)
	require.Equal(t, feeConfigs[0].Allowance.TypeURL, queriedFeeConfig.Allowance.TypeURL)
}
