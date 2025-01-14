package integration_tests

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path"
	"strings"
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/types"
	ibctest "github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
)

var configFileUrl string // Global variable to hold the file location argument
func init() {
	// Define command-line flags
	flag.StringVar(&configFileUrl, "configUrl", "", "URL to the configuration file")
}
func TestUpdateTreasuryConfigsWithLocalAndURL(t *testing.T) {
	flag.Parse()
	require.NotNil(t, configFileUrl, "No config file is provided via the configUrl flag")
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

	// Local File Test
	t.Log("Testing with local file")
	configData, err := os.ReadFile(path.Join(fp, "integration_tests", "testdata", "unsigned_msgs", "plain_config.json"))
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
	validateGrantConfigs(t, ctx, xion, treasuryAddr)

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
	validateGrantConfigs(t, ctx, xion, treasuryAddr)

	// Query and validate Fee Config
	validateFeeConfig(t, ctx, xion, treasuryAddr)
}

func validateGrantConfigs(t *testing.T, ctx context.Context, xion *cosmos.CosmosChain, treasuryAddr string) {
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
	require.Equal(t, "/cosmos.staking.v1.MsgDelegate", queriedGrantConfigUrls[1])
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
	require.Equal(t, "/cosmos.feegrant.v1.BasicAllowance", queriedFeeConfig.Allowance.TypeURL)
}

func TestUpdateTreasuryConfigsWithAALocalAndURL(t *testing.T) {
	require.NotNil(t, configFileUrl, "Skipping test as no config file is provided via the --config flag")
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

	_, err = ExecBin(t, ctx, xion.GetNode(),
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

	codeIDStr, err := xion.StoreContract(ctx, xionUser.FormattedAddress(),
		path.Join(fp, "integration_tests", "testdata", "contracts", "treasury-aarch64.wasm"))
	require.NoError(t, err)

	t.Log("Instantiating Treasury contract")
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
	t.Logf("Deployed and instantiated Treasury contract at address: %s", treasuryAddr)

	// Test with local config file
	t.Log("Testing with local config file")
	configData, err := os.ReadFile(path.Join(fp, "integration_tests", "testdata", "unsigned_msgs", "plain_config.json"))
	require.NoError(t, err)

	file, err := os.CreateTemp("", "*-config.json")
	require.NoError(t, err)
	_, err = file.Write(configData)
	require.NoError(t, err)
	err = UploadFileToContainer(t, ctx, xion.GetNode(), file)
	require.NoError(t, err)

	configFilePath := strings.Split(file.Name(), "/")
	cmd := []string{
		"tx", "xion", "update-configs", treasuryAddr, path.Join(xion.GetNode().HomeDir(), configFilePath[len(configFilePath)-1]),
		"--chain-id", xion.Config().ChainID,
		"--from", aaContractAddr,
		"--gas-prices", "1uxion", "--gas-adjustment", "2",
		"--gas", "400000",
		"--local",
		"--generate-only",
	}
	unsignedTx, err := ExecBin(t, ctx, xion.GetNode(), cmd...)
	require.NoError(t, err)

	t.Log("Signing transaction for local config")
	unsignedTxFile := WriteUnsignedTxToFile(t, unsignedTx)
	defer os.Remove(unsignedTxFile.Name())

	err = UploadFileToContainer(t, ctx, xion.GetNode(), unsignedTxFile)
	require.NoError(t, err)

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

	// Test with URL config file
	t.Log("Testing with URL config file")
	cmd = []string{
		"tx", "xion", "update-configs", treasuryAddr, configFileUrl,
		"--chain-id", xion.Config().ChainID,
		"--from", aaContractAddr,
		"--gas-prices", "1uxion", "--gas-adjustment", "2",
		"--gas", "400000",
		"--generate-only",
	}
	unsignedTx, err = ExecBin(t, ctx, xion.GetNode(), cmd...)
	require.NoError(t, err)

	t.Log("Signing transaction for URL config")
	unsignedTxFile = WriteUnsignedTxToFile(t, unsignedTx)
	defer os.Remove(unsignedTxFile.Name())

	err = UploadFileToContainer(t, ctx, xion.GetNode(), unsignedTxFile)
	require.NoError(t, err)

	unsignedTxFilePath = strings.Split(unsignedTxFile.Name(), "/")

	_, err = ExecTx(t, ctx, xion.GetNode(),
		xionUser.KeyName(),
		"xion", "sign",
		xionUser.KeyName(),
		aaContractAddr,
		path.Join(xion.GetNode().HomeDir(), unsignedTxFilePath[len(unsignedTxFilePath)-1]),
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err)

	// Validate Grant Configs
	validateGrantConfigs(t, ctx, xion, treasuryAddr)

	// Validate Fee Config
	validateFeeConfig(t, ctx, xion, treasuryAddr)
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
