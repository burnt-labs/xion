package integration_tests

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"testing"

	signingv1beta1 "cosmossdk.io/api/cosmos/tx/signing/v1beta1"
	"cosmossdk.io/math"
	txsigning "cosmossdk.io/x/tx/signing"
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

func TestZKEmailAuthenticator(t *testing.T) {
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
	var vkey map[string]any
	err = json.Unmarshal(vkeyBz, &vkey)
	require.NoError(t, err)

	vkeyJsonBz, err := json.Marshal(vkey)
	require.NoError(t, err)

	verificationContractAddress, err := xion.InstantiateContract(ctx, xionUser.FormattedAddress(), zkemailCodeID, fmt.Sprintf(`{"vkey": "%s"}`, string(vkeyJsonBz)), true)
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

	// this commitement is gotten from the email address and a salt currently "testuseronexion@gmail.com", "XRhMS5Nc2dTZW5kEpAB"
	emailCommitment := "6293004408188449527623842124792388138908023069134722952437088476138094090885"
	emailHash := base64.StdEncoding.EncodeToString([]byte(emailCommitment))
	dkimDomain := "google.com"
	// send a execute msg to add a zkemail authenticator to the account\
	_, err = xion.ExecuteContract(ctx, accountCodeID, xionUser.FormattedAddress(), fmt.Sprintf(`{"authenticator": {"ZKEmail": {"id": 1, "verification_contract": "%s", "email_hash": "%s", "dkim_domain": "%s"}}}`, verificationContractAddress, emailHash, dkimDomain))
	require.NoError(t, err)

	// Create a bank send message from the AA contract to a recipient
	recipient := "xion1qaf2xflx5j3agtlvqk5vhjpeuhl6g45hxshwqj"
	jsonMsg := RawJSONMsgSend(t, aaContractAddr, recipient, "uxion")
	require.NoError(t, err)

	tx, err := xion.Config().EncodingConfig.TxConfig.TxJSONDecoder()([]byte(jsonMsg))
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

	txBuilder, err := xion.Config().EncodingConfig.TxConfig.WrapTxBuilder(tx)
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

	signatureBz := sha256.Sum256(signBytes)
	signature := base64.StdEncoding.EncodeToString(signatureBz[:])
	t.Logf("signature: %s", signature)

	// Hardcoded proof (pre-generated externally)
	proofBz, err := os.ReadFile(path.Join(fp, "integration_tests", "testdata", "keys", "zkproof.json"))
	if err != nil {
		t.Fatalf("failed to read vkey.json file: %v", err)
	}

	sigData := signing.SingleSignatureData{
		SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
		Signature: proofBz,
	}

	sig := signing.SignatureV2{
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
	newBalance, err := xion.GetBalance(ctx, aaContractAddr, xion.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, int64(1_000_000-100_000), newBalance.Int64())
}
