package integration_tests

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path"
	"strings"
	"testing"

	signingv1beta1 "cosmossdk.io/api/cosmos/tx/signing/v1beta1"
	"cosmossdk.io/math"
	txsigning "cosmossdk.io/x/tx/signing"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	aatypes "github.com/burnt-labs/abstract-account/x/abstractaccount/types"
	dkimTypes "github.com/burnt-labs/xion/x/dkim/types"
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
type AuthExecuteMsg struct {
	AddAuthMethod struct {
		AddAuthenticator struct {
			ZKEmail struct {
				ID        int             `json:"id"`
				EmailSalt string          `json:"email_salt"`
				Signature json.RawMessage `json:"signature"`
			} `json:"ZKEmail"`
		} `json:"add_authenticator"`
	} `json:"add_auth_method"`
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

	dkimDomain := "gmail.com"
	// dkimSelector := "20230601"
	dkimPubkey := "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAntvSKT1hkqhKe0xcaZ0x+QbouDsJuBfby/S82jxsoC/SodmfmVs2D1KAH3mi1AqdMdU12h2VfETeOJkgGYq5ljd996AJ7ud2SyOLQmlhaNHH7Lx+Mdab8/zDN1SdxPARDgcM7AsRECHwQ15R20FaKUABGu4NTbR2fDKnYwiq5jQyBkLWP+LgGOgfUF4T4HZb2PY2bQtEP6QeqOtcW4rrsH24L7XhD+HSZb1hsitrE0VPbhJzxDwI4JF815XMnSVjZgYUXP8CxI1Y0FONlqtQYgsorZ9apoW1KPQe8brSSlRsi9sXB/tu56LmG7tEDNmrZ5XUwQYUUADBOu7t1niwXwIDAQAB"
	gPubKeyHash, err := dkimTypes.ComputePoseidonHash(dkimPubkey)
	require.NoError(t, err)

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

	emailSalt := signature.PublicInputs[len(signature.PublicInputs)-2]
	fmt.Println(emailSalt)
	// b64signture := "ewogICAgInByb29mIjogewogICAgICAgICJwaV9hIjogWwogICAgICAgICAgICAiMjU2NzQ5ODMwOTA5NTk0NTEyMzAwMTkxNTUyNTQyNTY3NTU5NzkwNTk5OTg1MTc2MDQ3ODgyNTA0NTUyNjY1MTY4MTIxNTYyNjMzMSIsCiAgICAgICAgICAgICIxNDk5OTQ4ODg1NDAwMTcyOTA5NjI2NDI2Mjc2NTQ4MTU0OTUyMDQxOTExMDEyMTcwNjYwNDA5MTM4Mjc5OTMzNTc2ODEzODM1OTcyOSIsCiAgICAgICAgICAgICIxIgogICAgICAgIF0sCiAgICAgICAgInBpX2IiOiBbCiAgICAgICAgICAgIFsKICAgICAgICAgICAgICAgICIxNzg5ODM5MTg1MzMwNTI1MDE2NTM2NDgwMzU3MjkxNDA0NjIxNzE0Mzg0NjA1OTgzMjQyMTk5ODExMzAzMDU3NzE2MjE4ODQ1MzMxMCIsCiAgICAgICAgICAgICAgICAiNDQ5NzEzNzEyNTY3ODg4MDg3MjIxOTE1MTAzNzA5MTA2ODI1MzI1ODg1NzA4Mjk5NzQyNDA2OTIxNjgyMjQzMTg0OTkyNTgyMjgzNiIKICAgICAgICAgICAgXSwKICAgICAgICAgICAgWwogICAgICAgICAgICAgICAgIjE5MzMwMDU1NTkwODg0MzA5OTUwNTUyMTYyNTU4NzQyNjE0NTM1MTkwNjc2NzM5MzA5MjgzMTY3Mjg3Mjg5NDE4NDk5NTM3NTU1NTEwIiwKICAgICAgICAgICAgICAgICIzNjYzOTgxMzk5ODM4NTU5Mzk3NjA4NDA3MTA4MDYzODYyNzQyNjQ3OTgzNjQ0NTUyODA1NDkxMzg1OTAyMjA5NTU3NTMzMDk4MCIKICAgICAgICAgICAgXSwKICAgICAgICAgICAgWwogICAgICAgICAgICAgICAgIjEiLAogICAgICAgICAgICAgICAgIjAiCiAgICAgICAgICAgIF0KICAgICAgICBdLAogICAgICAgICJwaV9jIjogWwogICAgICAgICAgICAiNjM3NjE5NTUzMDE4MDQ1NDM1NzcxODQwMjYzMDcxNTc3OTkyOTc1NzMzMTA5MTM1NTE4MTI4MDk5NTUzNDk5NzMxODQ5Mjg1NTMzMyIsCiAgICAgICAgICAgICIyMDU3NTI3MDEzNDcyMjI4MjY4OTg5MTg4NDMzNzYxOTMzMjE1MzEzMDg1MTI4MTExODE1MzEwMTYxNDY4MjczNDgxNzA2MTA2Nzk0IiwKICAgICAgICAgICAgIjEiCiAgICAgICAgXSwKICAgICAgICAicHJvdG9jb2wiOiAiZ3JvdGgxNiIsCiAgICAgICAgImN1cnZlIjogImJuMTI4IgogICAgfSwKICAgICJwdWJsaWNJbnB1dHMiOiBbCiAgICAgICAgIjIwMTg3MjE0MTQwMzg0MDQ4MjAzMjciLAogICAgICAgICIwIiwKICAgICAgICAiMCIsCiAgICAgICAgIjAiLAogICAgICAgICIwIiwKICAgICAgICAiMCIsCiAgICAgICAgIjAiLAogICAgICAgICIwIiwKICAgICAgICAiMCIsCiAgICAgICAgIjY2MzIzNTM3MTMwODUxNTc5MjU1MDQwMDg0NDMwNzg5MTk3MTYzMjIzODYxNTYxNjA2MDIyMTg1MzY5NjEwMjgwNDY0NjgyMzcxOTIiLAogICAgICAgICI2NDg4NDgxOTU5NDQ5NTMzMDcyMjIzMjY1NTEyOTM1ODI2OTU1MjkzNjEwNzk0NjIzNzE2MDI3MzA2NDQxODA5NTU3ODM4OTQyMTM3IiwKICAgICAgICAiMTc2MTAzNDk1NCIsCiAgICAgICAgIjE4NDM2MTU2NDA2MzA3MDQ1MzI3MzY4NTkyMjEzNjAwMzk2NjMzODY5MjkxNTg0NjQ2OTI2NzAxMzk4ODAxNjU4OTA4Mjc0MDU4MSIsCiAgICAgICAgIjE1NjE2OTA4NjI1MDIyNjIwMDMzMDU0MzM3MDgyMTkxMzQzNzAxOTMxMTU1Njk0MzcyODQyMjkzODQ1MjY5ODY4NjY4NDYxOTM3NyIsCiAgICAgICAgIjQzOTMzMTUyNTAwMjIwNjE2NzUyMDQ4NDMxNzEyNDEwNDUxODg0NjYyMzIwMzM4MjA1MDA2IiwKICAgICAgICAiMCIsCiAgICAgICAgIjAiLAogICAgICAgICIwIiwKICAgICAgICAiMCIsCiAgICAgICAgIjAiLAogICAgICAgICIwIiwKICAgICAgICAiMCIsCiAgICAgICAgIjAiLAogICAgICAgICIwIiwKICAgICAgICAiMCIsCiAgICAgICAgIjAiLAogICAgICAgICIwIiwKICAgICAgICAiMCIsCiAgICAgICAgIjAiLAogICAgICAgICIwIiwKICAgICAgICAiMCIsCiAgICAgICAgIjAiLAogICAgICAgICI4MTA2MzU1MDQzOTY4OTAxNTg3MzQ2NTc5NjM0NTk4MDk4NzY1OTMzMTYwMzk0MDAyMjUxOTQ4MTcwNDIwMjE5OTU4NTIzMjIwNDI1IiwKICAgICAgICAiMSIKICAgIF0KfQ=="
	// fmt.Println(b64signture)

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

	// poseidon([salt, email_address])
	emailCommitment := "17159366307350401517208657413587014704131356894001302493847352957889395820464"
	emailCommitmentBIG, isSet := new(big.Int).SetString(emailCommitment, 10)
	require.True(t, isSet)
	emailHash := base64.StdEncoding.EncodeToString(ToLittleEndian(emailCommitmentBIG.Bytes()))
	t.Logf("email hash: %s", emailHash)
	// tmpSignature := "eyJwcm9vZiI6eyJwaV9hIjpbIjEzMzU5MjM1NDM3OTA1NTEwMTQ2NDg4NTQ1MjY3NTgwODQ3ODY4NzY4NTYzOTYwNzgxNzI5MTk0OTM5NTI3NTIzMjQzNzk1Njg4NzcyIiwiMTYyNTUyMTI0Nzk0NjUwODk2Mzk1MDIwMTM0MzI5MzY1NzI0MTcxMDA3OTQwMjMwMDQ0MDg5MDY3NzAwODA4MzQxNDIxMjMwMDYxMzUiLCIxIl0sInBpX2IiOltbIjE5Mjg0NDEzOTA3MjQ4NTY4ODA5MDc2ODAyOTMxNDcxNjIwNDcxNTMwNzg3MjUyMzkyNDc4MzE1NTY5NDE0MDI4NTM2MTI3NTQwMzMyIiwiMzM5MTM0ODE3NzA0MzIwMDQ1MDQ1MTQ2MTc5MzMzMDA5Mjg4ODA4ODI2ODQ1MjI4MDg3ODg3MDM3ODY1NDc4ODA0ODgxNjQ2MzEwOCJdLFsiMTk4NTI4NTMxMzMyMzY0NjY5NjQ2MzMwMDY5OTg5OTg2MzA4ODIyMDI1OTg3MDExMDgyNzI3NDc5MTQzODAzMzYwMTYzMTA4Nzc3MjUiLCIxMzIwNTY2MDgyMjYyMTc2ODA0OTE3NTc0MjA4NjYzODY1NzY5NTI3NzE4NzcxNzE2OTI4MDk4OTAzNzAxNjgxMzU3MTQ2NTg2MTY5Il0sWyIxIiwiMCJdXSwicGlfYyI6WyIxNTY4MzI2OTMwMjk4NTQ0MzcwODk3MTgyMjUzMjIwOTk1NzY0NTYxODYzMDM5MzMwNjM2OTk4NDk1ODE0ODE2NzI4MzUzOTU4NjgyMSIsIjY0NDI0NzY5MzU3OTIyMjQxNTY1MTE5MDc2NjE1MDA0NzcxMjk1MTM1MjYxNDIxMzk1NTQ5MTUwNDM2ODUzMDE1NzI1Njg0MTYzODAiLCIxIl0sInByb3RvY29sIjoiZ3JvdGgxNiJ9LCJwdWJsaWNJbnB1dHMiOlsiMjAxODcyMTQxNDAzODQwNDgyMDMyNyIsIjAiLCIwIiwiMCIsIjAiLCIwIiwiMCIsIjAiLCIwIiwiNjYzMjM1MzcxMzA4NTE1NzkyNTUwNDAwODQ0MzA3ODkxOTcxNjMyMjM4NjE1NjE2MDYwMjIxODUzNjk2MTAyODA0NjQ2ODIzNzE5MiIsIjE5NTQ0NTE1NDg0Mjk0MTMzMzY1NjIxMTUwODYwNzk4MjQ4OTA4NzgxOTk0NzYwNDMyNTg5Nzg0ODAzODU4NDE4Njk4Nzg5MDUwMDg3IiwiMTc1OTE0NzI5MSIsIjEyNDQxMzU4ODAxMDkzNTU3MzEwMDQ0OTQ1NjQ2ODk1OTgzOTI3MDAyNzc1NzIxNTEzODQzOTgxNjk1NTAyNDczNjI3MTI5ODg4MyIsIjEyNTk4NzcxODUwNDg4MTE2ODcwMjgxNzM3Mjc1MTQwNTUxMTMxMTYyNjUxNTM5OTEyODExNTk1NzY4MzA1NTcwNjE2Mjg3OTA4MSIsIjEzODE3NDI5NDQxOTU2NjA3MzYzODkxNzM5ODQ3ODQ4MDIzMzc4MzQ2MjY1NTQ4MjI4MzQ4OTc3ODQ3NzAzMjEyOTg2MDQxNjMwOCIsIjg3MTY0NDI5OTM1MTgzNTMwMjMxMTA2NTI0MjM4NzcyNDY5MDgzMDIxMzc2NTM2ODU3NTQ3NjAxMjg2MzUwNTExODk1OTU3MDQyIiwiMTU5NTA4OTk1NTU0ODMwMjM1NDIyODgxMjIwMjIxNjU5MjIyODgyNDE2NzAxNTM3Njg0MzY3OTA3MjYyNTQxMDgxMTgxMTA3MDQxIiwiMjE2MTc3ODU5NjMzMDMzOTkzNjE2NjA3NDU2MDEwOTg3ODcwOTgwNzIzMjE0ODMyNjU3MzA0MjUwOTI5MDUyMDU0Mzg3NDUxMjUxIiwiMTM2ODcwMjkzMDc3NzYwMDUxNTM2NTE0Njg5ODE0NTI4MDQwNjUyOTgyMTU4MjY4MjM4OTI0MjExNDQzMTA1MTQzMzE1MzEyOTc3IiwiMjA5MDI3NjQ3MjcxOTQxNTQwNjM0MjYwMTI4MjI3MTM5MTQzMzA1MjEyNjI1NTMwMTMwOTg4Mjg2MzA4NTc3NDUxOTM0NDMzNjA0IiwiMjE2MDQxMDM3NDgwODE2NTAxODQ2MzQ4NzA1MzUzNzM4MDc5Nzc1ODAzNjIzNjA3MzczNjY1Mzc4NDk5ODc2NDc4NzU3NzIxOTU2IiwiMTg0MDk5ODA4ODkyNjA2MDYxOTQyNTU5MTQxMDU5MDgxNTI3MjYyODM0ODU5NjI5MTgxNTgxMjcwNTg1OTA4NTI5MDE0MDAwNDgzIiwiMTczOTI2ODIxMDgyMzA4MDU2ODI5NDQxNzczODYwNDgzODQ5MTI4NDA0OTk2MDg0OTMyOTE5NTA1OTQ2ODAyNDg4MzY3OTg5MDcwIiwiMTM2NDk4MDgzMzMyOTAwMzIxMjE1NTI2MjYwODY4NTYyMDU2NjcwODkyNDEyOTMyNjcxNTE5NTEwOTgxNzA0NDI3OTA1NDMwNTc4IiwiMCIsIjAiLCIwIiwiMCIsIjAiLCIwIiwiMCIsIjAiLCI4MTA2MzU1MDQzOTY4OTAxNTg3MzQ2NTc5NjM0NTk4MDk4NzY1OTMzMTYwMzk0MDAyMjUxOTQ4MTcwNDIwMjE5OTU4NTIzMjIwNDI1IiwiMSJdfQ=="

	// send a execute msg to add a zkemail authenticator to the account
	// TODO: update ZKEmail id, email_salt, signature
	/*
		authExecuteMsg := fmt.Sprintf(
			`{"add_auth_method":{"add_authenticator":{"ZKEmail": {"id": 1, "email_salt": "%s", "signature": "%b"}}}}`,
			emailSalt,
			fileContent,
		)
	*/
	tmpMsg := AuthExecuteMsg{}
	tmpMsg.AddAuthMethod.AddAuthenticator.ZKEmail.ID = 1
	tmpMsg.AddAuthMethod.AddAuthenticator.ZKEmail.EmailSalt = emailSalt
	tmpMsg.AddAuthMethod.AddAuthenticator.ZKEmail.Signature = json.RawMessage(fileContent)

	authExecuteMsg, err := json.Marshal(tmpMsg)
	require.NoError(t, err)
	t.Logf("auth execute msg: %s", authExecuteMsg)

	msgExec := &wasmtypes.MsgExecuteContract{
		Sender:   aaContractAddr, // contract is the sender in this case
		Contract: aaContractAddr, // target contract address is also the AA contract
		Msg:      authExecuteMsg,
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
	require.Equal(t, response["ZKEmail"].(map[string]any)["email_hash"].(string), emailHash, "Email hash should match")
	require.Equal(t, response["ZKEmail"].(map[string]any)["dkim_domain"].(string), dkimDomain, "DKIM domain should match")

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

	type Signature struct {
		Proof    map[string]any `json:"proof"`
		DkimHash string         `json:"dkim_hash"`
	}

	var proof map[string]any
	err = json.Unmarshal(proofBz, &proof)
	require.NoError(t, err)

	sig := &Signature{
		Proof:    proof,
		DkimHash: base64.StdEncoding.EncodeToString(ToLittleEndian(gPubKeyHash.Bytes())),
	}
	sigBz, err := json.Marshal(sig)
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
