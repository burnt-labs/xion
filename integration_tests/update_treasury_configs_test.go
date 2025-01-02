package integration_tests

import (
	"encoding/json"
	"os"
	"path"
	"strings"
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/types"
	ibctest "github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
)

func TestUpdateTreasuryConfigsWithCLI(t *testing.T) {
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
	t.Logf("Deployed and instantiated contract at address: %s", treasuryAddr)

	// Prepare JSON files for grants and fee configs
	t.Log("Preparing JSON files for grants and fee configs")
	grantsFile, err := os.CreateTemp("", "*-grants.json")
	require.NoError(t, err)
	feeConfigsFile, err := os.CreateTemp("", "*-fee_configs.json")
	require.NoError(t, err)

	grantsData, err := os.ReadFile(path.Join(fp, "integration_tests", "testdata", "unsigned_msgs", "grants.json"))
	require.NoError(t, err)

	feeConfigsData, err := os.ReadFile(path.Join(fp, "integration_tests", "testdata", "unsigned_msgs", "fee_configs.json"))
	require.NoError(t, err)

	_, err = grantsFile.Write(grantsData)
	require.NoError(t, err)

	_, err = feeConfigsFile.Write(feeConfigsData)
	require.NoError(t, err)

	err = UploadFileToContainer(t, ctx, xion.GetNode(), grantsFile)
	require.NoError(t, err)
	err = UploadFileToContainer(t, ctx, xion.GetNode(), feeConfigsFile)
	require.NoError(t, err)
	// Execute CLI command to submit transaction
	grantsFilePath := strings.Split(grantsFile.Name(), "/")
	feeConfigFilePath := strings.Split(feeConfigsFile.Name(), "/")
	t.Log("Executing CLI command to update configs")
	cmd := []string{
		"xion", "update-configs", treasuryAddr, path.Join(xion.GetNode().HomeDir(), grantsFilePath[len(grantsFilePath)-1]), path.Join(xion.GetNode().HomeDir(), feeConfigFilePath[len(grantsFilePath)-1]),
		"--from", xionUser.KeyName(),
		"--chain-id", xion.Config().ChainID,
		"--gas", "auto",
		"--fees", "1000uxion",
		"-y",
	}
	_, err = ExecTx(t, ctx, xion.GetNode(), xionUser.KeyName(), cmd...)
	require.NoError(t, err)

	// Wait for the transaction to be included in a block
	err = testutil.WaitForBlocks(ctx, 2, xion)
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
	require.Equal(t, 2, len(queriedGrantConfigUrls))
	require.Equal(t, "/cosmos.bank.v1.MsgSend", queriedGrantConfigUrls[0])

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
	require.Equal(t, "Fee allowance for user1", queriedFeeConfig.Description)
	require.Equal(t, "/cosmos.feegrant.v1.BasicAllowance", queriedFeeConfig.Allowance.TypeURL)
}
