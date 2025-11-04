package e2e_jwk

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/burnt-labs/xion/e2e_tests/testlib"

	signingv1beta1 "cosmossdk.io/api/cosmos/tx/signing/v1beta1"
	"cosmossdk.io/math"
	txsigning "cosmossdk.io/x/tx/signing"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	aatypes "github.com/burnt-labs/abstract-account/x/abstractaccount/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	ibctest "github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/cosmos/interchaintest/v10/testutil"
	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/anypb"
)

// TestJWKInvalidSignature tests that JWT tokens with incorrect signatures are rejected
// This is a Priority 1 security test preventing authentication bypass
func TestJWKInvalidSignature(t *testing.T) {
	ctx := t.Context()
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	t.Parallel()

	spec := testlib.XionLocalChainSpec(t, 3, 1)
	xion := testlib.BuildXionChainWithSpec(t, spec)

	config := types.GetConfig()
	config.SetBech32PrefixForAccount("xion", "xionpub")

	// Create and fund user
	fundAmount := math.NewInt(10_000_000)
	users := ibctest.GetAndFundTestUsers(t, ctx, "default", fundAmount, xion)
	xionUser := users[0]
	err := testutil.WaitForBlocks(ctx, 8, xion)
	require.NoError(t, err)
	t.Logf("created xion user %s", xionUser.FormattedAddress())

	// Load CORRECT private key (key A)
	privateKeyABz, err := os.ReadFile(testlib.IntegrationTestPath("testdata", "keys", "jwtRS256.key"))
	require.NoError(t, err)
	privateKeyA, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyABz)
	require.NoError(t, err)

	// Build JWK for key A
	testKeyA, err := jwk.ParseKey(privateKeyABz, jwk.WithPEM(true))
	require.NoError(t, err)
	err = testKeyA.Set("alg", "RS256")
	require.NoError(t, err)
	testKeyAPublic, err := testKeyA.PublicKey()
	require.NoError(t, err)
	testPublicKeyAJSON, err := json.Marshal(testKeyAPublic)
	require.NoError(t, err)

	// Register audience with public key A
	aud := "invalid-sig-test"
	sub := "test-user-invalid-sig"

	_, err = testlib.ExecTx(t, ctx, xion.GetNode(),
		xionUser.KeyName(),
		"jwk", "create-audience-claim",
		aud,
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err)

	_, err = testlib.ExecTx(t, ctx, xion.GetNode(),
		xionUser.KeyName(),
		"jwk", "create-audience",
		aud,
		string(testPublicKeyAJSON),
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err)
	t.Logf("✓ Registered audience with key A")

	// Deploy AA contract from abstract-account module
	codeIDStr, err := xion.StoreContract(ctx, xionUser.FormattedAddress(),
		testlib.IntegrationTestPath("testdata", "contracts", "account_updatable-aarch64.wasm"))
	require.NoError(t, err)

	codeResp, err := testlib.ExecQuery(t, ctx, xion.GetNode(), "wasm", "code-info", codeIDStr)
	require.NoError(t, err)
	codeHash, err := hex.DecodeString(codeResp["checksum"].(string))
	require.NoError(t, err)

	salt := "0"
	creatorAddr := types.AccAddress(xionUser.Address())
	predictedAddr := wasmkeeper.BuildContractAddressPredictable(codeHash, creatorAddr, []byte(salt), []byte{})
	t.Logf("predicted address: %s", predictedAddr.String())

	// Create authenticator config
	authenticatorDetails := map[string]interface{}{
		"sub": sub,
		"aud": aud,
		"id":  0,
	}
	authenticator := map[string]interface{}{
		"Jwt": authenticatorDetails,
	}
	instantiateMsg := map[string]interface{}{
		"authenticator": authenticator,
	}

	// Create valid JWT token WITH CORRECT SIGNATURE for registration
	now := time.Now()
	fiveAgo := now.Add(-time.Second * 5)
	inFive := now.Add(time.Minute * 5)
	signature := base64.StdEncoding.EncodeToString([]byte(predictedAddr.String()))
	auds := jwt.ClaimStrings{aud}

	validToken := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss":              aud,
		"sub":              sub,
		"aud":              auds,
		"exp":              inFive.Unix(),
		"nbf":              fiveAgo.Unix(),
		"iat":              fiveAgo.Unix(),
		"transaction_hash": signature,
	})

	signedValidToken, err := validToken.SignedString(privateKeyA)
	require.NoError(t, err)

	authenticatorDetails["token"] = []byte(signedValidToken)
	instantiateMsgStr, err := json.Marshal(instantiateMsg)
	require.NoError(t, err)

	// Register AA with valid token
	registerCmd := []string{
		"abstract-account", "register",
		codeIDStr, string(instantiateMsgStr),
		"--salt", salt,
		"--funds", "10000uxion",
		"--chain-id", xion.Config().ChainID,
	}

	_, err = testlib.ExecTx(t, ctx, xion.GetNode(), xionUser.KeyName(), registerCmd...)
	require.NoError(t, err)
	t.Logf("✓ AA registered with valid token")

	err = testutil.WaitForBlocks(ctx, 2, xion)
	require.NoError(t, err)

	// Get contract address
	contractsResponse, err := testlib.ExecQuery(t, ctx, xion.GetNode(), "wasm", "contracts", codeIDStr)
	require.NoError(t, err)
	contract := contractsResponse["contracts"].([]interface{})[0].(string)

	// Fund contract
	err = xion.SendFunds(ctx, xionUser.FormattedAddress(), ibc.WalletAmount{
		Address: contract,
		Denom:   xion.Config().Denom,
		Amount:  math.NewInt(100_000),
	})
	require.NoError(t, err)
	err = testutil.WaitForBlocks(ctx, 2, xion)
	require.NoError(t, err)

	// Get account details
	accountResponse, err := testlib.ExecQuery(t, ctx, xion.GetNode(), "auth", "account", contract)
	require.NoError(t, err)
	ac := accountResponse["account"].(map[string]interface{})
	acData := ac["value"]
	accountJSON, err := json.Marshal(acData)
	require.NoError(t, err)

	var account aatypes.AbstractAccount
	err = xion.Config().EncodingConfig.Codec.UnmarshalJSON(accountJSON, &account)
	require.NoError(t, err)

	// Wait for additional blocks to ensure all transactions are settled
	// Including registration funds (10k) and SendFunds (100k)
	err = testutil.WaitForBlocks(ctx, 5, xion)
	require.NoError(t, err)

	// Get initial balance before invalid signature test
	// Expected: 10,000 (from registration --funds) + 100,000 (from SendFunds) = 110,000
	initialBalance, err := xion.GetBalance(ctx, contract, xion.Config().Denom)
	require.NoError(t, err)
	expectedBalance := math.NewInt(110_000) // 10k from registration + 100k from SendFunds
	require.Equal(t, expectedBalance, initialBalance, "Initial balance should be 110,000 uxion (10k registration + 100k SendFunds)")
	t.Logf("Initial balance before invalid signature test: %s", initialBalance.String())

	// Now attempt transaction with INVALID signature (signed with different key)
	t.Log("Testing invalid signature rejection...")

	// Generate a DIFFERENT RSA key (key B) - simulating attacker's key
	privateKeyB, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	t.Logf("✓ Generated different key for attack simulation")

	// Create transaction to sign
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
	           "amount": "1000"
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
		`, contract, xionUser.FormattedAddress(), xion.Config().Denom)

	tx, err := xion.Config().EncodingConfig.TxConfig.TxJSONDecoder()([]byte(sendMsg))
	require.NoError(t, err)
	txBuilder, err := xion.Config().EncodingConfig.TxConfig.WrapTxBuilder(tx)
	require.NoError(t, err)

	// Create sign bytes
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
		},
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
	require.True(t, ok)
	txData := adaptableTx.GetSigningTxData()

	signBytes, err := xion.Config().EncodingConfig.TxConfig.SignModeHandler().GetSignBytes(
		ctx,
		signingv1beta1.SignMode(signing.SignMode_SIGN_MODE_DIRECT),
		signerData, txData)
	require.NoError(t, err)

	signatureBz := sha256.Sum256(signBytes)
	signature = base64.StdEncoding.EncodeToString(signatureBz[:])

	// Create JWT with WRONG key (key B instead of key A)
	now = time.Now()
	fiveAgo = now.Add(-time.Second * 5)
	inFive = now.Add(time.Minute * 5)

	invalidToken := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss":              aud,
		"sub":              sub,
		"aud":              auds,
		"exp":              inFive.Unix(),
		"nbf":              fiveAgo.Unix(),
		"iat":              fiveAgo.Unix(),
		"transaction_hash": signature,
	})

	// CRITICAL: Sign with WRONG key (privateKeyB instead of privateKeyA)
	signedInvalidToken, err := invalidToken.SignedString(privateKeyB)
	require.NoError(t, err)
	t.Logf("✓ Created JWT signed with incorrect key")

	sigBytes := append([]byte{0}, []byte(signedInvalidToken)...)
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

	// Attempt to broadcast transaction with invalid signature
	output, err := testlib.ExecBroadcast(t, ctx, xion.GetNode(), jsonTx)

	// Transaction should FAIL - invalid signature
	if err == nil {
		t.Logf("Broadcast output: %s", output)
		var result map[string]interface{}
		if unmarshalErr := json.Unmarshal([]byte(output), &result); unmarshalErr == nil {
			if code, ok := result["code"]; ok && code != float64(0) {
				t.Logf("✓ Transaction correctly rejected with code: %v", code)
				t.Logf("✓ Invalid signature was detected and rejected")
			} else {
				t.Fatalf("❌ SECURITY FAILURE: Transaction with invalid signature was ACCEPTED! This is a critical vulnerability.")
			}
		} else {
			t.Fatalf("❌ SECURITY FAILURE: Could not parse transaction result")
		}
	} else {
		// Also acceptable - failed at broadcast level
		t.Logf("✓ Transaction correctly rejected at broadcast: %s", err.Error())
		require.Contains(t, err.Error(), "signature", "Error should mention signature verification")
	}

	// Verify account balance unchanged (transaction didn't execute)
	finalBalance, err := xion.GetBalance(ctx, contract, xion.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, initialBalance, finalBalance, "Balance should be unchanged after failed transaction")

	t.Log("✅ SECURITY TEST PASSED: Invalid JWT signatures are correctly rejected")
	t.Log("✅ Authentication bypass prevented")
}
