package integration_tests

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"reflect"
	"strings"
	"testing"
	"time"

	signingv1beta1 "cosmossdk.io/api/cosmos/tx/signing/v1beta1"
	"cosmossdk.io/math"
	txsigning "cosmossdk.io/x/tx/signing"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/golang-jwt/jwt/v4"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/lestrrat-go/jwx/jwk"
	ibctest "github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	aatypes "github.com/larry0x/abstract-account/x/abstractaccount/types"
)

type jsonAuthenticator map[string]map[string]string

func TestXionAbstractAccountJWTCLI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	t.Parallel()

	td := BuildXionChain(t, "0.0uxion", ModifyInterChainGenesis(ModifyInterChainGenesisFn{ModifyGenesisShortProposals}, [][]string{{votingPeriod, maxDepositPeriod}}))
	xion, ctx := td.xionChain, td.ctx

	config := types.GetConfig()
	config.SetBech32PrefixForAccount("xion", "xionpub")

	// Create and Fund User Wallets
	t.Log("creating and funding user accounts")
	fundAmount := math.NewInt(10_000_000)
	users := ibctest.GetAndFundTestUsers(t, ctx, "default", fundAmount, xion)
	xionUser := users[0]
	err := testutil.WaitForBlocks(ctx, 8, xion)
	require.NoError(t, err)
	t.Logf("created xion user %s", xionUser.FormattedAddress())

	xionUserBalInitial, err := xion.GetBalance(ctx, xionUser.FormattedAddress(), xion.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, fundAmount, xionUserBalInitial)

	// load the test private key
	privateKeyBz, err := os.ReadFile("./integration_tests/testdata/keys/jwtRS256.key")
	require.NoError(t, err)
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyBz)
	require.NoError(t, err)
	t.Logf("private key: %v", privateKey)

	// log the test public key
	publicKey, err := jwk.New(privateKey)
	require.NoError(t, err)
	publicKey, err = publicKey.PublicKey()
	require.NoError(t, err)
	publicKeyJSON, err := json.Marshal(publicKey)
	require.NoError(t, err)
	t.Logf("public key: %s", publicKeyJSON)

	// build the jwk key
	testKey, err := jwk.ParseKey(privateKeyBz, jwk.WithPEM(true))
	require.NoError(t, err)
	err = testKey.Set("alg", "RS256")
	require.NoError(t, err)
	testKeyPublic, err := testKey.PublicKey()
	require.NoError(t, err)
	testPublicKeyJSON, err := json.Marshal(testKeyPublic)
	require.NoError(t, err)

	// deploy the key to the jwk module
	aud := "integration-test-project"

	createAudienceClaimHash, err := ExecTx(t, ctx, xion.GetNode(),
		xionUser.KeyName(),
		"jwk", "create-audience-claim",
		aud,
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err)
	t.Logf("create audience claim hash: %s", createAudienceClaimHash)

	txDetails, err := ExecQuery(t, ctx, xion.GetNode(), "tx", createAudienceClaimHash)
	require.NoError(t, err)
	t.Logf("TxDetails: %s", txDetails)

	createAudienceHash, err := ExecTx(t, ctx, xion.GetNode(),
		xionUser.KeyName(),
		"jwk", "create-audience",
		aud,
		string(testPublicKeyJSON),
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err)
	t.Logf("create audience hash: %s", createAudienceHash)

	// deploy the contract
	fp, err := os.Getwd()
	require.NoError(t, err)
	codeIDStr, err := xion.StoreContract(ctx, xionUser.FormattedAddress(),
		path.Join(fp, "integration_tests", "testdata", "contracts", "account_updatable-aarch64.wasm"))
	require.NoError(t, err)

	audienceQuery, err := ExecQuery(t, ctx, xion.GetNode(), "jwk", "list-audience")
	t.Logf("audiences: \n%s", audienceQuery)

	// retrieve the hash
	codeResp, err := ExecQuery(t, ctx, xion.GetNode(),
		"wasm", "code-info", codeIDStr)
	require.NoError(t, err)
	t.Logf("code response: %s", codeResp)

	sub := "integration-test-user"
	depositedFunds := fmt.Sprintf("%d%s", 10000, xion.Config().Denom)

	authenticatorDetails := map[string]interface{}{}
	authenticatorDetails["sub"] = sub
	authenticatorDetails["aud"] = aud
	authenticatorDetails["id"] = 0

	authenticator := map[string]interface{}{}
	authenticator["Jwt"] = authenticatorDetails

	instantiateMsg := map[string]interface{}{}
	instantiateMsg["authenticator"] = authenticator

	// predict the contract address so it can be verified
	salt := "0"
	creatorAddr := types.AccAddress(xionUser.Address())
	codeHash, err := hex.DecodeString(codeResp["data_hash"].(string))
	require.NoError(t, err)
	predictedAddr := wasmkeeper.BuildContractAddressPredictable(codeHash, creatorAddr, []byte(salt), []byte{})
	t.Logf("predicted address: %s", predictedAddr.String())

	// b64 the contract address to use as the transaction hash
	signature := base64.StdEncoding.EncodeToString([]byte(predictedAddr.String()))

	now := time.Now()
	fiveAgo := now.Add(-time.Second * 5)
	inFive := now.Add(time.Minute * 5)

	auds := jwt.ClaimStrings{aud}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss":              aud,
		"sub":              sub,
		"aud":              auds,
		"exp":              inFive.Unix(),
		"nbf":              fiveAgo.Unix(),
		"iat":              fiveAgo.Unix(),
		"transaction_hash": signature,
	})
	t.Logf("jwt claims: %v", token)

	// sign the JWT with the predefined key
	output, err := token.SignedString(privateKey)
	require.NoError(t, err)
	t.Logf("signed token: %s", output)

	authenticatorDetails["token"] = []byte(output)
	instantiateMsgStr, err := json.Marshal(instantiateMsg)
	require.NoError(t, err)
	t.Logf("inst msg: %s", string(instantiateMsgStr))

	// register the account

	registeredTxHash, err := ExecTx(t, ctx, xion.GetNode(),
		xionUser.KeyName(),
		"xion", "register",
		codeIDStr,
		xionUser.KeyName(),
		"--funds", depositedFunds,
		"--salt", "0",
		"--authenticator", "Jwt",
		"--token", output,
		"--sub", sub,
		"--aud", aud,
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err)
	t.Logf("tx hash: %s", registeredTxHash)

	contractsResponse, err := ExecQuery(t, ctx, xion.GetNode(), "wasm", "contracts", codeIDStr)
	require.NoError(t, err)

	contract := contractsResponse["contracts"].([]interface{})[0].(string)

	err = testutil.WaitForBlocks(ctx, 1, xion)
	require.NoError(t, err)
	newBalance, err := xion.GetBalance(ctx, contract, xion.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, int64(10_000), newBalance.Int64())

	// get the account from the chain. there might be a better way to do this
	accountResponse, err := ExecQuery(t, ctx, xion.GetNode(),
		"auth", "account", contract)
	require.NoError(t, err)
	t.Logf("account response: %s", accountResponse)

	ac, ok := accountResponse["account"]
	require.True(t, ok)

	ac2, ok := ac.(map[string]interface{})
	require.True(t, ok)

	acData, ok := ac2["value"]
	require.True(t, ok)

	accountJSON, err := json.Marshal(acData)
	require.NoError(t, err)

	var account aatypes.AbstractAccount
	err = xion.Config().EncodingConfig.Codec.UnmarshalJSON(accountJSON, &account)
	require.NoError(t, err)

	// create the raw tx
	sendMsg := fmt.Sprintf(`
		{
		 "body": {
		   "messages": [
		     {
		       "@type": "/cosmos.bank.v1beta1.MsgSend",
		       "from_address": "%s",
		       "to_address": "%s",
		       "amount": [
		         {
		           "denom": "%s",
		           "amount": "1337"
		         }
		       ]
		     }
		   ],
		   "memo": "",
		   "timeout_height": "0",
		   "extension_options": [],
		   "non_critical_extension_options": []
		 },
		 "auth_info": {
		   "signer_infos": [],
		   "fee": {
		     "amount": [],
		     "gas_limit": "200000",
		     "payer": "",
		     "granter": ""
		   },
		   "tip": null
		 },
		 "signatures": []
		}
			`, contract, xionUser.FormattedAddress(), "uxion")

	tx, err := xion.Config().EncodingConfig.TxConfig.TxJSONDecoder()([]byte(sendMsg))
	require.NoError(t, err)

	// create the sign bytes
	pubKey := account.GetPubKey()
	anyPk, err := codectypes.NewAnyWithValue(pubKey)
	require.NoError(t, err)
	signerData := txsigning.SignerData{
		Address:       account.GetAddress().String(),
		ChainID:       xion.Config().ChainID,
		AccountNumber: account.GetAccountNumber(),
		Sequence:      account.GetSequence(),
		PubKey: &anypb.Any{
			TypeUrl: anyPk.TypeUrl,
			Value:   anyPk.Value,
		}, // NOTE: NilPubKey
	}

	txBuilder, err := xion.Config().EncodingConfig.TxConfig.WrapTxBuilder(tx)
	require.NoError(t, err)

	sigData := signing.SingleSignatureData{
		SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
		Signature: nil,
	}

	sig := signing.SignatureV2{
		PubKey:   account.GetPubKey(),
		Data:     &sigData,
		Sequence: account.GetSequence(),
	}

	err = txBuilder.SetSignatures(sig)
	require.NoError(t, err)

	builtTx := txBuilder.GetTx()
	adaptableTx, ok := builtTx.(authsigning.V2AdaptableTx)
	if !ok {
		panic(fmt.Errorf("expected tx to implement V2AdaptableTx, got %T", builtTx))
	}
	txData := adaptableTx.GetSigningTxData()

	signBytes, err := xion.Config().EncodingConfig.TxConfig.SignModeHandler().GetSignBytes(
		ctx,
		signingv1beta1.SignMode(signing.SignMode_SIGN_MODE_DIRECT),
		signerData, txData)
	require.NoError(t, err)

	// our signature is the sha256 of the signbytes
	signatureBz := sha256.Sum256(signBytes)
	signature = base64.StdEncoding.EncodeToString(signatureBz[:])

	// we need to create a new valid token, making sure the time works
	now = time.Now()
	fiveAgo = now.Add(-time.Second * 5)
	inFive = now.Add(time.Minute * 5)

	token = jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss":              aud,
		"sub":              sub,
		"aud":              auds,
		"exp":              inFive.Unix(),
		"nbf":              fiveAgo.Unix(),
		"iat":              fiveAgo.Unix(),
		"transaction_hash": signature,
	})
	t.Logf("jwt claims: %v", token)

	// sign the JWT with the predefined key
	signedTokenStr, err := token.SignedString(privateKey)
	require.NoError(t, err)

	// add the auth index to the signature
	signedTokenBz := []byte(signedTokenStr)
	sigBytes := append([]byte{0}, signedTokenBz...)

	sigData = signing.SingleSignatureData{
		SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
		Signature: sigBytes,
	}

	sig = signing.SignatureV2{
		PubKey:   account.GetPubKey(),
		Data:     &sigData,
		Sequence: account.GetSequence(),
	}
	err = txBuilder.SetSignatures(sig)
	require.NoError(t, err)

	jsonTx, err := xion.Config().EncodingConfig.TxConfig.TxJSONEncoder()(txBuilder.GetTx())
	require.NoError(t, err)
	t.Logf("json tx: %s", jsonTx)

	output, err = ExecBroadcast(t, ctx, xion.GetNode(), jsonTx)
	require.NoError(t, err)
	t.Logf("output: %s", output)

	err = testutil.WaitForBlocks(ctx, 2, xion)
	require.NoError(t, err)
	newBalance, err = xion.GetBalance(ctx, contract, xion.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, int64(10_000-1337), newBalance.Int64())
}

func TestXionAbstractAccount(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()
	td := BuildXionChain(t, "0.0uxion", ModifyInterChainGenesis(ModifyInterChainGenesisFn{ModifyGenesisShortProposals}, [][]string{{votingPeriod, maxDepositPeriod}}))
	xion, ctx := td.xionChain, td.ctx

	config := types.GetConfig()
	config.SetBech32PrefixForAccount("xion", "xionpub")

	// Create and Fund User Wallets
	t.Log("creating and funding user accounts")
	fundAmount := math.NewInt(10_000_000)
	xionUser, err := ibctest.GetAndFundTestUserWithMnemonic(ctx, "default", deployerMnemonic, fundAmount, xion)
	require.NoError(t, err)
	currentHeight, _ := xion.Height(ctx)
	err = testutil.WaitForBlocks(ctx, int(currentHeight)+8, xion)
	require.NoError(t, err)
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
	account, err := ExecBin(t, ctx, xion.GetNode(),
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
	codeID, err := xion.StoreContract(ctx, xionUser.FormattedAddress(), path.Join(fp,
		"integration_tests", "testdata", "contracts", "account_updatable-aarch64.wasm"))
	require.NoError(t, err)

	// retrieve the hash
	codeResp, err := ExecQuery(t, ctx, xion.GetNode(),
		"wasm", "code-info", codeID)
	require.NoError(t, err)
	t.Logf("code response: %s", codeResp)

	depositedFunds := fmt.Sprintf("%d%s", 100000, xion.Config().Denom)

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
	t.Logf("TxDetails: %s", txDetails)
	aaContractAddr := GetAAContractAddress(t, txDetails)

	contractBalance, err := xion.GetBalance(ctx, aaContractAddr, xion.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, math.NewInt(100000), contractBalance)

	contractState, err := ExecQuery(t, ctx, xion.GetNode(), "wasm", "contract-state", "smart", aaContractAddr, `{"authenticator_by_i_d":{ "id": 0 }}`)
	require.NoError(t, err)

	pubkey64, ok := contractState["data"].(string)
	require.True(t, ok)
	pubkeyRawJSON, err := base64.StdEncoding.DecodeString(pubkey64)
	require.NoError(t, err)
	var pubKeyMap jsonAuthenticator
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

	err = UploadFileToContainer(t, ctx, xion.GetNode(), sendFile)
	require.NoError(t, err)

	// Sign and broadcast a transaction
	sendFilePath := strings.Split(sendFile.Name(), "/")
	_, err = ExecTx(t, ctx, xion.GetNode(),
		xionUser.KeyName(),
		"xion", "sign",
		xionUser.KeyName(),
		aaContractAddr,
		path.Join(xion.GetNode().HomeDir(), sendFilePath[len(sendFilePath)-1]),
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err)

	err = testutil.WaitForBlocks(ctx, 1, xion)
	require.NoError(t, err)

	// Confirm the updated balance
	balance, err := xion.GetBalance(ctx, recipientKeyAddress, xion.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, math.NewInt(100000).Uint64(), balance.Uint64())

	// Generate Key Rotation Msg
	account, err = ExecBin(t, ctx, xion.GetNode(),
		"keys", "show",
		xionUser.KeyName(),
		"--keyring-backend", keyring.BackendTest,
		"-p",
	)

	// add secondary authenticator to account. in this case, the same key but in a different position
	jsonExecMsgStr, err := GenerateTx(t, ctx, xion.GetNode(),
		xionUser.KeyName(),
		"xion", "add-authenticator", aaContractAddr,
		"--authenticator-id", "1",
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err)
	jsonExecMsg := []byte(jsonExecMsgStr)
	require.True(t, json.Valid(jsonExecMsg))

	rotateFile, err := os.CreateTemp("", "*-msg-exec-rotate-key.json")
	require.NoError(t, err)
	defer os.Remove(rotateFile.Name())

	_, err = rotateFile.Write(jsonExecMsg)
	require.NoError(t, err)

	err = UploadFileToContainer(t, ctx, xion.GetNode(), rotateFile)
	require.NoError(t, err)

	rotateFilePath := strings.Split(rotateFile.Name(), "/")

	_, err = ExecTx(t, ctx, xion.GetNode(),
		xionUser.KeyName(),
		"xion", "sign",
		xionUser.KeyName(),
		aaContractAddr,
		path.Join(xion.GetNode().HomeDir(), rotateFilePath[len(rotateFilePath)-1]),
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err)
	updatedContractState, err := ExecQuery(t, ctx, xion.GetNode(), "wasm", "contract-state", "smart", aaContractAddr, `{"authenticator_by_i_d":{ "id": 1 }}`)
	require.NoError(t, err)

	updatedPubKey, ok := updatedContractState["data"].(string)
	require.True(t, ok)

	updatedPubKeyRawJSON, err := base64.StdEncoding.DecodeString(updatedPubKey)
	require.NoError(t, err)
	var updatedPubKeyMap jsonAuthenticator

	err = json.Unmarshal(updatedPubKeyRawJSON, &updatedPubKeyMap)
	require.NoError(t, err)
	require.Equal(t, account["key"], updatedPubKeyMap["Secp256K1"]["pubkey"])

	// delete the original key
	jsonExecMsg = RawJSONMsgExecContractRemoveAuthenticator(aaContractAddr, aaContractAddr, 0)
	require.True(t, json.Valid(jsonExecMsg))

	removeFile, err := os.CreateTemp("", "*-msg-exec-remove-key.json")
	require.NoError(t, err)
	defer os.Remove(removeFile.Name())

	_, err = removeFile.Write(jsonExecMsg)
	require.NoError(t, err)

	err = UploadFileToContainer(t, ctx, xion.GetNode(), removeFile)
	require.NoError(t, err)

	removeFilePath := strings.Split(removeFile.Name(), "/")

	_, err = ExecTx(t, ctx, xion.GetNode(),
		xionUser.KeyName(),
		"xion", "sign",
		xionUser.KeyName(),
		aaContractAddr,
		path.Join(xion.GetNode().HomeDir(), removeFilePath[len(removeFilePath)-1]),
		"--chain-id", xion.Config().ChainID,
		"--authenticator-id", "1",
	)
	require.NoError(t, err)

	// validate original key was deleted
	updatedContractState, err = ExecQuery(t, ctx, xion.GetNode(), "wasm", "contract-state", "smart", aaContractAddr, `{"authenticator_i_ds":{}}`)
	require.NoError(t, err)
	resp := updatedContractState["data"]
	t.Logf("type: %v %T", resp, resp)
	rv := reflect.ValueOf(resp)
	if rv.Kind() == reflect.Slice {
		require.Equal(t, 1, rv.Len())
		require.Equal(t, uint8(1), uint8(rv.Index(0).Interface().(float64)))

	} else {
		require.Fail(t, "response not a slice")
	}
}

func GetAAContractAddress(t *testing.T, txDetails map[string]interface{}) string {
	logs, ok := txDetails["events"].([]interface{})
	require.True(t, ok)

	log, ok := logs[9].(map[string]interface{})
	require.True(t, ok)

	attributes, ok := log["attributes"].([]interface{})
	require.True(t, ok)

	attribute, ok := attributes[0].(map[string]interface{})
	require.True(t, ok)

	addr, ok := attribute["value"].(string)
	require.True(t, ok)

	return addr
}
