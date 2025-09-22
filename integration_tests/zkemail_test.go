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
	"time"

	signingv1beta1 "cosmossdk.io/api/cosmos/tx/signing/v1beta1"
	"cosmossdk.io/math"
	txsigning "cosmossdk.io/x/tx/signing"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
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

	dkimDomain := "gmail.com"
	dkimSelector := "20230601"
	dkimPubkey := "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAntvSKT1hkqhKe0xcaZ0x+QbouDsJuBfby/S82jxsoC/SodmfmVs2D1KAH3mi1AqdMdU12h2VfETeOJkgGYq5ljd996AJ7ud2SyOLQmlhaNHH7Lx+Mdab8/zDN1SdxPARDgcM7AsRECHwQ15R20FaKUABGu4NTbR2fDKnYwiq5jQyBkLWP+LgGOgfUF4T4HZb2PY2bQtEP6QeqOtcW4rrsH24L7XhD+HSZb1hsitrE0VPbhJzxDwI4JF815XMnSVjZgYUXP8CxI1Y0FONlqtQYgsorZ9apoW1KPQe8brSSlRsi9sXB/tu56LmG7tEDNmrZ5XUwQYUUADBOu7t1niwXwIDAQAB"
	gPubKeyHash, err := dkimTypes.ComputePoseidonHash(dkimPubkey)
	require.NoError(t, err)
	gPubKeyBz, _ := json.Marshal([]Dkim{
		{
			Domain:       dkimDomain,
			Selector:     dkimSelector,
			PubKey:       dkimPubkey,
			PoseidonHash: base64.StdEncoding.EncodeToString(ToLittleEndian(gPubKeyHash.Bytes())),
		},
	})
	vkey := "eyJ2a19hbHBoYV8xIjpbIjIwNDkxMTkyODA1MzkwNDg1Mjk5MTUzMDA5NzczNTk0NTM0OTQwMTg5MjYxODY2MjI4NDQ3OTE4MDY4NjU4NDcxOTcwNDgxNzYzMDQyIiwiOTM4MzQ4NTM2MzA1MzI5MDIwMDkxODM0NzE1NjE1NzgzNjU2NjU2Mjk2Nzk5NDAzOTcxMjI3MzQ0OTkwMjYyMTI2NjE3ODU0NTk1OCIsIjEiXSwidmtfYmV0YV8yIjpbWyI2Mzc1NjE0MzUxNjg4NzI1MjA2NDAzOTQ4MjYyODY4OTYyNzkzNjI1NzQ0MDQzNzk0MzA1NzE1MjIyMDExNTI4NDU5NjU2NzM4NzMxIiwiNDI1MjgyMjg3ODc1ODMwMDg1OTEyMzg5Nzk4MTQ1MDU5MTM1MzUzMzA3MzQxMzE5Nzc3MTc2ODY1MTQ0MjY2NTc1MjI1OTM5NzEzMiJdLFsiMTA1MDUyNDI2MjYzNzAyNjIyNzc1NTI5MDEwODIwOTQzNTY2OTc0MDk4MzU2ODAyMjA1OTA5NzE4NzMxNzExNDAzNzEzMzEyMDY4NTYiLCIyMTg0NzAzNTEwNTUyODc0NTQwMzI4ODIzMjY5MTE0NzU4NDcyODE5MTE2MjczMjI5OTg2NTMzODM3NzE1OTY5MjM1MDA1OTEzNjY3OSJdLFsiMSIsIjAiXV0sInZrX2dhbW1hXzIiOltbIjEwODU3MDQ2OTk5MDIzMDU3MTM1OTQ0NTcwNzYyMjMyODI5NDgxMzcwNzU2MzU5NTc4NTE4MDg2OTkwNTE5OTkzMjg1NjU1ODUyNzgxIiwiMTE1NTk3MzIwMzI5ODYzODcxMDc5OTEwMDQwMjEzOTIyODU3ODM5MjU4MTI4NjE4MjExOTI1MzA5MTc0MDMxNTE0NTIzOTE4MDU2MzQiXSxbIjg0OTU2NTM5MjMxMjM0MzE0MTc2MDQ5NzMyNDc0ODkyNzI0Mzg0MTgxOTA1ODcyNjM2MDAxNDg3NzAyODA2NDkzMDY5NTgxMDE5MzAiLCI0MDgyMzY3ODc1ODYzNDMzNjgxMzMyMjAzNDAzMTQ1NDM1NTY4MzE2ODUxMzI3NTkzNDAxMjA4MTA1NzQxMDc2MjE0MTIwMDkzNTMxIl0sWyIxIiwiMCJdXSwidmtfZGVsdGFfMiI6W1siNTY4MTAwNjE2NDMwODI1MTk1MzAwMjExMzkyNTU4NTQ5MDE1OTM4MjM2NTQ1OTkwMDYxNjY5NTA2OTczODA5OTYxNjIyMTAwNzQ0MCIsIjE3ODQzNzI1MzIwMjMyODUyMzQ3ODgxMTI5MTI2MDUwMTY0MTg3ODU4NTMwODI2Nzg0MzAxNjI4NDE4MzM2ODg5NzI0MDQ3NTQ1MjYiXSxbIjEzMjA3MzcyNzAwMzc0OTUxMjE3OTcyNzM0MzA1OTg3OTg0MjExMjk5NTQxOTUzMjcyMTk1ODA0NDAzMDE5Mjk0Mjg1MDE1NDMyMjMwIiwiMTEyMDgwOTg4MTU5MzE0OTgyNTg1NjYzMzU5NTMyNTU3Njg5MTU3ODEwMjQ1Njg0MzE0NzI5NzMwMDM3NDAzNTEyNDIxNDUyNzQ4ODIiXSxbIjEiLCIwIl1dLCJJQyI6W1siMTM1NTI3OTYxNTkzMjE1MDA0NDY1NDIwOTIzMzM5MjU3ODk2MjcyMjE3NjM5OTU5MTcwNTc3MDUwMjUwMjQ3MzY4NDU3MjE5NDIzNTMiLCI5NjgwOTg1Mjg1MDQwMjMwNzUxNjQ0NDk1ODkzMjE1MzgyNjc2MDgxNTM1ODc2MzYzNTUzNjYwMzc0MTE1NTg4ODYyMDYwNDg3MDM0IiwiMSJdLFsiMTUyNTQ3NzY3NzI2MTA1MzMzMjc0ODg4MTQyODE5MzEyODc2NjU4OTM5NDMxNzc5NzczMTAxNjgwNTY3MjIzNjgyODE3Mjg2MTg2NjIiLCIxNDM2NzY1NzcwMTI0OTU0NzkxODUxMzY3NTUzMzA5MTI4NjM4MzM1ODE4NTY2Mjg2OTg4NzMxNjQ2NjgxMzA2MjE0MjY3NDQ5OTI2MCIsIjEiXSxbIjE0ODY2ODU5MTc3NzU4NjM1MDMwMDc5MjI2MzQxODYyNjAxMTEyOTgzNDg3ODM4NTExNjkwNTY3ODk1NTc0MDMwNjcyMjEyNDQxNjc2IiwiMTEzMTQ1NDIyOTM1MzM5NzMzMjg0MTY0NDE2NjMxNzYwMzIzOTk4NDU1MTkzMDA0NDY4OTQ0MDc1MjAwODM0MTIwMzY4ODgzNDI5OCIsIjEiXSxbIjE0NDUyOTAyNDgyODI4NTU4MjcwMjk4Mzc0NzM3NzA1NjU4MzUxMjcwNzg4NDI4MDE5NTg1MjQxMDI2OTAxMzMxMzY4ODU4NTI3OTQ4IiwiMTkwOTc5OTc3NzQ2MDY1MjM1MjcxNTg1Mjk1NTI2NDU5NjEyNTcxNTAxODkzNTczNTc2MDEwMzUwNzgxMTE2NDM5NTMzODUxNzEzODQiLCIxIl1dfQ=="

	td := BuildXionChain(t, "0.0uxion", ModifyInterChainGenesis(ModifyInterChainGenesisFn{ModifyGenesisShortProposals, ModifyGenesisDKIMRecords, ModifyGenesisAAParams}, [][]string{{votingPeriod, maxDepositPeriod}, {string(gPubKeyBz), vkey}, {"20000000"}}))
	xion, ctx := td.xionChain, td.ctx

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

	// Store Abstract Account Contract
	fp, err := os.Getwd()
	require.NoError(t, err)

	accountCodeID, err := xion.StoreContract(ctx, xionUser.FormattedAddress(), path.Join(fp, "integration_tests", "testdata", "contracts", "xion-account.wasm"))
	require.NoError(t, err)

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

	// send a execute msg to add a zkemail authenticator to the account
	authExecuteMsg := fmt.Sprintf(
		`{"add_auth_method":{"add_authenticator":{"ZKEmail": {"id": 1, "email_hash": "%s", "dkim_domain": "%s"}}}}`,
		emailHash,
		dkimDomain,
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
	txBuilder.SetGasLimit(1_900_000) // 20 million because verification takes a lot of gas

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

	fmt.Println("waiting")
	err = testutil.WaitForBlocks(ctx, 2, xion)
	require.NoError(t, err)
	recipientBalance, err := xion.GetBalance(ctx, recipient, xion.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, int64(100_000), recipientBalance.Int64())
	require.NoError(t, err)
}
