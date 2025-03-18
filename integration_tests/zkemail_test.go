package integration_tests

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"
	"testing"

	signingv1beta1 "cosmossdk.io/api/cosmos/tx/signing/v1beta1"
	"cosmossdk.io/math"
	txsigning "cosmossdk.io/x/tx/signing"
	"github.com/CosmWasm/wasmd/x/wasm"
	dkimTypes "github.com/burnt-labs/xion/x/dkim/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	aatypes "github.com/larry0x/abstract-account/x/abstractaccount/types"
	ibctest "github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/anypb"
)

type SnarkJSVkey struct {
	VkAlpha1 [3]string    `json:"vk_alpha_1"`
	VkBeta2  [3][2]string `json:"vk_beta_2"`
	VkGamma2 [3][2]string `json:"vk_gamma_2"`
	VkDelta2 [3][2]string `json:"vk_delta_2"`
	IC       [][3]string  `json:"IC"`
}
type ZKVerificationInstantiateMsg struct {
	Vkey SnarkJSVkey `json:"vkey"`
}

type QueryContractRequest struct {
	AuthenticatorById map[string]interface{} `json:"authenticator_by_i_d"`
}

func TestZKEmailAuthenticator(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()

	dkimDomain := "google.com"
	dkimSelector := "20230601"
	dkimPubkey := "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA4zd3nfUoLHWFbfoPZzAb8bvjsFIIFsNypweLuPe4M+vAP1YxObFxRnpvLYz7Z+bORKLber5aGmgFF9iaufsH1z0+aw8Qex7uDaafzWoJOM/6lAS5iI0JggZiUkqNpRQLL7H6E7HcvOMC61nJcO4r0PwLDZKwEaCs8gUHiqRn/SS3wqEZX29v/VOUVcI4BjaOzOCLaz7V8Bkwmj4Rqq4kaLQQrbfpjas1naScHTAmzULj0Rdp+L1vVyGitm+dd460PcTIG3Pn+FYrgQQo2fvnTcGiFFuMa8cpxgfH3rJztf1YFehLWwJWgeXTriuIyuxUabGdRQu7vh7GrObTsHmIHwIDAQAB"
	gPubKeyHash, err := dkimTypes.ComputePoseidonHash(dkimPubkey)
	require.NoError(t, err)
	gPubKeyBz, _ := json.Marshal([]Dkim{
		{
			Domain:       dkimDomain,
			Selector:     dkimSelector,
			PubKey:       dkimPubkey,
			PoseidonHash: base64.StdEncoding.EncodeToString(gPubKeyHash.Bytes()),
		},
	})

	td := BuildXionChain(t, "0.0uxion", ModifyInterChainGenesis(ModifyInterChainGenesisFn{ModifyGenesisShortProposals, ModifyGenesisDKIMRecords}, [][]string{{votingPeriod, maxDepositPeriod}, {string(gPubKeyBz)}}))
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

	// Store Abstract Account Contract
	fp, err := os.Getwd()
	require.NoError(t, err)

	accountCodeID, err := xion.StoreContract(ctx, xionUser.FormattedAddress(), path.Join(fp, "integration_tests", "testdata", "contracts", "xion-account.wasm"))
	require.NoError(t, err)

	// Store ZKEmail Verification Contract
	zkemailCodeID, err := xion.StoreContract(ctx, xionUser.FormattedAddress(), path.Join(fp, "integration_tests", "testdata", "contracts", "zkemail.wasm"))
	require.NoError(t, err)

	// Instantiate ZKEmail Verification Contract
	// read the vkey from the vkey.json file
	vkeyBz, err := os.ReadFile(path.Join(fp, "integration_tests", "testdata", "keys", "vkey.json"))
	if err != nil {
		t.Fatalf("failed to read vkey.json file: %v", err)
	}
	var vkey SnarkJSVkey
	err = json.Unmarshal(vkeyBz, &vkey)
	require.NoError(t, err)
	instantiateMsg := ZKVerificationInstantiateMsg{Vkey: vkey}

	vkeyJsonBz, err := json.Marshal(instantiateMsg)
	require.NoError(t, err)

	verificationContractAddress, err := xion.InstantiateContract(ctx, xionUser.FormattedAddress(), zkemailCodeID, string(vkeyJsonBz), true, "--gas", "400000", "--gas-prices", "0.025uxion", "--gas-adjustment", "1.5")
	require.NoError(t, err)

	// Register Abstract Account Contract (Ensuring Fixed Address)
	registeredTxHash, err := ExecTx(t, ctx, xion.GetNode(),
		xionUser.KeyName(), "xion", "register",
		accountCodeID,
		xionUser.KeyName(),
		"--funds", "1000000uxion",
		"--salt", "0",
		"--authenticator", "Secp256K1",
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err)

	txDetails, err := ExecQuery(t, ctx, xion.GetNode(), "tx", registeredTxHash)
	require.NoError(t, err)
	t.Logf("TxDetails: %s", txDetails)
	aaContractAddr := GetAAContractAddress(t, txDetails)

	// this commitement is gotten from the email address and a salt currently "testuseronexion@gmail.com", "XRhMS5Nc2dTZW5kEpAB"
	emailCommitment := "6293004408188449527623842124792388138908023069134722952437088476138094090885"
	emailHash := base64.StdEncoding.EncodeToString([]byte(emailCommitment))

	// send a execute msg to add a zkemail authenticator to the account
	_, err = xion.ExecuteContract(ctx, aaContractAddr, xionUser.FormattedAddress(), fmt.Sprintf(`{"authenticator": {"ZKEmail": {"id": 1, "verification_contract": "%s", "email_hash": "%s", "dkim_domain": "%s"}}}`, verificationContractAddress, emailHash, dkimDomain))
	authExecuteMsg := fmt.Sprintf(
		`{"add_auth_method":{"add_authenticator":{"ZKEmail": {"id": 1, "verification_contract": "%s", "email_hash": "%s", "dkim_domain": "%s"}}}}`,
		verificationContractAddress,
		emailHash,
		dkimDomain,
	)

	msgExec := &wasm.MsgExecuteContract{
		Sender:   aaContractAddr, // contract is the sender in this case
		Contract: aaContractAddr, // target contract address is also the AA contract
		Msg:      []byte(authExecuteMsg),
		Funds:    types.Coins{}, // no funds attached
	}

	txBuilder := xion.Config().EncodingConfig.TxConfig.NewTxBuilder()
	err = txBuilder.SetMsgs(msgExec)
	require.NoError(t, err)

	txBuilder.SetFeeAmount(types.Coins{{Denom: xion.Config().Denom, Amount: math.NewInt(100000)}})
	txBuilder.SetGasLimit(200000)

	unsignedTxBz, err := xion.Config().EncodingConfig.TxConfig.TxJSONEncoder()(txBuilder.GetTx())
	require.NoError(t, err)

	file, err := os.CreateTemp("", "*-auth.json")
	require.NoError(t, err)
	_, err = file.Write(unsignedTxBz)
	require.NoError(t, err)
	err = UploadFileToContainer(t, ctx, xion.GetNode(), file)
	require.NoError(t, err)
	t.Log("Unsigned authenticator tx uploaded to container")

	configFilePath := strings.Split(file.Name(), "/")

	cmd := []string{
		"xion", "sign", xionUser.KeyName(), aaContractAddr, path.Join(xion.GetNode().HomeDir(), configFilePath[len(configFilePath)-1]),
		"--chain-id", xion.Config().ChainID,
		"--gas-prices", "1uxion", "--gas-adjustment", "1.4",
		"--gas", "400000",
		"-y",
	}

	txHash, err := ExecTx(t, ctx, xion.GetNode(), xionUser.KeyName(), cmd...)
	require.NoError(t, err)
	// Wait for the transaction to be included in a block
	err = testutil.WaitForBlocks(ctx, 2, xion)
	require.NoError(t, err)

	// Query the transaction result
	_, err = ExecQuery(t, ctx, xion.GetNode(), "tx", txHash)
	require.NoError(t, err)
	// Query the contract to verify the zk-email authenticator was created
	queryMsg := QueryContractRequest{
		AuthenticatorById: map[string]any{"id": 1},
	}

	var queryResult map[string]any
	err = xion.QueryContract(ctx, aaContractAddr, queryMsg, &queryResult)
	require.NoError(t, err)

	var response map[string]any
	data := queryResult["data"].(string)
	//base64 decode the data
	decodedData, err := base64.StdEncoding.DecodeString(data)
	require.NoError(t, err)
	//unmarshal the decoded data
	err = json.Unmarshal(decodedData, &response)
	require.NoError(t, err)

	// Verify the authenticator type is ZKEmail
	require.Contains(t, response, "ZKEmail", "Response should contain ZKEmail field")

	// Wait for a few blocks to ensure query is up to date
	err = testutil.WaitForBlocks(ctx, 2, xion)
	require.NoError(t, err)

	// Create a bank send message from the AA contract to a recipient
	recipient := "xion1qaf2xflx5j3agtlvqk5vhjpeuhl6g45hxshwqj"
	jsonMsg := RawJSONMsgSend(t, aaContractAddr, recipient, "uxion")
	require.NoError(t, err)

	tx, err := xion.Config().EncodingConfig.TxConfig.TxJSONDecoder()([]byte(jsonMsg))
	require.NoError(t, err)

	// get the account from the chain. there might be a better way to do this
	accountResponse, err := ExecQuery(t, ctx, xion.GetNode(),
		"auth", "account", aaContractAddr)
	require.NoError(t, err)
	t.Logf("account response: %s", accountResponse)

	ac, ok := accountResponse["account"]
	require.True(t, ok)

	ac2, ok := ac.(map[string]any)
	require.True(t, ok)

	acData, ok := ac2["value"]
	require.True(t, ok)

	accountJSON, err := json.Marshal(acData)
	require.NoError(t, err)

	var account aatypes.AbstractAccount
	err = xion.Config().EncodingConfig.Codec.UnmarshalJSON(accountJSON, &account)
	require.NoError(t, err)

	// create the sign byte
	pubKey := account.GetPubKey()
	anyPk, err := codectypes.NewAnyWithValue(pubKey)
	signerData := txsigning.SignerData{
		Address:       aaContractAddr,
		ChainID:       xion.Config().ChainID,
		AccountNumber: account.GetAccountNumber(),
		Sequence:      account.GetSequence(),
		PubKey: &anypb.Any{
			TypeUrl: anyPk.TypeUrl,
			Value:   anyPk.Value,
		},
	}

	txBuilder, err = xion.Config().EncodingConfig.TxConfig.WrapTxBuilder(tx)
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

	signBytesHash := sha256.Sum256(signBytes)
	signBytes64 := base64.StdEncoding.EncodeToString(signBytesHash[:])
	t.Logf("sign bytes hash: %s", signBytes64)

	// Hardcoded proof (pre-generated externally)
	proofBz, err := os.ReadFile(path.Join(fp, "integration_tests", "testdata", "keys", "zkproof.json"))
	if err != nil {
		t.Fatalf("failed to read vkey.json file: %v", err)
	}

	type Signature struct {
		Proof    map[string]any `json:"proof"`
		DkimHash string         `json:"dkim_hash"`
	}

	var proof map[string]any
	err = json.Unmarshal(proofBz, &proof)
	require.NoError(t, err)

	sig := &Signature{
		Proof:    proof,
		DkimHash: base64.StdEncoding.EncodeToString(gPubKeyHash.Bytes()),
	}
	sigBz, err := json.Marshal(sig)
	require.NoError(t, err)

	// prepend auth index to signature
	proofBz = append([]byte{1}, sigBz...)

	sigData := signing.SingleSignatureData{
		SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
		Signature: proofBz,
	}

	sigV2 := signing.SignatureV2{
		PubKey:   account.GetPubKey(),
		Data:     &sigData,
		Sequence: account.GetSequence(),
	}

	err = txBuilder.SetSignatures(sigV2)
	require.NoError(t, err)

	jsonTx, err := xion.Config().EncodingConfig.TxConfig.TxJSONEncoder()(txBuilder.GetTx())
	require.NoError(t, err)
	t.Logf("json tx: %s", jsonTx)

	output, err := ExecBroadcastWithFlags(t, ctx, xion.GetNode(), jsonTx, "--gas", "200000", "--gas-prices", "0.025uxion", "--gas-adjustment", "1.5")
	t.Logf("output: %s", output)
	require.NoError(t, err)

	err = testutil.WaitForBlocks(ctx, 2, xion)
	require.NoError(t, err)
	newBalance, err := xion.GetBalance(ctx, aaContractAddr, xion.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, int64(1_000_000-100_000), newBalance.Int64())
	recipientBalance, err := xion.GetBalance(ctx, recipient, xion.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, int64(100_000), recipientBalance.Int64())
	require.NoError(t, err)
}
