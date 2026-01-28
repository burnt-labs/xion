package e2e_zk

import (
	"context"
	"crypto/sha256"
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
	"github.com/burnt-labs/xion/e2e_tests/testlib"
	dkimTypes "github.com/burnt-labs/xion/x/dkim/types"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	govModule "github.com/cosmos/cosmos-sdk/x/gov/types"
	ibctest "github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/cosmos/interchaintest/v10/testutil"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestZKAddAuthenticators(t *testing.T) {
	ctx := t.Context()
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("xion", "xionpub")

	xion := testlib.BuildXionChain(t)
	proposalTracker := testlib.NewProposalTracker(1)

	fundAmount := math.NewInt(1_000_000_000_000)
	xionUser, err := ibctest.GetAndFundTestUserWithMnemonic(ctx, "zk-aa-auth", testlib.DeployerMnemonic, fundAmount, xion)
	require.NoError(t, err)

	currentHeight, _ := xion.Height(ctx)
	err = testutil.WaitForBlocks(ctx, int(currentHeight)+8, xion)
	require.NoError(t, err)

	seedDKIMRecords(t, ctx, xion, xionUser, proposalTracker)

	accountCodeID, err := xion.StoreContract(ctx, xionUser.FormattedAddress(), testlib.IntegrationTestPath("testdata", "contracts", "xion_account.wasm"))
	require.NoError(t, err)

	registeredTxHash, err := testlib.ExecTx(t, ctx, xion.GetNode(),
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

	txDetails, err := testlib.ExecQuery(t, ctx, xion.GetNode(), "tx", registeredTxHash)
	require.NoError(t, err)
	aaContractAddr := testlib.GetAAContractAddress(t, txDetails)

	addZKEmailAuthenticator(t, ctx, xion, xionUser.KeyName(), aaContractAddr)
	assertAuthenticatorPresent(t, ctx, xion, aaContractAddr, 1, "ZKEmail")

	zkAuthIndex := uint8(1)

	zkSignature := loadZKSignature(t, "Secp256K1")
	secpKey := secp256k1.GenPrivKeyFromSecret(make([]byte, 32))
	secpSig, err := secpKey.Sign([]byte(aaContractAddr))
	require.NoError(t, err)
	secpAuth := map[string]any{
		"id":        uint8(3),
		"pubkey":    secpKey.PubKey().Bytes(),
		"signature": secpSig,
	}
	addAuthenticatorWithZK(t, ctx, xion, aaContractAddr, zkAuthIndex, zkSignature, buildAddAuthMsg("Secp256K1", secpAuth))
	err = testutil.WaitForBlocks(ctx, 2, xion)
	require.NoError(t, err)
	assertAuthenticatorPresent(t, ctx, xion, aaContractAddr, 3, "Secp256K1")

	zkSignature = loadZKSignature(t, "Ed25519")
	edKey := ed25519.GenPrivKeyFromSecret(make([]byte, 32))
	// sha 256 hash of aaContractAddr
	aaContractAddrHash := sha256.Sum256([]byte(aaContractAddr))
	edSig, err := edKey.Sign(aaContractAddrHash[:])
	require.NoError(t, err)
	edAuth := map[string]any{
		"id":        uint8(4),
		"pubkey":    edKey.PubKey().Bytes(),
		"signature": edSig,
	}
	addAuthenticatorWithZK(t, ctx, xion, aaContractAddr, zkAuthIndex, zkSignature, buildAddAuthMsg("Ed25519", edAuth))
	err = testutil.WaitForBlocks(ctx, 2, xion)
	require.NoError(t, err)
	assertAuthenticatorPresent(t, ctx, xion, aaContractAddr, 4, "Ed25519")
}

func seedDKIMRecords(t *testing.T, ctx context.Context, xion *cosmos.CosmosChain, proposer ibc.Wallet, proposalTracker *testlib.ProposalTracker) {
	govModAddress := testlib.GetModuleAddress(t, xion, ctx, govModule.ModuleName)

	gmailPoseidonHashStr := "6632353713085157925504008443078919716322386156160602218536961028046468237192"
	gmailPoseidonHashInt := new(big.Int)
	gmailPoseidonHashInt.SetString(gmailPoseidonHashStr, 10)
	icloudPoseidonHashStr := "20739234269106695800684585604154242365519276934929568271517631027216377681264"
	icloudPoseidonHashInt := new(big.Int)
	icloudPoseidonHashInt.SetString(icloudPoseidonHashStr, 10)

	gmailDkimPubKey := "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv3bzh5rabT+IWegVAoGnS/kRO2kbgr+jls+Gm5S/bsYYCS/MFsWBuegRE8yHwfiyT5Q90KzwZGkeGL609yrgZKJDHv4TM2kmybi4Kr/CsnhjVojMM7iZVu2Ncx/i/PaCEJzo94dcd4nIS+GXrFnRxU/vIilLojJ01W+jwuxrrkNg8zx6a9wWRwdQUYGUIbGkYazPdYUd/8M8rviLwT9qsnJcM4b3Ie/gtcYzsL5LhuvhfbhRVNGXEMADasx++xxfbIpPr5AgpnZo+6rA1UCUfwZT83Q2pAybaOcpjGUEWpP8h30Gi5xiUBR8rLjweG3MtYlnqTHSyiHGUt9JSCXGPQIDAQAB"
	icloudDkimPubKey := "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA1ZEfbkf4TbO2TDZI67WhJ6G8Dwk3SJyAbBlE/QKdyXFZB4HfEU7AcuZBzcXSJFE03DlmyOkUAmaaR8yFlwooHyaKRLIaT3epGlL5YGowyfItLly2k0Jj0IOICRxWrB378b7qMeimE8KlH1UNaVpRTTi0XIYjIKAOpTlBmkM9a/3Rl4NWy8pLYApXD+WCkYxPcxoAAgaN8osqGTCJ5r+VHFU7Wm9xqq3MZmnfo0bzInF4UajCKjJAQa+HNuh95DWIYP/wV77/PxkEakOtzkbJMlFJiK/hMJ+HQUvTbtKW2s+t4uDK8DI16Rotsn6e0hS8xuXPmVte9ZzplD0fQgm2qwIDAQAB"

	dkimRecords := []dkimTypes.DkimPubKey{
		{
			Domain:       "gmail.com",
			Selector:     "selector1",
			PubKey:       gmailDkimPubKey,
			PoseidonHash: gmailPoseidonHashInt.Bytes(),
		},
		{
			Domain:       "icloud.com",
			Selector:     "1a1hai",
			PubKey:       icloudDkimPubKey,
			PoseidonHash: icloudPoseidonHashInt.Bytes(),
		},
	}

	createDkimMsg := &dkimTypes.MsgAddDkimPubKeys{
		Authority:   govModAddress,
		DkimPubkeys: dkimRecords,
	}
	require.NoError(t, createDkimMsg.ValidateBasic())

	dkimProposalID := proposalTracker.NextID()
	err := testlib.SubmitAndPassProposal(t, ctx, xion, proposer,
		[]cosmos.ProtoMessage{createDkimMsg},
		"Seed DKIM for ZKEmail", "Seed DKIM record for ZKEmail authentication", "Seed DKIM for ZKEmail", dkimProposalID)
	require.NoError(t, err)
}

func addZKEmailAuthenticator(t *testing.T, ctx context.Context, xion *cosmos.CosmosChain, keyName, aaContractAddr string) {
	zkAuthContent, err := os.ReadFile(testlib.IntegrationTestPath("testdata", "keys", "zk-auth.json"))
	require.NoError(t, err)

	var zkAuthData map[string]interface{}
	err = json.Unmarshal(zkAuthContent, &zkAuthData)
	require.NoError(t, err)

	publicInputs, ok := zkAuthData["publicInputs"].([]interface{})
	require.True(t, ok, "publicInputs should be an array")
	emailSalt, ok := publicInputs[68].(string)
	require.True(t, ok, "emailSalt should be a string")

	zkAuthJSONBytes, err := json.Marshal(zkAuthData)
	require.NoError(t, err)
	b64signature := base64.StdEncoding.EncodeToString(zkAuthJSONBytes)

	allowedEmailHosts := []string{"kushal@burnt.com", "zk@zk.burnt.com", "jane@burnt.com"}
	allowedEmailHostsJSON, err := json.Marshal(allowedEmailHosts)
	require.NoError(t, err)

	authExecuteMsg := fmt.Sprintf(
		`{"add_auth_method":{"add_authenticator":{"ZKEmail": {"id": 1, "email_salt": "%s", "allowed_email_hosts": %s, "signature": "%s"}}}}`,
		emailSalt,
		string(allowedEmailHostsJSON),
		b64signature,
	)

	msgExec := &wasmtypes.MsgExecuteContract{
		Sender:   aaContractAddr,
		Contract: aaContractAddr,
		Msg:      []byte(authExecuteMsg),
		Funds:    sdk.Coins{},
	}

	txBuilder := xion.Config().EncodingConfig.TxConfig.NewTxBuilder()
	err = txBuilder.SetMsgs(msgExec)
	require.NoError(t, err)

	txBuilder.SetFeeAmount(sdk.Coins{{Denom: xion.Config().Denom, Amount: math.NewInt(100000)}})
	txBuilder.SetGasLimit(500000)

	unsignedTxBz, err := xion.Config().EncodingConfig.TxConfig.TxJSONEncoder()(txBuilder.GetTx())
	require.NoError(t, err)

	file, err := os.CreateTemp("", "*-auth.json")
	require.NoError(t, err)
	defer os.Remove(file.Name())
	_, err = file.Write(unsignedTxBz)
	require.NoError(t, err)
	err = testlib.UploadFileToContainer(t, ctx, xion.GetNode(), file)
	require.NoError(t, err)

	configFilePath := strings.Split(file.Name(), "/")
	cmd := []string{
		"xion", "sign", keyName, aaContractAddr, path.Join(xion.GetNode().HomeDir(), configFilePath[len(configFilePath)-1]),
		"--chain-id", xion.Config().ChainID,
		"--gas-prices", "1uxion", "--gas-adjustment", "1.4",
		"--gas", "400000",
		"-y",
	}

	_, err = testlib.ExecTx(t, ctx, xion.GetNode(), keyName, cmd...)
	require.NoError(t, err)
	err = testutil.WaitForBlocks(ctx, 2, xion)
	require.NoError(t, err)
}

func loadZKSignature(t *testing.T, auth string) testlib.ZKEmailSignature {
	proofPath := testlib.IntegrationTestPath("testdata", "keys", "all_auth_proof.json")
	if _, err := os.Stat(proofPath); err != nil {
		t.Fatalf("all authenticator proof file does not exist: %s", proofPath)
	}

	proofContent, err := os.ReadFile(proofPath)
	require.NoError(t, err)

	type ZKAuthProof map[string]testlib.ZKEmailSignature
	var zkSignature ZKAuthProof
	err = json.Unmarshal(proofContent, &zkSignature)
	require.NoError(t, err)

	return zkSignature[auth]
}

func buildAddAuthMsg(authType string, details map[string]any) []byte {
	addAuthenticator := map[string]any{authType: details}
	addAuthMethod := map[string]any{"add_authenticator": addAuthenticator}
	msg := map[string]any{"add_auth_method": addAuthMethod}
	msgBz, _ := json.Marshal(msg)
	return msgBz
}

func addAuthenticatorWithZK(t *testing.T, ctx context.Context, xion *cosmos.CosmosChain, aaContractAddr string, authIndex uint8, zkSignature testlib.ZKEmailSignature, executeMsg []byte) {
	msgExec := &wasmtypes.MsgExecuteContract{
		Sender:   aaContractAddr,
		Contract: aaContractAddr,
		Msg:      executeMsg,
		Funds:    sdk.Coins{},
	}

	account := fetchAAAccount(t, ctx, xion, aaContractAddr)

	txBuilder := xion.Config().EncodingConfig.TxConfig.NewTxBuilder()
	err := txBuilder.SetMsgs(msgExec)
	require.NoError(t, err)

	txBuilder.SetFeeAmount(sdk.Coins{{Denom: xion.Config().Denom, Amount: math.NewInt(100000)}})
	txBuilder.SetGasLimit(800000)

	pubKey := account.GetPubKey()
	anyPk, err := types.NewAnyWithValue(pubKey)
	require.NoError(t, err)

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

	sigData := signing.SingleSignatureData{
		SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
		Signature: nil,
	}

	sig := signing.SignatureV2{
		PubKey:   pubKey,
		Data:     &sigData,
		Sequence: account.GetSequence(),
	}

	err = txBuilder.SetSignatures(sig)
	require.NoError(t, err)

	adaptableTx, ok := txBuilder.GetTx().(authsigning.V2AdaptableTx)
	require.True(t, ok, "expected tx to implement V2AdaptableTx")
	txData := adaptableTx.GetSigningTxData()

	sb, err := xion.Config().EncodingConfig.TxConfig.SignModeHandler().GetSignBytes(
		ctx,
		signingv1beta1.SignMode(signing.SignMode_SIGN_MODE_DIRECT),
		signerData,
		txData,
	)
	require.NoError(t, err)
	t.Logf("Sign bytes: %s", base64.StdEncoding.EncodeToString(sb))

	sigBz, err := json.Marshal(zkSignature)
	require.NoError(t, err)
	sigBytes := append([]byte{authIndex}, sigBz...)

	sigData = signing.SingleSignatureData{
		SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
		Signature: sigBytes,
	}

	sig = signing.SignatureV2{
		PubKey:   pubKey,
		Data:     &sigData,
		Sequence: account.GetSequence(),
	}

	err = txBuilder.SetSignatures(sig)
	require.NoError(t, err)

	jsonTx, err := xion.Config().EncodingConfig.TxConfig.TxJSONEncoder()(txBuilder.GetTx())
	require.NoError(t, err)

	txHash, err := testlib.ExecBroadcastWithFlags(t, ctx, xion.GetNode(), jsonTx, "--output", "json")
	require.NoError(t, err)
	// query the tx by hash
	_, err = testlib.ExecQuery(t, ctx, xion.GetNode(), "tx", txHash)
	require.NoError(t, err)
}

func fetchAAAccount(t *testing.T, ctx context.Context, xion *cosmos.CosmosChain, aaContractAddr string) aatypes.AbstractAccount {
	accountResponse, err := testlib.ExecQuery(t, ctx, xion.GetNode(), "auth", "account", aaContractAddr)
	require.NoError(t, err)

	ac, ok := accountResponse["account"]
	require.True(t, ok)
	acMap, ok := ac.(map[string]interface{})
	require.True(t, ok)
	acData, ok := acMap["value"]
	require.True(t, ok)

	accountJSON, err := json.Marshal(acData)
	require.NoError(t, err)

	var account aatypes.AbstractAccount
	err = xion.Config().EncodingConfig.Codec.UnmarshalJSON(accountJSON, &account)
	require.NoError(t, err)

	return account
}

func assertAuthenticatorPresent(t *testing.T, ctx context.Context, xion *cosmos.CosmosChain, aaContractAddr string, id uint8, expectedKey string) {
	// query the authenticator by ID
	queryMsg := fmt.Sprintf(`{"authenticator_by_i_d":{"id":%d}}`, id)
	resp, err := testlib.ExecQuery(t, ctx, xion.GetNode(), "wasm", "contract-state", "smart", aaContractAddr, queryMsg)
	require.NoError(t, err)

	data64, ok := resp["data"].(string)
	require.True(t, ok)
	decoded, err := base64.StdEncoding.DecodeString(data64)
	require.NoError(t, err)

	var authResp map[string]any
	err = json.Unmarshal(decoded, &authResp)
	require.NoError(t, err)
	require.Contains(t, authResp, expectedKey)
}
