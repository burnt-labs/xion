package integration_tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"
	"testing"

	xiontypes "github.com/burnt-labs/xion/x/xion/types"
	"github.com/cosmos/cosmos-sdk/types"
	ibctest "github.com/strangelove-ventures/interchaintest/v7"
	"github.com/strangelove-ventures/interchaintest/v7/testutil"
	"github.com/stretchr/testify/require"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	aatypes "github.com/larry0x/abstract-account/x/abstractaccount/types"
)

func TestXionAbstractAccount(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()
	td := BuildXionChain(t, "0.0uxion", ModifyInterChainGenesis(ModifyInterChainGenesisFn{ModifyGenesisShortProposals}, [][]string{{votingPeriod, maxDepositPeriod}}))
	xion, ctx := td.xionChain, td.ctx

	// Register All messages we are interacting.
	xion.Config().EncodingConfig.InterfaceRegistry.RegisterImplementations(
		(*types.Msg)(nil),
		&xiontypes.MsgSetPlatformPercentage{},
		&xiontypes.MsgSend{},
		&wasmtypes.MsgInstantiateContract{},
		&wasmtypes.MsgExecuteContract{},
		&wasmtypes.MsgStoreCode{},
		&aatypes.MsgUpdateParams{},
		&aatypes.MsgRegisterAccount{},
	)

	xion.Config().EncodingConfig.InterfaceRegistry.RegisterImplementations((*authtypes.AccountI)(nil), &aatypes.AbstractAccount{})
	xion.Config().EncodingConfig.InterfaceRegistry.RegisterImplementations((*cryptotypes.PubKey)(nil), &aatypes.NilPubKey{})

	// Create and Fund User Wallets
	t.Log("creating and funding user accounts")
	fundAmount := int64(10_000_000)
	users := ibctest.GetAndFundTestUsers(t, ctx, "default", fundAmount, xion)
	xionUser := users[0]
	currentHeight, _ := xion.Height(ctx)
	testutil.WaitForBlocks(ctx, int(currentHeight)+8, xion)
	t.Logf("created xion user %s", xionUser.FormattedAddress())

	xionUserBalInitial, err := xion.GetBalance(ctx, xionUser.FormattedAddress(), xion.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, fundAmount, xionUserBalInitial)

	// Create a Secondary Key For Rotation
	recipientKeyName := "recipient-key"
	err = xion.CreateKey(ctx, recipientKeyName)
	require.NoError(t, err)
	receipientKeyAddressBytes, err := xion.GetAddress(ctx, recipientKeyName)
	require.NoError(t, err)
	recipientKeyAddress, err := types.Bech32ifyAddressBytes(xion.Config().Bech32Prefix, receipientKeyAddressBytes)
	require.NoError(t, err)

	// Get Public Key For Funded Account
	account, err := ExecBin(t, ctx, xion.FullNodes[0],
		xionUser.KeyName(),
		"keys", "show",
		xionUser.KeyName(),
		"--keyring-backend", keyring.BackendTest,
		"-p",
	)
	require.NoError(t, err)

	// Store Wasm Contract
	codeID, err := xion.StoreContract(ctx, xionUser.FormattedAddress(), "./testdata/contracts/account_updatable-aarch64.wasm")
	require.NoError(t, err)

	depositedFunds := fmt.Sprintf("%d%s", 100000, xion.Config().Denom)

	// Register Abstract Account Using Public Key
	registeredTxHash, err := ExecTx(t, ctx, xion.FullNodes[0],
		xionUser.KeyName(),
		"abstract-account", "register",
		codeID,
		fmt.Sprintf(`{"pubkey": "%s"}`, account["key"]),
		"--funds", depositedFunds,
		"--salt", "foo",
		"--chain-id", xion.Config().ChainID,
	)

	txDetails, err := ExecQuery(t, ctx, xion.FullNodes[0], "tx", registeredTxHash)
	require.NoError(t, err)
	aaContractAddr := GetAAContractAddress(t, txDetails)

	contractBalance, err := xion.GetBalance(ctx, aaContractAddr, xion.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, uint64(100000), uint64(contractBalance))

	/*
			NOTE: Ideally we would use this metod, however the QueryContract formats the string making it harder to predict.
		var ContractResponse interface{}
			require.NoError(t, xion.QueryContract(ctx, aaContractAddr, fmt.Sprintf(`{"pubkey":{}}`), ContractResponse))
	*/

	contractState, err := ExecQuery(t, ctx, xion.FullNodes[0], "wasm", "contract-state", "smart", aaContractAddr, fmt.Sprintf(`{"pubkey":{}}`))
	require.NoError(t, err)

	pubkey, ok := contractState["data"].(string)
	require.True(t, ok)
	require.Equal(t, account["key"], pubkey)

	// Generate Msg Send without signatures
	jsonMsg := RawJSONMsgSend(t, aaContractAddr, recipientKeyAddress, xion.Config().Denom)
	require.NoError(t, err)
	require.True(t, json.Valid(jsonMsg))

	sendFile, err := os.CreateTemp("", "*-msg-bank-send.json")
	require.NoError(t, err)
	defer os.Remove(sendFile.Name())

	_, err = sendFile.Write(jsonMsg)
	require.NoError(t, err)

	err = UploadFileToContainer(t, ctx, xion.FullNodes[0], sendFile)
	require.NoError(t, err)

	// Sign and broadcast a transaction
	sendFilePath := strings.Split(sendFile.Name(), "/")
	_, err = ExecTx(t, ctx, xion.FullNodes[0],
		xionUser.KeyName(),
		"xion", "sign",
		xionUser.KeyName(),
		path.Join(xion.FullNodes[0].HomeDir(), sendFilePath[len(sendFilePath)-1]),
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err)

	// Confirm the updated balance
	balance, err := xion.GetBalance(ctx, recipientKeyAddress, xion.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, uint64(100000), uint64(balance))

	// Generate Key Rotation Msg
	account, err = ExecBin(t, ctx, xion.FullNodes[0],
		xionUser.KeyName(),
		"keys", "show",
		xionUser.KeyName(),
		"--keyring-backend", keyring.BackendTest,
		"-p",
	)

	jsonExecMsg := RawJSONMsgExecContractNewPubKey(t, aaContractAddr, aaContractAddr, fmt.Sprintf("%s", account["key"]))
	require.NoError(t, err)
	require.True(t, json.Valid(jsonExecMsg))

	rotateFile, err := os.CreateTemp("", "*-msg-exec-rotate-key.json")
	require.NoError(t, err)
	defer os.Remove(rotateFile.Name())

	_, err = rotateFile.Write(jsonExecMsg)
	require.NoError(t, err)

	err = UploadFileToContainer(t, ctx, xion.FullNodes[0], rotateFile)
	require.NoError(t, err)

	rotateFilePath := strings.Split(rotateFile.Name(), "/")
	_, err = ExecTx(t, ctx, xion.FullNodes[0],
		xionUser.KeyName(),
		"xion", "sign",
		xionUser.KeyName(),
		path.Join(xion.FullNodes[0].HomeDir(), rotateFilePath[len(rotateFilePath)-1]),
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err)

	updatedContractstate, err := ExecQuery(t, ctx, xion.FullNodes[0], "wasm", "contract-state", "smart", aaContractAddr, fmt.Sprintf(`{"pubkey":{}}`))
	require.NoError(t, err)

	updatedPubKey, ok := updatedContractstate["data"].(string)
	require.True(t, ok)
	require.Equal(t, account["key"], updatedPubKey)

}

func GetAAContractAddress(t *testing.T, txDetails map[string]interface{}) string {
	logs, ok := txDetails["logs"].([]interface{})
	require.True(t, ok)

	log, ok := logs[0].(map[string]interface{})
	require.True(t, ok)

	events, ok := log["events"].([]interface{})
	require.True(t, ok)

	event, ok := events[4].(map[string]interface{})
	require.True(t, ok)

	attributes, ok := event["attributes"].([]interface{})
	require.True(t, ok)

	attribute, ok := attributes[0].(map[string]interface{})
	require.True(t, ok)

	addr, ok := attribute["value"].(string)
	require.True(t, ok)

	return addr
}
