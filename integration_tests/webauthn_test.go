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

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	xionapp "github.com/burnt-labs/xion/app"
	xiontypes "github.com/burnt-labs/xion/x/xion/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	aatypes "github.com/larry0x/abstract-account/x/abstractaccount/types"
	ibctest "github.com/strangelove-ventures/interchaintest/v7"
	"github.com/strangelove-ventures/interchaintest/v7/testutil"
	"github.com/stretchr/testify/require"
)

var deployerMnemonic = "decorate corn happy degree artist trouble color mountain shadow hazard canal zone hunt unfold deny glove famous area arrow cup under sadness salute item"

func TestWebAuthNAbstractAccount(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	td := BuildXionChain(t, "0.0uxion", ModifyInterChainGenesis(ModifyInterChainGenesisFn{ModifyGenesisShortProposals}, [][]string{{votingPeriod, maxDepositPeriod}}))
	xion, ctx := td.xionChain, td.ctx

	config := types.GetConfig()
	config.SetBech32PrefixForAccount("xion", "xionpub")

	// Create and Fund User Wallets
	t.Log("creating and funding user accounts")
	fundAmount := int64(10_000_000)
	// users := ibctest.GetAndFundTestUsers(t, ctx, "default", fundAmount, xion)
	deployerAddr, err := ibctest.GetAndFundTestUserWithMnemonic(ctx, "default", deployerMnemonic, fundAmount, xion)
	require.NoError(t, err)
	err = testutil.WaitForBlocks(ctx, 8, xion)
	require.NoError(t, err)
	t.Logf("created xion user %s", deployerAddr.FormattedAddress())

	xionUserBalInitial, err := xion.GetBalance(ctx, deployerAddr.FormattedAddress(), xion.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, fundAmount, xionUserBalInitial)

	// register any needed msg types
	xion.Config().EncodingConfig.InterfaceRegistry.RegisterImplementations(
		(*types.Msg)(nil),
		&xiontypes.MsgSetPlatformPercentage{},
		&xiontypes.MsgSend{},
		&wasmtypes.MsgInstantiateContract{},
		&wasmtypes.MsgStoreCode{},
		&aatypes.MsgUpdateParams{},
		&aatypes.MsgRegisterAccount{},
	)
	xion.Config().EncodingConfig.InterfaceRegistry.RegisterImplementations((*authtypes.AccountI)(nil), &aatypes.AbstractAccount{})
	xion.Config().EncodingConfig.InterfaceRegistry.RegisterImplementations((*cryptotypes.PubKey)(nil), &aatypes.NilPubKey{})
	// t.Log(xion.Config().EncodingConfig.InterfaceRegistry.ListImplementations("/xion.v1.Msg/Send"))
	fp, err := os.Getwd()
	require.NoError(t, err)

	// deploy the contract
	codeIDStr, err := xion.StoreContract(ctx, deployerAddr.FormattedAddress(),
		path.Join(fp, "testdata", "contracts", "account.wasm"))
	require.NoError(t, err)

	// retrieve the hash
	codeResp, err := ExecQuery(t, ctx, xion.FullNodes[0],
		"wasm", "code-info", codeIDStr)
	require.NoError(t, err)
	t.Logf("code response: %s", codeResp)

	// predict the contract address so it can be verified
	salt := "0"
	creatorAddr := types.AccAddress(deployerAddr.Address())
	codeHash, err := hex.DecodeString(codeResp["data_hash"].(string))
	require.NoError(t, err)
	predictedAddr := wasmkeeper.BuildContractAddressPredictable(codeHash, creatorAddr, []byte(salt), []byte{})
	t.Logf("predicted address: %s", predictedAddr.String())

	authenticatorDetails := map[string]interface{}{}
	authenticatorDetails["url"] = "https://xion-dapp-example-git-feat-faceid-burntfinance.vercel.app"
	authenticatorDetails["credential"] = "eyJ0eXBlIjoicHVibGljLWtleSIsImlkIjoiLUhKMVFvak1mb2NyTXZBM0FtVThwdXVsc1pVIiwicmF3SWQiOiItSEoxUW9qTWZvY3JNdkEzQW1VOHB1dWxzWlUiLCJhdXRoZW50aWNhdG9yQXR0YWNobWVudCI6ImNyb3NzLXBsYXRmb3JtIiwicmVzcG9uc2UiOnsiY2xpZW50RGF0YUpTT04iOiJleUowZVhCbElqb2lkMlZpWVhWMGFHNHVZM0psWVhSbElpd2lZMmhoYkd4bGJtZGxJam9pWlVkc2RtSnFSbmxPV0doNVducEJNVTVFWkhSaVYyaHVUa2RHTUUxSGNIRk9NMUl3VDFScmVXUllTbnBhUkZreFpESndNbU16V2pKbFNGSnFZVWRTZEZwdGNIaFBSRnBzV2toR2VHRnRUVEpPTW5RMklpd2liM0pwWjJsdUlqb2lhSFIwY0hNNkx5OTRhVzl1TFdSaGNIQXRaWGhoYlhCc1pTMW5hWFF0Wm1WaGRDMW1ZV05sYVdRdFluVnliblJtYVc1aGJtTmxMblpsY21ObGJDNWhjSEFpTENKamNtOXpjMDl5YVdkcGJpSTZabUZzYzJWOSIsImF0dGVzdGF0aW9uT2JqZWN0IjoibzJObWJYUmtibTl1WldkaGRIUlRkRzEwb0doaGRYUm9SR0YwWVZpWXNHTUJpRGNFcHBpTWZ4UTEwVFBDZTItRmFLckxlVGt2cHp4Y3puZ1RNdzFkQUFBQUFQdjhNQWNWVGs3TWpBdHVBZ1ZYMTcwQUZQaHlkVUtJekg2SEt6THdOd0psUEticnBiR1ZwUUVDQXlZZ0FTRllJTjM0ajlZVldDcjRMZjRPeUkwckEtNGlzeTRuX3J0d0FwSkFqV211akZBQUlsZ2d5bFZXQ0YxNDJWY3d6ZHF5U2FhbGpCWER1azlxNks4REl3SXM3WE5kdkFRIiwidHJhbnNwb3J0cyI6WyJoeWJyaWQiLCJpbnRlcm5hbCJdfSwiY2xpZW50RXh0ZW5zaW9uUmVzdWx0cyI6e319"
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

	txHash, err := ExecTx(t, ctx, xion.FullNodes[0], deployerAddr.KeyName(), registerCmd...)
	require.NoError(t, err)
	t.Logf("tx hash: %s", txHash)

	contractsResponse, err := ExecQuery(t, ctx, xion.FullNodes[0], "wasm", "contracts", codeIDStr)
	require.NoError(t, err)

	contract := contractsResponse["contracts"].([]interface{})[0].(string)

	// get the account from the chain. there might be a better way to do this
	accountResponse, err := ExecQuery(t, ctx, xion.FullNodes[0],
		"account", contract)
	require.NoError(t, err)
	t.Logf("account response: %s", accountResponse)

	delete(accountResponse, "@type")
	var account aatypes.AbstractAccount
	accountJSON, err := json.Marshal(accountResponse)
	require.NoError(t, err)

	encodingConfig := xionapp.MakeEncodingConfig()
	err = encodingConfig.Marshaler.UnmarshalJSON(accountJSON, &account)
	require.NoError(t, err)

	// create the raw tx
	sendMsg := fmt.Sprintf(`
	{
	 "body": {
	   "messages": [
	     {
	       "@type": "/cosmos.bank.v1beta1.MsgSend",
	       "from_address": "xion1z63j80gdeu0dcrlnc80w520347v6rfh4899s35qupyewv79hnjfqczcquh",
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
		`, deployerAddr.FormattedAddress(), "uxion")

	tx, err := encodingConfig.TxConfig.TxJSONDecoder()([]byte(sendMsg))
	require.NoError(t, err)
	txBuilder, err := encodingConfig.TxConfig.WrapTxBuilder(tx)
	require.NoError(t, err)
	testAcc := aatypes.AbstractAccount{Address: "xion1z63j80gdeu0dcrlnc80w520347v6rfh4899s35qupyewv79hnjfqczcquh",
		AccountNumber: 40, Sequence: 0}
	// create the sign bytes
	signerData := authsigning.SignerData{
		Address:       "xion1z63j80gdeu0dcrlnc80w520347v6rfh4899s35qupyewv79hnjfqczcquh",
		ChainID:       "xion-local-testnet-1",
		AccountNumber: 40,
		Sequence:      0,
		PubKey:        testAcc.GetPubKey(),
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

	signBytes, err := encodingConfig.TxConfig.SignModeHandler().GetSignBytes(signing.SignMode_SIGN_MODE_DIRECT, signerData, txBuilder.GetTx())
	require.NoError(t, err)
	t.Log("Sign bytes: ", signBytes)
	// our signature is the sha256 of the signbytes
	signatureBz := sha256.Sum256(signBytes)
	t.Log("Sign hash: ", signatureBz)
	challenge := base64.StdEncoding.EncodeToString(signatureBz[:])

	t.Log("challenge ", challenge)

	signedChallenge := "MEYCIQCp-oalKD6jeiA74nrPN3wFfLi8jD7z1T9dS1Vl-0yzzwIhANG2OIUXXS6gv2xF7yQOG_RLS0PV4PZSFRuBLXjABVxi"

	// add the auth index to the signature
	signedTokenBz := []byte(signedChallenge)
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

	jsonTx, err := encodingConfig.TxConfig.TxJSONEncoder()(txBuilder.GetTx())
	require.NoError(t, err)
	t.Logf("json tx: %s", jsonTx)

	output, err := ExecBroadcast(t, ctx, xion.FullNodes[0], jsonTx)
	require.NoError(t, err)
	t.Logf("output: %s", output)

	err = testutil.WaitForBlocks(ctx, 2, xion)
	require.NoError(t, err)
	newBalance, err := xion.GetBalance(ctx, contract, xion.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, int64(10_000-1337), newBalance)
}
