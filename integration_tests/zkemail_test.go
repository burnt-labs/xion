package integration_tests

import (
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
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	aatypes "github.com/burnt-labs/abstract-account/x/abstractaccount/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	ibctest "github.com/strangelove-ventures/interchaintest/v10"
	"github.com/strangelove-ventures/interchaintest/v10/testutil"
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
type ProofData struct {
	PiA      []string   `json:"pi_a"`
	PiB      [][]string `json:"pi_b"`
	PiC      []string   `json:"pi_c"`
	Protocol string     `json:"protocol"`
	Curve    string     `json:"curve"`
}

type Signature struct {
	Proof        ProofData `json:"proof"`
	PublicInputs []string  `json:"publicInputs"`
}

type QueryContractRequest struct {
	AuthenticatorById map[string]interface{} `json:"authenticator_by_i_d"`
}

func ToLittleEndian(b []byte) []byte {
	result := make([]byte, len(b))
	for i := 0; i < len(b); i++ {
		result[i] = b[len(b)-1-i]
	}
	return result
}

func TestZKEmailAuthenticator(t *testing.T) {
	ctx := t.Context()
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	xion := BuildXionChain(t)

	t.Parallel()

	config := types.GetConfig()
	config.SetBech32PrefixForAccount("xion", "xionpub")

	// Create and Fund User Wallets
	t.Log("creating and funding user accounts")
	fundAmount := math.NewInt(1_000_000_000_000)
	xionUser, err := ibctest.GetAndFundTestUserWithMnemonic(ctx, "default", deployerMnemonic, fundAmount, xion)
	require.NoError(t, err)
	currentHeight, _ := xion.Height(ctx)
	err = testutil.WaitForBlocks(ctx, int(currentHeight)+8, xion)
	require.NoError(t, err)
	t.Logf("created xion user %s", xionUser.FormattedAddress())

	_ = ibctest.GetAndFundTestUsers(t, ctx, "tmp", fundAmount, xion)

	// Store Abstract Account Contract
	fp, err := os.Getwd()
	require.NoError(t, err)

	accountCodeID, err := xion.StoreContract(ctx, xionUser.FormattedAddress(), path.Join(fp, "integration_tests", "testdata", "contracts", "xion_account.wasm"))
	require.NoError(t, err)

	signatureJSONPath := path.Join(fp, "integration_tests", "testdata", "keys", "signature.json")
	// Read the file
	fileContent, err := os.ReadFile(signatureJSONPath)
	require.NoError(t, err)

	var signature Signature
	err = json.Unmarshal(fileContent, &signature)
	require.NoError(t, err)

	emailSalt := signature.PublicInputs[32]
	fmt.Println(emailSalt)
	b64signture := "ewogICAgInByb29mIjogewogICAgICAgICJwaV9hIjogWwogICAgICAgICAgICAiNjA0MzY0MzQzMzE0MDY0MjU2OTI4MDg5ODI1OTU0MTEyODQzMTkwNzYzNTg3ODU0NzYxNDkzNTY4MTQ0MDgyMDY4MzAzODk2Mzc5MiIsCiAgICAgICAgICAgICI5OTkyMTMyMTkyNzc5MTEyODY1OTU4NjY3MzgxOTE1MTIwNTMyNDk3NDAxNDQ1ODYzMzgxNjkzMTI1NzA4ODc4NDEyODY3ODE5NDI5IiwKICAgICAgICAgICAgIjEiCiAgICAgICAgXSwKICAgICAgICAicGlfYiI6IFsKICAgICAgICAgICAgWwogICAgICAgICAgICAgICAgIjg1NzE1MDcwMzAzNjE1MTAwOTAwNDEzMDgzNDg4NTU3Nzg2MDk0NDU0NTMyMTEwNTI3MjU4MTE0OTYyMDI4ODE0ODkwMjM4NTQ0MCIsCiAgICAgICAgICAgICAgICAiMzMxMzQxOTk3MjQ2NjM0MjAzMDQ2NzcwMTg4MjEyNjg1MDUzNzQ5MTExNTQ0NjY4MTA5MzIyMjMzNTQ2ODg1NzMyMzIxMDY5NzI5NSIKICAgICAgICAgICAgXSwKICAgICAgICAgICAgWwogICAgICAgICAgICAgICAgIjIxNzEyNDQ1MzQ0MTcyNzk1OTU2MTAyMzYxOTkzNjQ3MjY4Nzc2Njc0NzI5MDAzNTY5NTg0NTA2MDQ3MTkwNjMwNDc0NjI1ODg3Mjk1IiwKICAgICAgICAgICAgICAgICIxMzE4MDEyNjYxOTc4NzY0NDk1MjQ3NTQ0MTQ1NDg0NDI5NDk5MTE5ODI1MTY2OTE5MTk2Mjg1MjQ1OTM1NTI2OTg4MTQ3ODU5NzA3NCIKICAgICAgICAgICAgXSwKICAgICAgICAgICAgWwogICAgICAgICAgICAgICAgIjEiLAogICAgICAgICAgICAgICAgIjAiCiAgICAgICAgICAgIF0KICAgICAgICBdLAogICAgICAgICJwaV9jIjogWwogICAgICAgICAgICAiNTYwODg3NDUzMDQxNTc2ODkwOTUzMTM3OTI5NzUwOTI1ODAyODM5ODQ2NTIwMTM1MTY4MDk1NTI3MDU4NDI4MDUyNDgwNzU2MzMyNyIsCiAgICAgICAgICAgICIxMjgyNTM4OTM3NTg1OTI5NDUzNzIzNjU2ODc2MzI3MDUwNjIwNjkwMTY0NjQzMjY0NDAwNzM0Mzk1NDg5MzQ4NTg2NDkwNTQwMTMxMyIsCiAgICAgICAgICAgICIxIgogICAgICAgIF0sCiAgICAgICAgInByb3RvY29sIjogImdyb3RoMTYiCiAgICB9LAogICAgInB1YmxpY0lucHV0cyI6IFsKICAgICAgICAiMjAxODcyMTQxNDAzODQwNDgyMDMyNyIsCiAgICAgICAgIjAiLAogICAgICAgICIwIiwKICAgICAgICAiMCIsCiAgICAgICAgIjAiLAogICAgICAgICIwIiwKICAgICAgICAiMCIsCiAgICAgICAgIjAiLAogICAgICAgICIwIiwKICAgICAgICAiNjYzMjM1MzcxMzA4NTE1NzkyNTUwNDAwODQ0MzA3ODkxOTcxNjMyMjM4NjE1NjE2MDYwMjIxODUzNjk2MTAyODA0NjQ2ODIzNzE5MiIsCiAgICAgICAgIjEyMDU3Nzk0NTQ3NDg1MjEwNTE2OTI4ODE3ODc0ODI3MDQ4NzA4ODQ0MjUyNjUxNTEwODc1MDg2MjU3NDU1MTYzNDE2Njk3NzQ2NTEyIiwKICAgICAgICAiMCIsCiAgICAgICAgIjEyNDQxMzU4ODAxMDkzNTU3MzEwMDQ0OTQ1NjQ2ODk1OTgzOTI3MDAyNzc1NzIxNTEzODQzOTgxNjk1NTAyNDczNjI3MTI5ODg4MyIsCiAgICAgICAgIjEyNTk4NzcxODUwNDg4MTE2ODcwMjgxNzM3Mjc1MTQwNTUxMTMxMTYyNjUxNTM5OTEyODExNTk1NzY4MzA1NTcwNjE2Mjg3OTA4MSIsCiAgICAgICAgIjEzODE3NDI5NDQxOTU2NjA3MzYzODkxNzM5ODQ3ODQ4MDIzMzc4MzQ2MjY1NTQ4MjI4MzQ4OTc3ODQ3NzAzMjEyOTg2MDQxNjMwOCIsCiAgICAgICAgIjg3MTY0NDI5OTM1MTgzNTMwMjMxMTA2NTI0MjM4NzcyNDY5MDgzMDIxMzc2NTM2ODU3NTQ3NjAxMjg2MzUwNTExODk1OTU3MDQyIiwKICAgICAgICAiMTU5NTA4OTk1NTU0ODMwMjM1NDIyODgxMjIwMjIxNjU5MjIyODgyNDE2NzAxNTM3Njg0MzY3OTA3MjYyNTQxMDgxMTgxMTA3MDQxIiwKICAgICAgICAiMjE2MTc3ODU5NjMzMDMzOTkzNjE2NjA3NDU2MDEwOTg3ODcwOTgwNzIzMjE0ODMyNjU3MzA0MjUwOTI5MDUyMDU0Mzg3NDUxMjUxIiwKICAgICAgICAiMTM2ODcwMjkzMDc3NzYwMDUxNTM2NTE0Njg5ODE0NTI4MDQwNjUyOTgyMTU4MjY4MjM4OTI0MjExNDQzMTA1MTQzMzE1MzEyOTc3IiwKICAgICAgICAiMjA5MDI3NjQ3MjcxOTQxNTQwNjM0MjYwMTI4MjI3MTM5MTQzMzA1MjEyNjI1NTMwMTMwOTg4Mjg2MzA4NTc3NDUxOTM0NDMzNjA0IiwKICAgICAgICAiMjE2MDQxMDM3NDgwODE2NTAxODQ2MzQ4NzA1MzUzNzM4MDc5Nzc1ODAzNjIzNjA3MzczNjY1Mzc4NDk5ODc2NDc4NzU3NzIxOTU2IiwKICAgICAgICAiMTg0MDk5ODA4ODkyNjA2MDYxOTQyNTU5MTQxMDU5MDgxNTI3MjYyODM0ODU5NjI5MTgxNTgxMjcwNTg1OTA4NTI5MDE0MDAwNDgzIiwKICAgICAgICAiMTczOTI2ODIxMDgyMzA4MDU2ODI5NDQxNzczODYwNDgzODQ5MTI4NDA0OTk2MDg0OTMyOTE5NTA1OTQ2ODAyNDg4MzY3OTg5MDcwIiwKICAgICAgICAiMTM2NDk4MDgzMzMyOTAwMzIxMjE1NTI2MjYwODY4NTYyMDU2NjcwODkyNDEyOTMyNjcxNTE5NTEwOTgxNzA0NDI3OTA1NDMwNTc4IiwKICAgICAgICAiMCIsCiAgICAgICAgIjAiLAogICAgICAgICIwIiwKICAgICAgICAiMCIsCiAgICAgICAgIjAiLAogICAgICAgICIwIiwKICAgICAgICAiMCIsCiAgICAgICAgIjAiLAogICAgICAgICIxOTQ0NjQyNzYwNTAyNjQyODMzMjY5NzQ0NTE3MzI0NTEyOTcwMzQyODc4NDM1NjY2Mzk5ODUzMzczNzQzNDkzNTkyNTM5MTIxMDg0MCIsCiAgICAgICAgIjEiLAogICAgICAgICIxNDU0NjQyMDgxMzA5MzMyMTY2NzkzNzQ4NzM0Njg3MTA2NDcxNDciLAogICAgICAgICIwIiwKICAgICAgICAiMCIsCiAgICAgICAgIjAiCiAgICBdCn0="
	fmt.Println(b64signture)
	// create allowed email hosts json marshalled string
	allowedEmailHosts := []string{"kushal@burnt.com", "jose@burnt.com", "jane@burnt.com"}
	allowedEmailHostsJSON, err := json.Marshal(allowedEmailHosts)
	require.NoError(t, err)
	allowedEmailHostsString := string(allowedEmailHostsJSON)

	// signatre a conjunction of (email_salt, proof)
	//
	//
	// Register Abstract Account Contract (Ensuring Fixed Address)
	registeredTxHash, err := ExecTx(t, ctx, xion.GetNode(),
		xionUser.KeyName(), "xion", "register",
		accountCodeID,
		xionUser.KeyName(),
		"--funds", "2000000000uxion",
		"--salt", "0",
		"--authenticator", "Secp256K1",
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err)
	err = testutil.WaitForBlocks(ctx, 2, xion)
	require.NoError(t, err)

	txDetails, err := ExecQuery(t, ctx, xion.GetNode(), "tx", registeredTxHash)
	require.NoError(t, err)
	t.Logf("TxDetails: %s", txDetails)
	aaContractAddr := GetAAContractAddress(t, txDetails)

	// send a execute msg to add a zkemail authenticator to the account
	// TODO: update ZKEmail id, email_salt, signature
	require.NoError(t, err)
	authExecuteMsg := fmt.Sprintf(
		`{"add_auth_method":{"add_authenticator":{"ZKEmail": {"id": 1, "email_salt": "%s", "allowed_email_hosts": %s, "signature": "%s"}}}}`,
		emailSalt,
		allowedEmailHostsString,
		b64signture,
	)

	t.Logf("auth execute msg: %s", authExecuteMsg)

	msgExec := &wasmtypes.MsgExecuteContract{
		Sender:   aaContractAddr, // contract is the sender in this case
		Contract: aaContractAddr, // target contract address is also the AA contract
		Msg:      []byte(authExecuteMsg),
		Funds:    types.Coins{}, // no funds attached
	}

	txBuilder := xion.Config().EncodingConfig.TxConfig.NewTxBuilder()
	err = txBuilder.SetMsgs(msgExec)
	require.NoError(t, err)

	txBuilder.SetFeeAmount(types.Coins{{Denom: xion.Config().Denom, Amount: math.NewInt(100000)}})
	txBuilder.SetGasLimit(500000)

	unsignedTxBz, err := xion.Config().EncodingConfig.TxConfig.TxJSONEncoder()(txBuilder.GetTx())
	require.NoError(t, err)

	file, err := os.CreateTemp("", "*-auth.json")
	require.NoError(t, err)
	_, err = file.Write(unsignedTxBz)
	require.NoError(t, err)
	err = UploadFileToContainer(t, ctx, xion.GetNode(), file)
	require.NoError(t, err)

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
	txDetails, err = ExecQuery(t, ctx, xion.GetNode(), "tx", txHash)
	require.NoError(t, err)
	fmt.Println(txDetails)

	// Query the contract to verify the zk-email authenticator was created
	queryMsg := QueryContractRequest{
		AuthenticatorById: map[string]any{"id": 1},
	}

	var queryResult map[string]any
	err = xion.QueryContract(ctx, aaContractAddr, queryMsg, &queryResult)
	require.NoError(t, err)

	var response map[string]any
	data := queryResult["data"].(string)
	// base64 decode the data
	decodedData, err := base64.StdEncoding.DecodeString(data)
	require.NoError(t, err)
	// unmarshal the decoded data
	err = json.Unmarshal(decodedData, &response)
	require.NoError(t, err)

	// Verify the authenticator type is ZKEmail
	require.Contains(t, response, "ZKEmail", "Response should contain ZKEmail field")

	// Wait for a few blocks to ensure query is up to date
	err = testutil.WaitForBlocks(ctx, 2, xion)
	require.NoError(t, err)

	// Create a bank send message from the AA contract to a recipient
	// Note if the tx changes, a new proof will need to be generated
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
	fmt.Printf("signer data: %v\n", signerData)

	txBuilder, err = xion.Config().EncodingConfig.TxConfig.WrapTxBuilder(tx)
	require.NoError(t, err)

	// Hardcoded proof (pre-generated externally)
	proofBz, err := os.ReadFile(path.Join(fp, "integration_tests", "testdata", "keys", "zkproof.json"))
	if err != nil {
		t.Fatalf("failed to read vkey.json file: %v", err)
	}

	var proof map[string]any
	err = json.Unmarshal(proofBz, &proof)
	require.NoError(t, err)

	sigBz, err := json.Marshal(signature)
	require.NoError(t, err)

	// prepend auth index to signature
	proofBz = append([]byte{uint8(1)}, sigBz...)

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

	txBuilder.SetFeeAmount(types.Coins{{Denom: xion.Config().Denom, Amount: math.NewInt(60_000)}})
	txBuilder.SetGasLimit(2_000_000) // 2 million because verification takes a lot of gas

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

	signBytes64 := base64.StdEncoding.EncodeToString(signBytes)
	t.Logf("sign bytes: %s %s %v", signBytes64, string(signBytes), signBytes)

	jsonTx, err := xion.Config().EncodingConfig.TxConfig.TxJSONEncoder()(txBuilder.GetTx())
	require.NoError(t, err)
	t.Logf("json tx: %s", jsonTx)

	output, err := ExecBroadcast(t, ctx, xion.GetNode(), jsonTx)
	t.Logf("tx details: %s", output)
	require.NoError(t, err)

	err = testutil.WaitForBlocks(ctx, 2, xion)
	require.NoError(t, err)
	recipientBalance, err := xion.GetBalance(ctx, recipient, xion.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, int64(100_000), recipientBalance.Int64())
	require.NoError(t, err)
}
