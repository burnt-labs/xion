package integration_tests

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"testing"

	signingv1beta1 "cosmossdk.io/api/cosmos/tx/signing/v1beta1"
	"cosmossdk.io/math"
	txsigning "cosmossdk.io/x/tx/signing"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"google.golang.org/protobuf/types/known/anypb"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/dvsekhvalnov/jose2go/base64url"
	aatypes "github.com/larry0x/abstract-account/x/abstractaccount/types"
	ibctest "github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
)

var deployerMnemonic = "decorate corn happy degree artist trouble color mountain shadow hazard canal zone hunt unfold deny glove famous area arrow cup under sadness salute item"

func setupChain(t *testing.T) (TestData, ibc.Wallet, []byte, string, error) {
	td := BuildXionChain(t, "0.0uxion", ModifyInterChainGenesis(ModifyInterChainGenesisFn{ModifyGenesisShortProposals}, [][]string{{votingPeriod, maxDepositPeriod}}))
	xion, ctx := td.xionChain, td.ctx

	config := types.GetConfig()
	config.SetBech32PrefixForAccount("xion", "xionpub")

	// Create and Fund User Wallets
	t.Log("creating and funding user accounts")
	fundAmount := math.NewInt(10_000_000)
	// users := ibctest.GetAndFundTestUsers(t, ctx, "default", fundAmount, xion)
	deployerAddr, err := ibctest.GetAndFundTestUserWithMnemonic(ctx, "default", deployerMnemonic, fundAmount, xion)
	require.NoError(t, err)
	err = testutil.WaitForBlocks(ctx, 8, xion)
	require.NoError(t, err)
	t.Logf("created xion user %s", deployerAddr.FormattedAddress())

	xionUserBalInitial, err := xion.GetBalance(ctx, deployerAddr.FormattedAddress(), xion.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, fundAmount, xionUserBalInitial)

	fp, err := os.Getwd()
	require.NoError(t, err)

	// deploy the contract
	codeIDStr, err := xion.StoreContract(ctx, deployerAddr.FormattedAddress(),
		path.Join(fp, "integration_tests", "testdata", "contracts", "account_updatable-aarch64.wasm"))
	require.NoError(t, err)

	// retrieve the hash
	codeResp, err := ExecQuery(t, ctx, xion.GetNode(),
		"wasm", "code-info", codeIDStr)
	require.NoError(t, err)
	t.Logf("code response: %s", codeResp)

	codeHash, err := hex.DecodeString(codeResp["data_hash"].(string))
	require.NoError(t, err)

	return td, deployerAddr, codeHash, codeIDStr, nil
}

func TestWebAuthNAbstractAccount(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	td, deployerAddr, codeHash, codeIDStr, err := setupChain(t)
	require.NoError(t, err)

	xion, ctx := td.xionChain, td.ctx

	// predict the contract address so it can be verified
	salt := "0"
	creatorAddr := types.AccAddress(deployerAddr.Address())
	require.NoError(t, err)
	predictedAddr := wasmkeeper.BuildContractAddressPredictable(codeHash, creatorAddr, []byte(salt), []byte{})
	t.Logf("predicted address: %s", predictedAddr.String())

	authenticatorDetails := map[string]interface{}{}
	authenticatorDetails["url"] = "https://xion-dapp-example-git-feat-faceid-burntfinance.vercel.app"
	/*
		The following is a valid public key response from a webauthn authenticator the using above url. This will need to be updated from time to time when the accounts contract is updated.
		To regenerate, use the following steps:
		1. Run this test and make note of the `predicted address` output above.
		2. Go to the url above and open the developer tools console
		3. Enter the `predicted address` in the second field within the WebAuthN section (populated with the string "test-challenge" by default)
		4. Click "Register" and copy the result from the console
	*/
	cred := CreateWebAuthNAttestationCred(t, base64url.Encode([]byte(predictedAddr.String())))
	authenticatorDetails["credential"] = cred
	authenticatorDetails["id"] = 0

	authenticator := map[string]interface{}{}
	authenticator["Passkey"] = authenticatorDetails

	instantiateMsg := map[string]interface{}{}
	instantiateMsg["authenticator"] = authenticator

	instantiateMsgStr, err := json.Marshal(instantiateMsg)
	require.NoError(t, err)

	registerCmd := []string{
		"abstract-account", "register",
		codeIDStr, string(instantiateMsgStr),
		"--salt", salt,
		"--chain-id", xion.Config().ChainID,
	}
	t.Logf("sender: %s", deployerAddr.FormattedAddress())

	txHash, err := ExecTx(t, ctx, xion.GetNode(), deployerAddr.KeyName(), registerCmd...)
	require.NoError(t, err)
	t.Logf("tx hash: %s", txHash)

	contractsResponse, err := ExecQuery(t, ctx, xion.GetNode(), "wasm", "contracts", codeIDStr)
	require.NoError(t, err)

	contract := contractsResponse["contracts"].([]interface{})[0].(string)

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

	err = xion.SendFunds(ctx, deployerAddr.FormattedAddress(), ibc.WalletAmount{Address: contract, Denom: "uxion", Amount: math.NewInt(10_000)})
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
		`, contract, deployerAddr.FormattedAddress(), "uxion")

	tx, err := xion.Config().EncodingConfig.TxConfig.TxJSONDecoder()([]byte(sendMsg))
	require.NoError(t, err)
	txBuilder, err := xion.Config().EncodingConfig.TxConfig.WrapTxBuilder(tx)
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
	challenge := base64url.Encode([]byte(base64.StdEncoding.EncodeToString(signatureBz[:])))

	t.Log("challenge ", challenge)

	/*
		The following is a valid signed challenge from a webauthn authenticator the using above url. This will need to be updated from time to time when the accounts contract is updated.
		To regenerate, use the following steps:
		1. Run this test and make note of the `challenge` output above.
		2. Go to the url above and open the developer tools console.
		3. Enter the `challenge` in the first field within the WebAuthN section (populated with the string "test-challenge" by default)
		4. Click "Sign" and copy the result from the console
	*/
	signedChallenge := CreateWebAuthNSignature(t, challenge, predictedAddr.String())
	sigBytes := append([]byte{0}, signedChallenge...)

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

	output, err := ExecBroadcast(t, ctx, xion.GetNode(), jsonTx)
	require.NoError(t, err)
	t.Logf("output: %s", output)

	err = testutil.WaitForBlocks(ctx, 2, xion)
	require.NoError(t, err)
	newBalance, err := xion.GetBalance(ctx, contract, xion.Config().Denom)
	require.NoError(t, err)
	require.True(t, math.NewInt(10_000-1337).Equal(newBalance))
}
