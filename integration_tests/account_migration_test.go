package integration_tests

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	signingv1beta1 "cosmossdk.io/api/cosmos/tx/signing/v1beta1"
	"cosmossdk.io/math"
	txsigning "cosmossdk.io/x/tx/signing"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/golang-jwt/jwt/v4"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	aatypes "github.com/larry0x/abstract-account/x/abstractaccount/types"
	"github.com/lestrrat-go/jwx/jwk"
	ibctest "github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
)

func TestAbstractAccountMigration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	t.Parallel()

	td := BuildXionChain(t, "0.0uxion", ModifyInterChainGenesis(ModifyInterChainGenesisFn{ModifyGenesisShortProposals, ModifyGenesisAAAllowedCodeIDs}, [][]string{{votingPeriod, maxDepositPeriod}, {votingPeriod, maxDepositPeriod}}))
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

	// prepare the JWT key and data
	fp, err := os.Getwd()
	require.NoError(t, err)

	// deploy the contract
	codeIDStr, err := xion.StoreContract(ctx, xionUser.FormattedAddress(),
		path.Join(fp, "integration_tests", "testdata", "contracts", "account_updatable-aarch64-previous.wasm"))
	require.NoError(t, err)

	predictedAddrs := addAccounts(t, ctx, xion, 50, codeIDStr, xionUser)

	// deploy the new contract
	newCodeIDStr, err := xion.StoreContract(ctx, xionUser.FormattedAddress(),
		path.Join(fp, "integration_tests", "testdata", "contracts", "account_updatable-aarch64.wasm"))
	require.NoError(t, err)

	// retrieve the new hash
	newCodeResp, err := ExecQuery(t, ctx, xion.GetNode(),
		"wasm", "code-info", newCodeIDStr)
	require.NoError(t, err)
	t.Logf("code response: %s", newCodeResp)

	CosmosChainUpgradeTest(t, &td, "xion", "upgrade", "v6")
	// todo: validate that verification or tx submission still works

	newCodeResp, err = ExecQuery(t, ctx, td.xionChain.GetNode(),
		"wasm", "code-info", newCodeIDStr)
	require.NoError(t, err)
	t.Logf("code response: %+v", newCodeResp)

	err = testutil.WaitForBlocks(ctx, int(blocksAfterUpgrade), td.xionChain)
	require.NoError(t, err, "chain did not produce blocks after upgrade")

	for _, predictedAddr := range predictedAddrs {
		rawUpdatedContractInfo, err := ExecQuery(t, ctx, td.xionChain.GetNode(),
			"wasm", "contract", predictedAddr.String())
		require.NoError(t, err)
		t.Logf("updated contract info: %s", rawUpdatedContractInfo)

		updatedContractInfo := rawUpdatedContractInfo["contract_info"].(map[string]interface{})
		updatedCodeID := updatedContractInfo["code_id"].(string)
		require.Equal(t, updatedCodeID, newCodeIDStr)
	}
}

func addAccounts(t *testing.T, ctx context.Context, xion *cosmos.CosmosChain, noOfAccounts int, codeIDStr string, xionUser ibc.Wallet) []sdk.AccAddress {
	predictedAddrs := make([]sdk.AccAddress, 0)
	sub := "integration-test-user"
	aud := "integration-test-project"

	authenticatorDetails := map[string]string{}
	authenticatorDetails["sub"] = sub
	authenticatorDetails["aud"] = aud

	authenticator := map[string]interface{}{}
	authenticator["Jwt"] = authenticatorDetails

	instantiateMsg := map[string]interface{}{}
	instantiateMsg["id"] = 0
	instantiateMsg["authenticator"] = authenticator

	codeResp, err := ExecQuery(t, ctx, xion.GetNode(),
		"wasm", "code-info", codeIDStr)
	require.NoError(t, err)
	t.Logf("code response: %s", codeResp)

	for i := 0; i < noOfAccounts; i++ {
		salt := fmt.Sprintf("%d", i)
		creatorAddr := types.AccAddress(xionUser.Address())
		codeHash, err := hex.DecodeString(codeResp["data_hash"].(string))
		require.NoError(t, err)
		predictedAddr := wasmkeeper.BuildContractAddressPredictable(codeHash, creatorAddr, []byte(salt), []byte{})
		t.Logf("predicted address: %s", predictedAddr.String())

		privateKeyBz, err := os.ReadFile("./integration_tests/testdata/keys/jwtRS256.key")
		require.NoError(t, err)
		privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyBz)
		require.NoError(t, err)
		t.Logf("private key: %v", privateKey)

		publicKey, err := jwk.New(privateKey)
		require.NoError(t, err)
		publicKeyJSON, err := json.Marshal(publicKey)
		require.NoError(t, err)
		t.Logf("public key: %s", publicKeyJSON)

		// sha256 the contract addr, as it expects
		signatureBz := sha256.Sum256([]byte(predictedAddr.String()))
		signature := base64.StdEncoding.EncodeToString(signatureBz[:])

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

		instantiateMsg["signature"] = []byte(output)
		instantiateMsgStr, err := json.Marshal(instantiateMsg)
		require.NoError(t, err)
		t.Logf("inst msg: %s", string(instantiateMsgStr))

		// register the account
		t.Logf("registering account: %s", instantiateMsgStr)
		registerCmd := []string{
			"abstract-account", "register",
			codeIDStr, string(instantiateMsgStr),
			"--salt", salt,
			"--funds", "10000uxion",
			"--chain-id", xion.Config().ChainID,
		}
		t.Logf("sender: %s", xionUser.FormattedAddress())
		t.Logf("register cmd: %s", registerCmd)

		txHash, err := ExecTx(t, ctx, xion.GetNode(), xionUser.KeyName(), registerCmd...)
		require.NoError(t, err)
		t.Logf("tx hash: %s", txHash)

		contractsResponse, err := ExecQuery(t, ctx, xion.GetNode(), "wasm", "contracts", codeIDStr)
		require.NoError(t, err)

		contract := contractsResponse["contracts"].([]interface{})[0].(string)

		err = testutil.WaitForBlocks(ctx, 1, xion)
		require.NoError(t, err)
		newBalance, err := xion.GetBalance(ctx, contract, xion.Config().Denom)
		require.NoError(t, err)
		require.Equal(t, int64(10_000), newBalance)

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
		predictedAddrs = append(predictedAddrs, predictedAddr)
	}
	return predictedAddrs
}

func TestSingleAbstractAccountMigration(t *testing.T) {
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

	// Store Wasm Contracts
	fp, err := os.Getwd()
	require.NoError(t, err)
	codeIDStr, err := xion.StoreContract(ctx, xionUser.FormattedAddress(), path.Join(fp,
		"integration_tests", "testdata", "contracts", "account_updatable-aarch64-previous.wasm"))
	require.NoError(t, err)
	t.Logf("loaded previous contract at ID %s", codeIDStr)

	migrateTargetCodeIDStr, err := xion.StoreContract(ctx, xionUser.FormattedAddress(), path.Join(fp,
		"integration_tests", "testdata", "contracts", "account_updatable-aarch64.wasm"))
	require.NoError(t, err)
	t.Logf("loaded new contract at ID %s", migrateTargetCodeIDStr)

	// retrieve the hash
	codeResp, err := ExecQuery(t, ctx, xion.GetNode(),
		"wasm", "code-info", codeIDStr)
	require.NoError(t, err)
	t.Logf("code response: %s", codeResp)

	sub := "integration-test-user"

	authenticatorDetails := map[string]interface{}{}
	authenticatorDetails["sub"] = sub
	authenticatorDetails["aud"] = aud
	// authenticatorDetails["id"] = 0

	authenticator := map[string]interface{}{}
	authenticator["Jwt"] = authenticatorDetails

	instantiateMsg := map[string]interface{}{}
	instantiateMsg["id"] = 0
	instantiateMsg["authenticator"] = authenticator

	// predict the contract address so it can be verified
	salt := "0"
	creatorAddr := types.AccAddress(xionUser.Address())
	codeHash, err := hex.DecodeString(codeResp["data_hash"].(string))
	require.NoError(t, err)
	predictedAddr := wasmkeeper.BuildContractAddressPredictable(codeHash, creatorAddr, []byte(salt), []byte{})
	t.Logf("predicted address: %s", predictedAddr.String())

	// b64 the contract address to use as the transaction hash
	signatureBz := sha256.Sum256([]byte(predictedAddr.String()))
	signature := base64.StdEncoding.EncodeToString(signatureBz[:])

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

	instantiateMsg["signature"] = []byte(output)
	instantiateMsgStr, err := json.Marshal(instantiateMsg)
	require.NoError(t, err)
	t.Logf("inst msg: %s", string(instantiateMsgStr))

	// register the account
	t.Logf("registering account: %s", instantiateMsgStr)
	registerCmd := []string{
		"abstract-account", "register",
		codeIDStr, string(instantiateMsgStr),
		"--salt", salt,
		"--funds", "10000uxion",
		"--chain-id", xion.Config().ChainID,
	}
	t.Logf("sender: %s", xionUser.FormattedAddress())
	t.Logf("register cmd: %s", registerCmd)

	txHash, err := ExecTx(t, ctx, xion.GetNode(), xionUser.KeyName(), registerCmd...)
	require.NoError(t, err)
	t.Logf("tx hash: %s", txHash)

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

	// Generate Msg Send without signatures
	jsonMsg := RawJSONMsgMigrateContract(account.GetAddress().String(), migrateTargetCodeIDStr)
	require.NoError(t, err)
	require.True(t, json.Valid(jsonMsg))

	tx, err := xion.Config().EncodingConfig.TxConfig.TxJSONDecoder()([]byte(jsonMsg))
	require.NoError(t, err)

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
	signatureBz = sha256.Sum256(signBytes)
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

	// confirm the new contract code ID
	rawUpdatedContractInfo, err := ExecQuery(t, ctx, td.xionChain.GetNode(),
		"wasm", "contract", account.GetAddress().String())
	require.NoError(t, err)
	t.Logf("updated contract info: %s", rawUpdatedContractInfo)

	updatedContractInfo := rawUpdatedContractInfo["contract_info"].(map[string]interface{})
	updatedCodeID := updatedContractInfo["code_id"].(string)
	require.Equal(t, updatedCodeID, migrateTargetCodeIDStr)
}
