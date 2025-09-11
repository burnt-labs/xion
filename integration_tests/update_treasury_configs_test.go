package integration_tests

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/types"
	ibctest "github.com/strangelove-ventures/interchaintest/v10"
	"github.com/strangelove-ventures/interchaintest/v10/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v10/testutil"
	"github.com/stretchr/testify/require"
)

// Global variable to hold the file location argument
var configFileUrl = "https://raw.githubusercontent.com/burnt-labs/xion/refs/heads/main/integration_tests/testdata/unsigned_msgs/plain_config.json"

func TestUpdateTreasuryConfigsWithLocalAndURL(t *testing.T) {
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

	codeIDStr, err := xion.StoreContract(ctx, xionUser.FormattedAddress(),
		IntegrationTestPath("testdata", "contracts", "treasury-aarch64.wasm"))
	require.NoError(t, err)

	userAddr := xionUser.FormattedAddress()
	// Instantiate contract
	t.Log("Instantiating contract")
	instantiateMsg := TreasuryInstantiateMsg{
		Admin:        &userAddr,
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

	treasuryAddr, err := xion.InstantiateContract(ctx, xionUser.KeyName(), codeIDStr, string(instantiateMsgStr), true)
	require.NoError(t, err)
	t.Logf("Deployed and instantiated contract at address: %s", treasuryAddr)

	// Local File Test
	t.Log("Testing with local file")
	configData, err := os.ReadFile(IntegrationTestPath("testdata", "unsigned_msgs", "plain_config.json"))
	require.NoError(t, err)

	file, err := os.CreateTemp("", "*-config.json")
	require.NoError(t, err)
	_, err = file.Write(configData)
	require.NoError(t, err)
	err = UploadFileToContainer(t, ctx, xion.GetNode(), file)
	require.NoError(t, err)

	configFilePath := strings.Split(file.Name(), "/")
	cmd := []string{
		"xion", "update-configs", treasuryAddr, path.Join(xion.GetNode().HomeDir(), configFilePath[len(configFilePath)-1]),
		"--chain-id", xion.Config().ChainID,
		"--from", xionUser.KeyName(),
		"--gas-prices", "1uxion", "--gas-adjustment", "1.4",
		"--gas", "400000",
		"--local",
		"-y",
	}
	_, err = ExecTx(t, ctx, xion.GetNode(), xionUser.KeyName(), cmd...)
	require.NoError(t, err)

	// Wait for the transaction to be included in a block
	err = testutil.WaitForBlocks(ctx, 2, xion)
	require.NoError(t, err)

	// Query and validate Grant Config URLs
	validateDefaultGrantConfigs(t, ctx, xion, treasuryAddr)

	// Query and validate Fee Config
	validateFeeConfig(t, ctx, xion, treasuryAddr)

	// URL Test
	t.Log("Testing with URL")
	cmd = []string{
		"xion", "update-configs", treasuryAddr, configFileUrl,
		"--chain-id", xion.Config().ChainID,
		"--from", xionUser.KeyName(),
		"--gas-prices", "1uxion", "--gas-adjustment", "1.4",
		"--gas", "400000",
		"-y",
	}
	_, err = ExecTx(t, ctx, xion.GetNode(), xionUser.KeyName(), cmd...)
	require.NoError(t, err)

	// Wait for the transaction to be included in a block
	err = testutil.WaitForBlocks(ctx, 2, xion)
	require.NoError(t, err)

	// Query and validate Grant Config URLs
	validateDefaultGrantConfigs(t, ctx, xion, treasuryAddr)

	// Query and validate Fee Config
	validateFeeConfig(t, ctx, xion, treasuryAddr)
}

func validateDefaultGrantConfigs(t *testing.T, ctx context.Context, xion *cosmos.CosmosChain, treasuryAddr string) {
	check := []string{"/cosmos.bank.v1beta1.MsgSend", "/cosmos.staking.v1beta1.MsgDelegate", "/cosmos.gov.v1beta1.MsgVote"}
	validateGrantConfigs(t, ctx, xion, treasuryAddr, 3, check...)
}

func validateGrantConfigs(t *testing.T, ctx context.Context, xion *cosmos.CosmosChain, treasuryAddr string, expected int, msgs ...string) {
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
	require.Equal(t, expected, len(queriedGrantConfigUrls), fmt.Sprintf("got: %d, expected: %d,\n these are the grants: %v", len(queriedGrantConfigUrls), 3, queriedGrantConfigUrls))
	// check := []string{"/cosmos.bank.v1beta1.MsgSend", "/cosmos.staking.v1beta1.MsgDelegate", "/cosmos.gov.v1beta1.MsgVote"}
	for _, msg := range msgs {
		require.Contains(t, queriedGrantConfigUrls, msg)
	}
	/*
		exists := make(map[string]bool)
		for _, str := range queriedGrantConfigUrls {
			exists[str] = true
		}
		for _, str := range check {
			require.True(t, exists[str], "Expected %s to be in the grant config type URLs", str)
		}
	*/
}

func validateFeeConfig(t *testing.T, ctx context.Context, xion *cosmos.CosmosChain, treasuryAddr string) {
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
	require.Equal(t, "/cosmos.feegrant.v1beta1.AllowedMsgAllowance", queriedFeeConfig.Allowance.TypeURL)
}

func TestUpdateTreasuryConfigsWithAALocalAndURL(t *testing.T) {
	ctx := t.Context()

	// Setup Xion chain
	xion := BuildXionChain(t)

	config := types.GetConfig()
	config.SetBech32PrefixForAccount("xion", "xionpub")

	// Create and fund test user
	t.Log("Creating and funding user accounts")
	fundAmount := math.NewInt(100_000_000)
	users := ibctest.GetAndFundTestUsers(t, ctx, "default", fundAmount, xion)
	xionUser := users[0]

	err := testutil.WaitForBlocks(ctx, 2, xion)
	require.NoError(t, err)

	// Create a Secondary Key For Rotation
	recipientKeyName := "recipient-key"
	err = xion.CreateKey(ctx, recipientKeyName)
	require.NoError(t, err)
	receipientKeyAddressBytes, err := xion.GetAddress(ctx, recipientKeyName)
	require.NoError(t, err)
	_, err = types.Bech32ifyAddressBytes(xion.Config().Bech32Prefix, receipientKeyAddressBytes)
	require.NoError(t, err)

	_, err = ExecBin(t, ctx, xion.GetNode(),
		"keys", "show",
		xionUser.KeyName(),
		"--keyring-backend", keyring.BackendTest,
		"-p",
	)
	require.NoError(t, err)

	// Store AA Wasm Contract
	codeID, err := xion.StoreContract(ctx, xionUser.FormattedAddress(),
		IntegrationTestPath("testdata", "contracts", "account_updatable-aarch64.wasm"))
	require.NoError(t, err)

	depositedFunds := fmt.Sprintf("%d%s", 10000000, xion.Config().Denom)

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
	t.Logf("AA Contract Address: %s", aaContractAddr)

	codeIDStr, err := xion.StoreContract(ctx, xionUser.FormattedAddress(),
		IntegrationTestPath("testdata", "contracts", "treasury-aarch64.wasm"))
	require.NoError(t, err)

	t.Log("Instantiating Treasury contract")
	accAddr, err := types.AccAddressFromBech32(aaContractAddr)
	require.NoError(t, err)

	accAddrStr := accAddr.String()
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

	treasuryAddr1, err := xion.InstantiateContract(ctx, xionUser.KeyName(), codeIDStr, string(instantiateMsgStr), true)
	require.NoError(t, err)
	treasuryAddr2, err := xion.InstantiateContract(ctx, xionUser.KeyName(), codeIDStr, string(instantiateMsgStr), true)
	require.NoError(t, err)
	t.Logf("Deployed and instantiated Treasury contract at address: %s %s", treasuryAddr1, treasuryAddr2)

	// Test with local config file
	t.Log("Testing with local config file")
	configData, err := os.ReadFile(IntegrationTestPath("testdata", "unsigned_msgs", "plain_config.json"))
	require.NoError(t, err)

	file, err := os.CreateTemp("", "*-config.json")
	require.NoError(t, err)
	_, err = file.Write(configData)
	require.NoError(t, err)
	err = UploadFileToContainer(t, ctx, xion.GetNode(), file)
	require.NoError(t, err)

	configFilePath := strings.Split(file.Name(), "/")
	cmd := []string{
		"tx", "xion", "update-configs", treasuryAddr1, path.Join(xion.GetNode().HomeDir(), configFilePath[len(configFilePath)-1]),
		"--chain-id", xion.Config().ChainID,
		"--from", aaContractAddr,
		"--gas-prices", "1uxion", "--gas-adjustment", "2",
		"--gas", "4000000",
		"--local",
		"--generate-only",
	}
	unsignedTx, err := ExecBin(t, ctx, xion.GetNode(), cmd...)
	require.NoError(t, err)

	// Marshal the unsignedTx to JSON for logging
	_, err = json.MarshalIndent(unsignedTx, "", "  ")
	require.NoError(t, err)

	unsignedTxFile := WriteUnsignedTxToFile(t, unsignedTx)

	defer os.Remove(unsignedTxFile.Name())

	err = UploadFileToContainer(t, ctx, xion.GetNode(), unsignedTxFile)
	require.NoError(t, err)

	unsignedTxFilePath := strings.Split(unsignedTxFile.Name(), "/")

	_, err = ExecBinRaw(t, ctx, xion.GetNode(),
		"tx", "xion", "sign", xionUser.KeyName(), aaContractAddr, path.Join(xion.GetNode().HomeDir(), unsignedTxFilePath[len(unsignedTxFilePath)-1]),
		"--from", xionUser.KeyName(),
		"--chain-id", xion.Config().ChainID,
		"--keyring-backend", keyring.BackendTest,
		"--output", "json",
		"-y",
		"--node", fmt.Sprintf("tcp://%s:26657", xion.GetNode().HostName()),
	)
	require.NoError(t, err)

	// Wait for the transaction to be included in a block
	err = testutil.WaitForBlocks(ctx, 2, xion)
	require.NoError(t, err)

	// Validate Grant Configs
	validateDefaultGrantConfigs(t, ctx, xion, treasuryAddr1)

	// Validate Fee Config
	validateFeeConfig(t, ctx, xion, treasuryAddr1)

	// Test with URL config file
	t.Log("Testing with URL config file")
	cmd = []string{
		"tx", "xion", "update-configs", treasuryAddr2, configFileUrl,
		"--chain-id", xion.Config().ChainID,
		"--from", aaContractAddr,
		"--gas-prices", "1uxion", "--gas-adjustment", "2",
		"--gas", "4000000",
		"--generate-only",
	}
	unsignedTx, err = ExecBin(t, ctx, xion.GetNode(), cmd...)
	require.NoError(t, err)

	t.Log("Signing transaction for URL config")
	unsignedTxFile2 := WriteUnsignedTxToFile(t, unsignedTx)
	defer os.Remove(unsignedTxFile2.Name())

	err = UploadFileToContainer(t, ctx, xion.GetNode(), unsignedTxFile2)
	require.NoError(t, err)

	unsignedTxFilePath2 := strings.Split(unsignedTxFile2.Name(), "/")

	_, err = ExecBinRaw(t, ctx, xion.GetNode(),
		"tx", "xion", "sign", xionUser.KeyName(), aaContractAddr, path.Join(xion.GetNode().HomeDir(), unsignedTxFilePath2[len(unsignedTxFilePath2)-1]),
		"--from", xionUser.KeyName(),
		"--chain-id", xion.Config().ChainID,
		"--keyring-backend", keyring.BackendTest,
		"--output", "json",
		"-y",
		"--node", fmt.Sprintf("tcp://%s:26657", xion.GetNode().HostName()),
	)
	require.NoError(t, err)

	// Wait for the transaction to be included in a block
	err = testutil.WaitForBlocks(ctx, 2, xion)
	require.NoError(t, err)

	// Validate Grant Configs
	validateDefaultGrantConfigs(t, ctx, xion, treasuryAddr2)

	// Validate Fee Config
	validateFeeConfig(t, ctx, xion, treasuryAddr2)
}

func WriteUnsignedTxToFile(t *testing.T, unsignedTx map[string]interface{}) *os.File {
	t.Helper()
	unsignedTxFile, err := os.CreateTemp("", "*-msg-update-config.json")
	require.NoError(t, err)

	unsignedTxBz, err := json.Marshal(unsignedTx)
	require.NoError(t, err)
	_, err = unsignedTxFile.Write(unsignedTxBz)
	require.NoError(t, err)

	return unsignedTxFile
}
