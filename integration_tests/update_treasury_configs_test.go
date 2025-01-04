package integration_tests

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
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

func TestUpdateTreasuryConfigsWithAA(t *testing.T) {
	// Setup Xion chain
	td := BuildXionChain(t, "0.0uxion", nil)
	xion, ctx := td.xionChain, td.ctx

	config := types.GetConfig()
	config.SetBech32PrefixForAccount("xion", "xionpub")

	fp, err := os.Getwd()
	require.NoError(t, err)

	// Create and fund test user
	t.Log("Creating and funding user accounts")
	fundAmount := math.NewInt(10_000_000)
	users := ibctest.GetAndFundTestUsers(t, ctx, "default", fundAmount, xion)
	xionUser := users[0]

	err = testutil.WaitForBlocks(ctx, 2, xion)
	require.NoError(t, err)

	// Create a Secondary Key For Rotation
	recipientKeyName := "recipient-key"
	err = xion.CreateKey(ctx, recipientKeyName)
	require.NoError(t, err)
	receipientKeyAddressBytes, err := xion.GetAddress(ctx, recipientKeyName)
	require.NoError(t, err)
	_, err = types.Bech32ifyAddressBytes(xion.Config().Bech32Prefix, receipientKeyAddressBytes)
	require.NoError(t, err)

	account, err := ExecBin(t, ctx, xion.GetNode(),
		"keys", "show",
		xionUser.KeyName(),
		"--keyring-backend", keyring.BackendTest,
		"-p",
	)
	require.NoError(t, err)

	// Store AA Wasm Contract
	codeID, err := xion.StoreContract(ctx, xionUser.FormattedAddress(), path.Join(fp,
		"integration_tests", "testdata", "contracts", "account_updatable-aarch64.wasm"))
	require.NoError(t, err)

	depositedFunds := fmt.Sprintf("%d%s", 1000000, xion.Config().Denom)

	registeredTxHash, err := ExecTx(t, ctx, xion.GetNode(),
		xionUser.KeyName(),
		"xion", "register",
		codeID,
		xionUser.KeyName(),
		"--funds", depositedFunds,
		"--salt", "0",
		"--authenticator", "Secp256K1",
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err)
	t.Logf("tx hash: %s", registeredTxHash)

	txDetails, err := ExecQuery(t, ctx, xion.GetNode(), "tx", registeredTxHash)
	require.NoError(t, err)
	aaContractAddr := GetAAContractAddress(t, txDetails)

	contractBalance, err := xion.GetBalance(ctx, aaContractAddr, xion.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, math.NewInt(1000000), contractBalance)

	contractState, err := ExecQuery(t, ctx, xion.GetNode(), "wasm", "contract-state", "smart", aaContractAddr, `{"authenticator_by_i_d":{ "id": 0 }}`)
	require.NoError(t, err)

	pubkey64, ok := contractState["data"].(string)
	require.True(t, ok)
	pubkeyRawJSON, err := base64.StdEncoding.DecodeString(pubkey64)
	require.NoError(t, err)
	var pubKeyMap jsonAuthenticator
	json.Unmarshal(pubkeyRawJSON, &pubKeyMap)
	require.Equal(t, account["key"], pubKeyMap["Secp256K1"]["pubkey"])

	// Deploy contract
	t.Log("Deploying contract")
	codeIDStr, err := xion.StoreContract(ctx, xionUser.FormattedAddress(),
		path.Join(fp, "integration_tests", "testdata", "contracts", "treasury-aarch64.wasm"))
	require.NoError(t, err)

	// Instantiate contract
	t.Log("Instantiating contract")
	accAddr, err := types.AccAddressFromBech32(aaContractAddr)
	require.NoError(t, err)
	instantiateMsg := TreasuryInstantiateMsg{
		Admin:        accAddr,
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
	unsignedTxFile, err := os.CreateTemp("", "*-msg-update-config.json")
	require.NoError(t, err)
	cmd := []string{
		"tx", "xion", "update-configs", treasuryAddr, path.Join(xion.GetNode().HomeDir(), grantsFilePath[len(grantsFilePath)-1]), path.Join(xion.GetNode().HomeDir(), feeConfigFilePath[len(grantsFilePath)-1]),
		"--chain-id", xion.Config().ChainID,
		"--from", aaContractAddr,
		"--gas-prices", "1uxion", "--gas-adjustment", "2",
		"--gas", "400000", // set gas limit high because auto broadcasts the transaction
		"--generate-only",
	}

	unsignedTx, err := ExecBin(t, ctx, xion.GetNode(), cmd...)
	require.NoError(t, err)
	t.Log("Unsigned Tx: ", unsignedTx)
	unsignedTxBz, err := json.Marshal(unsignedTx)
	require.NoError(t, err)
	// Write the unsigned transaction to a file
	_, err = unsignedTxFile.Write(unsignedTxBz)
	require.NoError(t, err)
	err = UploadFileToContainer(t, ctx, xion.GetNode(), unsignedTxFile)
	require.NoError(t, err)

	// TODO: sign the unsigned transaction
	unsignedTxFilePath := strings.Split(unsignedTxFile.Name(), "/")
	_, err = ExecTx(t, ctx, xion.GetNode(),
		xionUser.KeyName(),
		"xion", "sign",
		xionUser.KeyName(),
		aaContractAddr,
		path.Join(xion.GetNode().HomeDir(), unsignedTxFilePath[len(unsignedTxFilePath)-1]),
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err)

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
