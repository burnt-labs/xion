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

	// Query and validate contract state
	t.Log("Validating updated contract state")
	contractState, err := ExecQuery(t, ctx, xion.GetNode(), "wasm", "contract-state", "all", treasuryAddr)
	require.NoError(t, err)
	t.Logf("Updated Contract State: %s", contractState)

	// Assert the state contains the updated grants and fee configs
	storedGrants := contractState["grant_configs"].([]map[string]GrantConfig)
	require.Equal(t, len(grants), len(storedGrants))
	for i, storedGrant := range storedGrants {
		require.Equal(t, grants[i]["msg_type_url"], storedGrant["msg_type_url"])
		require.Equal(t, grants[i]["grant_config"].(GrantConfig).Description, storedGrant["grant_config"].Description)
		require.Equal(t, grants[i]["grant_config"].(GrantConfig).Authorization.TypeURL, storedGrant["grant_config"].Authorization.TypeURL)
	}

	storedFeeConfigs := contractState["fee_configs"].([]interface{})
	require.Equal(t, len(feeConfigs), len(storedFeeConfigs))
	for i, storedFeeConfig := range storedFeeConfigs {
		stored := storedFeeConfig.(map[string]interface{})
		require.Equal(t, feeConfigs[i].Description, stored["description"])
		require.Equal(t, feeConfigs[i].Allowance.TypeURL, stored["allowance"].(map[string]interface{})["type_url"])
	}
}
