package integration_tests

import (
	"encoding/base64"
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

type jsonauthenticator map[string]map[string]string

func TestXionAbstractAccount(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()
	td := BuildXionChain(t, "0.0uxion", ModifyInterChainGenesis(ModifyInterChainGenesisFn{ModifyGenesisShortProposals}, [][]string{{votingPeriod, maxDepositPeriod}}))
	xion, ctx := td.xionChain, td.ctx

	config := types.GetConfig()
	config.SetBech32PrefixForAccount("xion", "xionpub")

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
	t.Log("Funded Account:")
	for k, v := range account {
		t.Logf("[%s]: %v", k, v)
	}

	fp, err := os.Getwd()
	require.NoError(t, err)

	// Store Wasm Contract
	codeID, err := xion.StoreContract(ctx, xionUser.FormattedAddress(), path.Join(fp, "testdata", "contracts", "account_updatable-aarch64.wasm"))
	require.NoError(t, err)

	// retrieve the hash
	codeResp, err := ExecQuery(t, ctx, xion.FullNodes[0],
		"wasm", "code-info", codeID)
	require.NoError(t, err)
	t.Logf("code response: %s", codeResp)

	depositedFunds := fmt.Sprintf("%d%s", 100000, xion.Config().Denom)

	// NOTE: 1HERE
	/*
		// predict the contract address so it can be verified
		salt := "0"
		creatorAddr := types.AccAddress(xionUser.Address())
		codeHash, err := hex.DecodeString(codeResp["data_hash"].(string))
		require.NoError(t, err)
		predictedAddr := wasmkeeper.BuildContractAddressPredictable(codeHash, creatorAddr, []byte(salt), []byte{})
		t.Logf("predicted address: %s", predictedAddr.String())

		// Testdata create private key
		// CREATE PRIVATE KEY
		// USE PRIVATE KEY TO SIGN PRECOMPUTE ADDRESS
		// BUILD MESSAGE WITH NEW SIGNATURE
		privateKey := secp256k1.GenPrivKey()
		publicKey := privateKey.PubKey()
		publicKeyJSON, err := json.Marshal(publicKey)
		require.NoError(t, err)
		t.Logf("private key: %s", privateKey)
		t.Logf("public key: %s", publicKeyJSON)

		// sha256 the contract addr, as it expects
		signature, err := clientCtx.Keyring.Sign(clientCtx.GetFromName(), []byte(predictedAddr.String()))
		require.NoError(t, err)

		// Check if it's verifiable
		require.True(t, publicKey.VerifySignature([]byte(predictedAddr.String()), signature[:]))

		authenticatorDetails := map[string]interface{}{}
		authenticatorDetails["pubkey"] = publicKey.Bytes()

		authenticator := map[string]interface{}{}
		authenticator["Secp256K1"] = authenticatorDetails
		instantiateMsg := map[string]interface{}{}
		instantiateMsg["id"] = 0
		instantiateMsg["authenticator"] = authenticator

		instantiateMsg["signature"] = signature
		instantiateMsgStr, err := json.Marshal(instantiateMsg)
		require.NoError(t, err)
		t.Logf("inst msg: %s", string(instantiateMsgStr))

		// Register Abstract Account Using Public Key
		registeredTxHash, err := ExecTx(t, ctx, xion.FullNodes[0],
			xionUser.KeyName(),
			"abstract-account", "register",
			codeID,
			string(instantiateMsgStr),
			"--funds", depositedFunds,
			"--salt", "0",
			"--chain-id", xion.Config().ChainID,
		)
		require.NoError(t, err)
	*/
	// NOTE: TO HERE

	registeredTxHash, err := ExecTx(t, ctx, xion.FullNodes[0],
		xionUser.KeyName(),
		"xion", "register",
		codeID,
		xionUser.KeyName(),
		"--funds", depositedFunds,
		"--authenticator", "Secp256K1",
		"--salt", "0",
		"--authenticator-id", "0",
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err)

	txDetails, err := ExecQuery(t, ctx, xion.FullNodes[0], "tx", registeredTxHash)
	require.NoError(t, err)
	t.Logf("TxDetails: %s", txDetails)
	aaContractAddr := GetAAContractAddress(t, txDetails)

	contractBalance, err := xion.GetBalance(ctx, aaContractAddr, xion.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, uint64(100000), uint64(contractBalance))

	contractState, err := ExecQuery(t, ctx, xion.FullNodes[0], "wasm", "contract-state", "smart", aaContractAddr, fmt.Sprintf(`{"authenticator_by_i_d":{ "id": 0 }}`))
	require.NoError(t, err)

	pubkey64, ok := contractState["data"].(string)
	require.True(t, ok)
	pubkeyRawJSON, err := base64.StdEncoding.DecodeString(pubkey64)
	require.NoError(t, err)
	var pubKeyMap jsonauthenticator
	json.Unmarshal(pubkeyRawJSON, &pubKeyMap)
	require.Equal(t, account["key"], pubKeyMap["Secp256K1"]["pubkey"])

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

	updatedContractstate, err := ExecQuery(t, ctx, xion.FullNodes[0], "wasm", "contract-state", "smart", aaContractAddr, fmt.Sprintf(`{"authenticator_by_i_d":{ "id": 0 }}`))
	require.NoError(t, err)

	updatedPubKey, ok := updatedContractstate["data"].(string)
	require.True(t, ok)

	updatedPubKeyRawJSON, err := base64.StdEncoding.DecodeString(updatedPubKey)
	require.NoError(t, err)
	var updatedPubKeyMap jsonauthenticator

	json.Unmarshal(updatedPubKeyRawJSON, &updatedPubKeyMap)
	require.Equal(t, account["key"], updatedPubKeyMap["Secp256K1"]["pubkey"])

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
