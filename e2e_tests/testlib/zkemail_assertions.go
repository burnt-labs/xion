package testlib

import (
	"context"
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
	govModule "github.com/cosmos/cosmos-sdk/x/gov/types"
	ibctest "github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/cosmos/interchaintest/v10/testutil"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/anypb"
)

// ZKEmailProofData represents the structure of a ZK proof.
type ZKEmailProofData struct {
	PiA      []string   `json:"pi_a"`
	PiB      [][]string `json:"pi_b"`
	PiC      []string   `json:"pi_c"`
	Protocol string     `json:"protocol"`
	Curve    string     `json:"curve"`
}

// ZKEmailSignature represents the signature structure for ZKEmail.
type ZKEmailSignature struct {
	Proof        ZKEmailProofData `json:"proof"`
	PublicInputs []string         `json:"publicInputs"`
}

// ZKEmailContractQuery is used for querying the AA contract.
type ZKEmailContractQuery struct {
	AuthenticatorById map[string]interface{} `json:"authenticator_by_i_d"`
}

// ZKEmailAssertionConfig contains configuration for running ZKEmail assertions.
type ZKEmailAssertionConfig struct {
	Chain           *cosmos.CosmosChain
	Ctx             context.Context
	User            ibc.Wallet       // Optional: if nil, will create and fund a new user with DeployerMnemonic
	ProposalTracker *ProposalTracker // Required: tracks proposal IDs for DKIM seeding
}

// ZKEmailTestDataPaths contains paths to test data files.
type ZKEmailTestDataPaths struct {
	ZKAuthJSON        string // Path to zk-auth.json
	ZKTransactionJSON string // Path to zk-transaction.json
	AccountWasm       string // Path to xion_account.wasm
}

// DefaultZKEmailTestDataPaths returns the standard paths for ZKEmail test data.
func DefaultZKEmailTestDataPaths() ZKEmailTestDataPaths {
	return ZKEmailTestDataPaths{
		ZKAuthJSON:        IntegrationTestPath("testdata", "keys", "zk-auth.json"),
		ZKTransactionJSON: IntegrationTestPath("testdata", "keys", "zk-transaction.json"),
		AccountWasm:       IntegrationTestPath("testdata", "contracts", "xion_account.wasm"),
	}
}

// RunZKEmailAuthenticatorAssertions tests the full ZKEmail authenticator flow.
// This is extracted from TestZKEmailAuthenticator in zkemail_test.go.
func RunZKEmailAuthenticatorAssertions(t *testing.T, cfg ZKEmailAssertionConfig) {
	t.Log("Running ZKEmail authenticator assertions")

	ctx := cfg.Ctx
	xion := cfg.Chain
	testDataPaths := DefaultZKEmailTestDataPaths()

	config := types.GetConfig()
	config.SetBech32PrefixForAccount("xion", "xionpub")

	// Get or create user
	var xionUser ibc.Wallet
	var err error
	if cfg.User != nil {
		xionUser = cfg.User
	} else {
		fundAmount := math.NewInt(1_000_000_000_000)
		xionUser, err = ibctest.GetAndFundTestUserWithMnemonic(ctx, "zkemail-test", DeployerMnemonic, fundAmount, xion)
		require.NoError(t, err)
	}

	currentHeight, _ := xion.Height(ctx)
	err = testutil.WaitForBlocks(ctx, int(currentHeight)+8, xion)
	require.NoError(t, err)
	t.Logf("Using xion user %s for ZKEmail test", xionUser.FormattedAddress())

	// Fund a temp user (needed for some internal setup)
	_ = ibctest.GetAndFundTestUsers(t, ctx, "tmp", math.NewInt(1_000_000_000_000), xion)

	// Seed DKIM record required for ZKEmail authentication
	// The ZKEmail authenticator verifies against DKIM records in the dkim module
	t.Log("Seeding DKIM record for ZKEmail authentication...")
	govModAddress := GetModuleAddress(t, xion, ctx, govModule.ModuleName)

	// The poseidon hash from publicInputs[9] in the ZK proof
	gmailPoseidonHashStr := "6632353713085157925504008443078919716322386156160602218536961028046468237192"
	gmailPoseidonHashInt := new(big.Int)
	gmailPoseidonHashInt.SetString(gmailPoseidonHashStr, 10)
	icloudPoseidonHashStr := "20739234269106695800684585604154242365519276934929568271517631027216377681264"
	icloudPoseidonHashInt := new(big.Int)
	icloudPoseidonHashInt.SetString(icloudPoseidonHashStr, 10)

	// Standard test DKIM public key (same as used in DKIM tests)
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

	dkimProposalID := cfg.ProposalTracker.NextID()
	err = SubmitAndPassProposal(t, ctx, xion, xionUser,
		[]cosmos.ProtoMessage{createDkimMsg},
		"Seed DKIM for ZKEmail", "Seed DKIM record for ZKEmail authentication", "Seed DKIM for ZKEmail",
		dkimProposalID)
	require.NoError(t, err)
	t.Log("DKIM record seeded successfully")

	// Verify DKIM record was added correctly
	gmailDkimRecord, err := ExecQuery(t, ctx, xion.GetNode(), "dkim", "dkim-pubkey", "gmail.com", "selector1")
	require.NoError(t, err, "DKIM record query failed")
	gmailDkimPubKeyResult, ok := gmailDkimRecord["dkim_pub_key"].(map[string]interface{})
	require.True(t, ok, "dkim_pub_key should be a map")
	require.Equal(t, gmailDkimPubKey, gmailDkimPubKeyResult["pub_key"].(string), "DKIM pubkey should match")
	require.Equal(t, "gmail.com", gmailDkimPubKeyResult["domain"].(string), "DKIM domain should be gmail.com")
	require.Equal(t, "selector1", gmailDkimPubKeyResult["selector"].(string), "DKIM selector should be selector1")

	icloudDkimRecord, err := ExecQuery(t, ctx, xion.GetNode(), "dkim", "dkim-pubkey", "icloud.com", "1a1hai")
	require.NoError(t, err, "DKIM record query failed")
	icloudDkimPubKeyResult, ok := icloudDkimRecord["dkim_pub_key"].(map[string]interface{})
	require.True(t, ok, "dkim_pub_key should be a map")
	require.Equal(t, icloudDkimPubKey, icloudDkimPubKeyResult["pub_key"].(string), "DKIM pubkey should match")
	require.Equal(t, "icloud.com", icloudDkimPubKeyResult["domain"].(string), "DKIM domain should be icloud.com")
	require.Equal(t, "1a1hai", icloudDkimPubKeyResult["selector"].(string), "DKIM selector should be 1a1hai")

	t.Log("DKIM record verified successfully")

	// Store Abstract Account Contract
	accountCodeID, err := xion.StoreContract(ctx, xionUser.FormattedAddress(), testDataPaths.AccountWasm)
	require.NoError(t, err)
	t.Logf("Stored AA contract with code ID: %s", accountCodeID)

	// Read zk-auth.json and generate base64 signature
	zkAuthContent, err := os.ReadFile(testDataPaths.ZKAuthJSON)
	require.NoError(t, err)

	var zkAuthData map[string]interface{}
	err = json.Unmarshal(zkAuthContent, &zkAuthData)
	require.NoError(t, err)

	// Extract emailSalt from publicInputs
	publicInputs, ok := zkAuthData["publicInputs"].([]interface{})
	require.True(t, ok, "publicInputs should be an array")
	emailSalt, ok := publicInputs[32].(string)
	require.True(t, ok, "emailSalt should be a string")

	zkAuthJSONBytes, err := json.Marshal(zkAuthData)
	require.NoError(t, err)
	b64signature := base64.StdEncoding.EncodeToString(zkAuthJSONBytes)

	// Create allowed email hosts
	allowedEmailHosts := []string{"kushal@burnt.com", "jose@burnt.com", "jane@burnt.com"}
	allowedEmailHostsJSON, err := json.Marshal(allowedEmailHosts)
	require.NoError(t, err)

	// Register Abstract Account Contract
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
	aaContractAddr := GetAAContractAddress(t, txDetails)
	t.Logf("Registered AA contract at: %s", aaContractAddr)

	// Add ZKEmail authenticator to the account
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
		Funds:    types.Coins{},
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
	defer os.Remove(file.Name())
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
	err = testutil.WaitForBlocks(ctx, 2, xion)
	require.NoError(t, err)

	// Verify authenticator was added
	txDetails, err = ExecQuery(t, ctx, xion.GetNode(), "tx", txHash)
	require.NoError(t, err)

	queryMsg := ZKEmailContractQuery{
		AuthenticatorById: map[string]interface{}{"id": 1},
	}

	var queryResult map[string]interface{}
	err = xion.QueryContract(ctx, aaContractAddr, queryMsg, &queryResult)
	require.NoError(t, err)

	var response map[string]interface{}
	data := queryResult["data"].(string)
	decodedData, err := base64.StdEncoding.DecodeString(data)
	require.NoError(t, err)
	err = json.Unmarshal(decodedData, &response)
	require.NoError(t, err)
	require.Contains(t, response, "ZKEmail", "Response should contain ZKEmail field")

	t.Log("ZKEmail authenticator added successfully, testing transaction signing...")

	err = testutil.WaitForBlocks(ctx, 2, xion)
	require.NoError(t, err)

	// Create and broadcast a transaction using the ZK proof
	recipient := "xion1qaf2xflx5j3agtlvqk5vhjpeuhl6g45hxshwqj"
	jsonMsg := RawJSONMsgSend(t, aaContractAddr, recipient, "uxion")

	tx, err := xion.Config().EncodingConfig.TxConfig.TxJSONDecoder()([]byte(jsonMsg))
	require.NoError(t, err)

	// Get the account from the chain
	accountResponse, err := ExecQuery(t, ctx, xion.GetNode(), "auth", "account", aaContractAddr)
	require.NoError(t, err)

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

	// Create signer data
	pubKey := account.GetPubKey()
	anyPk, err := codectypes.NewAnyWithValue(pubKey)
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

	txBuilder, err = xion.Config().EncodingConfig.TxConfig.WrapTxBuilder(tx)
	require.NoError(t, err)

	// Load pre-generated proof
	zkTransactionContent, err := os.ReadFile(testDataPaths.ZKTransactionJSON)
	require.NoError(t, err, "failed to read zk-transaction.json file")

	var zkTransaction ZKEmailSignature
	err = json.Unmarshal(zkTransactionContent, &zkTransaction)
	require.NoError(t, err)

	sigBz, err := json.Marshal(zkTransaction)
	require.NoError(t, err)

	// Prepend auth index to signature
	zkTransactionBz := append([]byte{uint8(1)}, sigBz...)

	sigData := signing.SingleSignatureData{
		SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
		Signature: zkTransactionBz,
	}

	sigV2 := signing.SignatureV2{
		PubKey:   account.GetPubKey(),
		Data:     &sigData,
		Sequence: account.GetSequence(),
	}

	err = txBuilder.SetSignatures(sigV2)
	require.NoError(t, err)

	txBuilder.SetFeeAmount(types.Coins{{Denom: xion.Config().Denom, Amount: math.NewInt(60_000)}})
	txBuilder.SetGasLimit(2_000_000)

	builtTx := txBuilder.GetTx()
	adaptableTx, ok := builtTx.(authsigning.V2AdaptableTx)
	require.True(t, ok, "expected tx to implement V2AdaptableTx")
	txData := adaptableTx.GetSigningTxData()

	signBytes, err := xion.Config().EncodingConfig.TxConfig.SignModeHandler().GetSignBytes(
		ctx,
		signingv1beta1.SignMode(signing.SignMode_SIGN_MODE_DIRECT),
		signerData, txData)
	require.NoError(t, err)
	_ = signBytes // signBytes used for signature verification

	jsonTx, err := xion.Config().EncodingConfig.TxConfig.TxJSONEncoder()(txBuilder.GetTx())
	require.NoError(t, err)

	// Broadcast transaction and verify it succeeded
	txHash, err = ExecBroadcastWithFlags(t, ctx, xion.GetNode(), jsonTx, "--output", "json")
	require.NoError(t, err, "ZK email transaction broadcast failed")
	require.NotEmpty(t, txHash, "Expected non-empty transaction hash")

	err = testutil.WaitForBlocks(ctx, 5, xion)
	require.NoError(t, err)

	// Verify recipient received funds
	recipientBalance, err := xion.GetBalance(ctx, recipient, xion.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, int64(100_000), recipientBalance.Int64())

	t.Log("ZKEmail authenticator assertions completed successfully")
}
