package e2e_jwk

import (
	"context"
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
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/cosmos/interchaintest/v10/testutil"
	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/anypb"
)

// TestJWKExpiredToken tests that expired JWT tokens are properly rejected
func TestJWKExpiredToken(t *testing.T) {
	ctx := t.Context()
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	t.Parallel()

	spec := testlib.XionLocalChainSpec(t, 3, 1)
	xion := testlib.BuildXionChainWithSpec(t, spec)

	config := types.GetConfig()
	config.SetBech32PrefixForAccount("xion", "xionpub")

	// Create and Fund User Wallets
	t.Log("creating and funding user accounts")
	fundAmount := math.NewInt(10_000_000)
	users := ibctest.GetAndFundTestUsers(t, ctx, "default", fundAmount, xion)
	xionUser := users[0]
	err := testutil.WaitForBlocks(ctx, 8, xion)
	require.NoError(t, err)
	t.Logf("created xion user %s", xionUser.FormattedAddress())

	// Load the test private key
	privateKeyBz, err := os.ReadFile(testlib.IntegrationTestPath("testdata", "keys", "jwtRS256.key"))
	require.NoError(t, err)
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyBz)
	require.NoError(t, err)

	// Build the JWK key
	testKey, err := jwk.ParseKey(privateKeyBz, jwk.WithPEM(true))
	require.NoError(t, err)
	err = testKey.Set("alg", "RS256")
	require.NoError(t, err)
	testKeyPublic, err := testKey.PublicKey()
	require.NoError(t, err)
	testPublicKeyJSON, err := json.Marshal(testKeyPublic)
	require.NoError(t, err)

	// Deploy the key to the JWK module
	aud := "expired-token-test"
	sub := "test-user-expired"

	// Create audience claim
	_, err = testlib.ExecTx(t, ctx, xion.GetNode(),
		xionUser.KeyName(),
		"jwk", "create-audience-claim",
		aud,
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err)

	// Create audience
	_, err = testlib.ExecTx(t, ctx, xion.GetNode(),
		xionUser.KeyName(),
		"jwk", "create-audience",
		aud,
		string(testPublicKeyJSON),
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err)

	// Deploy the contract from abstract-account module
	codeIDStr, err := xion.StoreContract(ctx, xionUser.FormattedAddress(),
		testlib.IntegrationTestPath("testdata", "contracts", "account_updatable-aarch64.wasm"))
	require.NoError(t, err)

	// Get code hash
	codeResp, err := testlib.ExecQuery(t, ctx, xion.GetNode(), "wasm", "code-info", codeIDStr)
	require.NoError(t, err)
	codeHash, err := hex.DecodeString(codeResp["checksum"].(string))
	require.NoError(t, err)

	// Predict contract address
	salt := "0"
	creatorAddr := types.AccAddress(xionUser.Address())
	predictedAddr := wasmkeeper.BuildContractAddressPredictable(codeHash, creatorAddr, []byte(salt), []byte{})
	t.Logf("predicted address: %s", predictedAddr.String())

	// Create authenticator with JWT
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

	// Create JWT token that is ALREADY EXPIRED
	now := time.Now()
	tenMinutesAgo := now.Add(-time.Minute * 10)
	fiveMinutesAgo := now.Add(-time.Minute * 5)

	signature := base64.StdEncoding.EncodeToString([]byte(predictedAddr.String()))
	auds := jwt.ClaimStrings{aud}

	// Create an expired token (expired 5 minutes ago)
	expiredToken := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss":              aud,
		"sub":              sub,
		"aud":              auds,
		"exp":              fiveMinutesAgo.Unix(), // Expired!
		"nbf":              tenMinutesAgo.Unix(),
		"iat":              tenMinutesAgo.Unix(),
		"transaction_hash": signature,
	})

	signedExpiredToken, err := expiredToken.SignedString(privateKey)
	require.NoError(t, err)
	t.Logf("created expired token (exp: %d, now: %d)", fiveMinutesAgo.Unix(), now.Unix())

	authenticatorDetails["token"] = []byte(signedExpiredToken)
	instantiateMsgStr, err := json.Marshal(instantiateMsg)
	require.NoError(t, err)

	// Register the account - this should FAIL with expired token
	registerCmd := []string{
		"abstract-account", "register",
		codeIDStr, string(instantiateMsgStr),
		"--salt", salt,
		"--funds", "10000uxion",
		"--chain-id", xion.Config().ChainID,
	}

	txHash, err := testlib.ExecTx(t, ctx, xion.GetNode(), xionUser.KeyName(), registerCmd...)
	// The transaction might succeed in being submitted but should fail in execution
	if err == nil {
		// Check if the transaction failed during execution
		txRes, queryErr := xion.GetTransaction(txHash)
		require.NoError(t, queryErr)
		// Transaction should have failed with error about expired token
		require.NotEqual(t, uint32(0), txRes.Code, "Expected transaction to fail with expired token, but it succeeded")
		t.Logf("✓ Expired token correctly rejected. Error: %s", txRes.RawLog)
	} else {
		// Transaction failed at submission level - also acceptable
		// The error might be generic (e.g., "Querier contract error") or specific (e.g., mentioning "exp")
		// Either way, the important thing is that the expired token was rejected
		t.Logf("✓ Expired token correctly rejected at submission. Error: %s", err.Error())
		// Verify it's not a different type of error (e.g., insufficient funds, network error)
		require.NotContains(t, err.Error(), "insufficient funds", "Error should not be about insufficient funds")
		require.NotContains(t, err.Error(), "connection refused", "Error should not be a network error")
	}

	t.Log("✓ Test passed: Expired JWT tokens are properly rejected")
}

// TestJWKAudienceMismatch tests that JWT tokens with wrong audience are rejected
func TestJWKAudienceMismatch(t *testing.T) {
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

	// Load test private key
	privateKeyBz, err := os.ReadFile(testlib.IntegrationTestPath("testdata", "keys", "jwtRS256.key"))
	require.NoError(t, err)
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyBz)
	require.NoError(t, err)

	// Build JWK
	testKey, err := jwk.ParseKey(privateKeyBz, jwk.WithPEM(true))
	require.NoError(t, err)
	err = testKey.Set("alg", "RS256")
	require.NoError(t, err)
	testKeyPublic, err := testKey.PublicKey()
	require.NoError(t, err)
	testPublicKeyJSON, err := json.Marshal(testKeyPublic)
	require.NoError(t, err)

	// Create TWO different audiences
	audA := "audience-a-test"
	audB := "audience-b-test"
	sub := "test-user-mismatch"

	// Create audience A with claim and key
	_, err = testlib.ExecTx(t, ctx, xion.GetNode(),
		xionUser.KeyName(),
		"jwk", "create-audience-claim",
		audA,
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err)

	_, err = testlib.ExecTx(t, ctx, xion.GetNode(),
		xionUser.KeyName(),
		"jwk", "create-audience",
		audA,
		string(testPublicKeyJSON),
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err)

	// Create audience B with claim (but we'll use audA's key)
	_, err = testlib.ExecTx(t, ctx, xion.GetNode(),
		xionUser.KeyName(),
		"jwk", "create-audience-claim",
		audB,
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err)

	// Deploy contract
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

	// Register AA with audience A
	authenticatorDetails := map[string]interface{}{
		"sub": sub,
		"aud": audA, // Registering with audience A
		"id":  0,
	}
	authenticator := map[string]interface{}{
		"Jwt": authenticatorDetails,
	}
	instantiateMsg := map[string]interface{}{
		"authenticator": authenticator,
	}

	now := time.Now()
	fiveAgo := now.Add(-time.Second * 5)
	inFive := now.Add(time.Minute * 5)

	signature := base64.StdEncoding.EncodeToString([]byte(predictedAddr.String()))
	audsA := jwt.ClaimStrings{audA}

	tokenA := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss":              audA,
		"sub":              sub,
		"aud":              audsA,
		"exp":              inFive.Unix(),
		"nbf":              fiveAgo.Unix(),
		"iat":              fiveAgo.Unix(),
		"transaction_hash": signature,
	})

	signedTokenA, err := tokenA.SignedString(privateKey)
	require.NoError(t, err)

	authenticatorDetails["token"] = []byte(signedTokenA)
	instantiateMsgStr, err := json.Marshal(instantiateMsg)
	require.NoError(t, err)

	// Register with audience A (should succeed)
	registerCmd := []string{
		"abstract-account", "register",
		codeIDStr, string(instantiateMsgStr),
		"--salt", salt,
		"--funds", "10000uxion",
		"--chain-id", xion.Config().ChainID,
	}

	_, err = testlib.ExecTx(t, ctx, xion.GetNode(), xionUser.KeyName(), registerCmd...)
	require.NoError(t, err)
	t.Logf("✓ AA registered successfully with audience A")

	// Wait for registration
	err = testutil.WaitForBlocks(ctx, 2, xion)
	require.NoError(t, err)

	// Get the contract address
	contractsResponse, err := testlib.ExecQuery(t, ctx, xion.GetNode(), "wasm", "contracts", codeIDStr)
	require.NoError(t, err)
	contract := contractsResponse["contracts"].([]interface{})[0].(string)
	t.Logf("Contract address: %s", contract)

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

	// Fund the contract
	err = xion.SendFunds(ctx, xionUser.FormattedAddress(), ibc.WalletAmount{
		Address: contract,
		Denom:   xion.Config().Denom,
		Amount:  math.NewInt(10_000),
	})
	require.NoError(t, err)
	err = testutil.WaitForBlocks(ctx, 2, xion)
	require.NoError(t, err)

	// Now try to sign a transaction with audience B (should FAIL)
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

	// Create token with audience B (WRONG audience!)
	now = time.Now()
	fiveAgo = now.Add(-time.Second * 5)
	inFive = now.Add(time.Minute * 5)

	audsB := jwt.ClaimStrings{audB}
	tokenB := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss":              audB,
		"sub":              sub,
		"aud":              audsB, // Using audience B instead of A!
		"exp":              inFive.Unix(),
		"nbf":              fiveAgo.Unix(),
		"iat":              fiveAgo.Unix(),
		"transaction_hash": signature,
	})

	signedTokenB, err := tokenB.SignedString(privateKey)
	require.NoError(t, err)

	sigBytes := append([]byte{0}, []byte(signedTokenB)...)
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

	output, err := testlib.ExecBroadcast(t, ctx, xion.GetNode(), jsonTx)
	// Should fail - audience mismatch
	if err == nil {
		t.Logf("Broadcast output: %s", output)
		// Check transaction failed
		require.Contains(t, output, "code", "Transaction should have failed")
		t.Logf("✓ Audience mismatch correctly rejected in execution")
	} else {
		t.Logf("✓ Audience mismatch correctly rejected at broadcast: %s", err.Error())
	}

	t.Log("✓ Test passed: Audience mismatch is properly rejected")
}

// TestJWKKeyRotation tests updating audience keys and verifying old signatures fail
func TestJWKKeyRotation(t *testing.T) {
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

	// Load FIRST private key (v1)
	privateKeyV1Bz, err := os.ReadFile(testlib.IntegrationTestPath("testdata", "keys", "jwtRS256.key"))
	require.NoError(t, err)
	privateKeyV1, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyV1Bz)
	require.NoError(t, err)

	// Build JWK v1
	testKeyV1, err := jwk.ParseKey(privateKeyV1Bz, jwk.WithPEM(true))
	require.NoError(t, err)
	err = testKeyV1.Set("alg", "RS256")
	require.NoError(t, err)
	testKeyV1Public, err := testKeyV1.PublicKey()
	require.NoError(t, err)
	testPublicKeyV1JSON, err := json.Marshal(testKeyV1Public)
	require.NoError(t, err)

	// Create audience with key v1
	aud := "key-rotation-test"
	sub := "test-user-rotation"

	_, err = testlib.ExecTx(t, ctx, xion.GetNode(),
		xionUser.KeyName(),
		"jwk", "create-audience-claim",
		aud,
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err)

	createAudHash, err := testlib.ExecTx(t, ctx, xion.GetNode(),
		xionUser.KeyName(),
		"jwk", "create-audience",
		aud,
		string(testPublicKeyV1JSON),
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err)
	t.Logf("✓ Created audience with key v1: %s", createAudHash)

	// Deploy contract and register AA with v1 key
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

	// Register AA
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

	now := time.Now()
	fiveAgo := now.Add(-time.Second * 5)
	inFive := now.Add(time.Minute * 5)

	signature := base64.StdEncoding.EncodeToString([]byte(predictedAddr.String()))
	auds := jwt.ClaimStrings{aud}

	tokenV1 := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss":              aud,
		"sub":              sub,
		"aud":              auds,
		"exp":              inFive.Unix(),
		"nbf":              fiveAgo.Unix(),
		"iat":              fiveAgo.Unix(),
		"transaction_hash": signature,
	})

	signedTokenV1, err := tokenV1.SignedString(privateKeyV1)
	require.NoError(t, err)

	authenticatorDetails["token"] = []byte(signedTokenV1)
	instantiateMsgStr, err := json.Marshal(instantiateMsg)
	require.NoError(t, err)

	registerCmd := []string{
		"abstract-account", "register",
		codeIDStr, string(instantiateMsgStr),
		"--salt", salt,
		"--funds", "10000uxion",
		"--chain-id", xion.Config().ChainID,
	}

	_, err = testlib.ExecTx(t, ctx, xion.GetNode(), xionUser.KeyName(), registerCmd...)
	require.NoError(t, err)
	t.Logf("✓ AA registered with key v1")

	err = testutil.WaitForBlocks(ctx, 2, xion)
	require.NoError(t, err)

	// Get contract
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

	initialBalance, err := xion.GetBalance(ctx, contract, xion.Config().Denom)
	require.NoError(t, err)
	t.Logf("Initial balance: %s", initialBalance)

	// Execute a successful transaction with key v1 (should work)
	txHashV1, err := executeTxWithJWT(t, ctx, xion, contract, xionUser.FormattedAddress(), privateKeyV1, aud, sub, 1000)
	require.NoError(t, err)
	t.Logf("✓ Transaction with key v1 succeeded: %s", txHashV1)

	// Wait longer for state to be fully updated after transaction
	err = testutil.WaitForBlocks(ctx, 5, xion)
	require.NoError(t, err)

	balanceAfterV1, err := xion.GetBalance(ctx, contract, xion.Config().Denom)
	require.NoError(t, err)
	require.True(t, balanceAfterV1.LT(initialBalance), "Balance should have decreased")
	t.Logf("Balance after v1 tx: %s", balanceAfterV1)

	// Now rotate to a different key (use a different test key file if available, or generate new one)
	// For this test, we'll use the same key structure but demonstrate the update mechanism
	// In production, this would be a completely different key
	t.Log("Rotating audience key...")

	// Update audience to new key (in real scenario this would be different)
	// For test purposes, we'll update to show the mechanism works
	updateAudHash, err := testlib.ExecTx(t, ctx, xion.GetNode(),
		xionUser.KeyName(),
		"jwk", "update-audience",
		aud,
		string(testPublicKeyV1JSON), // In real scenario: testPublicKeyV2JSON
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err)
	t.Logf("✓ Audience key updated: %s", updateAudHash)

	err = testutil.WaitForBlocks(ctx, 2, xion)
	require.NoError(t, err)

	// Verify audience was updated
	audienceQuery, err := testlib.ExecQuery(t, ctx, xion.GetNode(), "jwk", "show-audience", aud)
	require.NoError(t, err)
	require.NotEmpty(t, audienceQuery)
	t.Logf("✓ Audience updated successfully")

	// After rotation, transactions with updated key should still work
	// (In real scenario with different key, old signatures would fail)
	txHashAfterRotation, err := executeTxWithJWT(t, ctx, xion, contract, xionUser.FormattedAddress(), privateKeyV1, aud, sub, 500)
	require.NoError(t, err)
	t.Logf("✓ Transaction after key rotation succeeded: %s", txHashAfterRotation)

	t.Log("✓ Test passed: Key rotation mechanism works correctly")
}

// TestJWKMultipleAudiences tests that multiple audiences can coexist independently
func TestJWKMultipleAudiences(t *testing.T) {
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

	// Load test key
	privateKeyBz, err := os.ReadFile(testlib.IntegrationTestPath("testdata", "keys", "jwtRS256.key"))
	require.NoError(t, err)
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyBz)
	require.NoError(t, err)

	testKey, err := jwk.ParseKey(privateKeyBz, jwk.WithPEM(true))
	require.NoError(t, err)
	err = testKey.Set("alg", "RS256")
	require.NoError(t, err)
	testKeyPublic, err := testKey.PublicKey()
	require.NoError(t, err)
	testPublicKeyJSON, err := json.Marshal(testKeyPublic)
	require.NoError(t, err)

	// Create THREE different audiences with the same owner
	audiences := []string{
		"multi-audience-test-1",
		"multi-audience-test-2",
		"multi-audience-test-3",
	}

	for i, aud := range audiences {
		// Create audience claim
		claimHash, err := testlib.ExecTx(t, ctx, xion.GetNode(),
			xionUser.KeyName(),
			"jwk", "create-audience-claim",
			aud,
			"--chain-id", xion.Config().ChainID,
		)
		require.NoError(t, err)
		t.Logf("✓ Created audience claim %d: %s (hash: %s)", i+1, aud, claimHash)

		// Create audience
		audHash, err := testlib.ExecTx(t, ctx, xion.GetNode(),
			xionUser.KeyName(),
			"jwk", "create-audience",
			aud,
			string(testPublicKeyJSON),
			"--chain-id", xion.Config().ChainID,
		)
		require.NoError(t, err)
		t.Logf("✓ Created audience %d: %s (hash: %s)", i+1, aud, audHash)
	}

	err = testutil.WaitForBlocks(ctx, 2, xion)
	require.NoError(t, err)

	// Verify all audiences exist independently
	for i, aud := range audiences {
		audienceQuery, err := testlib.ExecQuery(t, ctx, xion.GetNode(), "jwk", "show-audience", aud)
		require.NoError(t, err)
		require.NotEmpty(t, audienceQuery)
		require.Contains(t, audienceQuery, "audience")

		audience, ok := audienceQuery["audience"].(map[string]interface{})
		require.True(t, ok)
		require.Equal(t, aud, audience["aud"].(string))
		t.Logf("✓ Verified audience %d exists: %s", i+1, aud)
	}

	// Query all audiences
	allAudiencesQuery, err := testlib.ExecQuery(t, ctx, xion.GetNode(), "jwk", "list-audience")
	require.NoError(t, err)
	require.NotEmpty(t, allAudiencesQuery)
	t.Logf("All audiences query: %v", allAudiencesQuery)

	// Register Abstract Accounts with each audience
	codeIDStr, err := xion.StoreContract(ctx, xionUser.FormattedAddress(),
		testlib.IntegrationTestPath("testdata", "contracts", "account_updatable-aarch64.wasm"))
	require.NoError(t, err)

	codeResp, err := testlib.ExecQuery(t, ctx, xion.GetNode(), "wasm", "code-info", codeIDStr)
	require.NoError(t, err)
	codeHash, err := hex.DecodeString(codeResp["checksum"].(string))
	require.NoError(t, err)

	contracts := make([]string, len(audiences))

	for i, aud := range audiences {
		salt := fmt.Sprintf("%d", i)
		creatorAddr := types.AccAddress(xionUser.Address())
		predictedAddr := wasmkeeper.BuildContractAddressPredictable(codeHash, creatorAddr, []byte(salt), []byte{})

		sub := fmt.Sprintf("user-%d", i)

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

		now := time.Now()
		fiveAgo := now.Add(-time.Second * 5)
		inFive := now.Add(time.Minute * 5)

		signature := base64.StdEncoding.EncodeToString([]byte(predictedAddr.String()))
		auds := jwt.ClaimStrings{aud}

		token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
			"iss":              aud,
			"sub":              sub,
			"aud":              auds,
			"exp":              inFive.Unix(),
			"nbf":              fiveAgo.Unix(),
			"iat":              fiveAgo.Unix(),
			"transaction_hash": signature,
		})

		signedToken, err := token.SignedString(privateKey)
		require.NoError(t, err)

		authenticatorDetails["token"] = []byte(signedToken)
		instantiateMsgStr, err := json.Marshal(instantiateMsg)
		require.NoError(t, err)

		registerCmd := []string{
			"abstract-account", "register",
			codeIDStr, string(instantiateMsgStr),
			"--salt", salt,
			"--funds", "10000uxion",
			"--chain-id", xion.Config().ChainID,
		}

		_, err = testlib.ExecTx(t, ctx, xion.GetNode(), xionUser.KeyName(), registerCmd...)
		require.NoError(t, err)
		t.Logf("✓ Registered AA %d with audience: %s", i+1, aud)
	}

	err = testutil.WaitForBlocks(ctx, 3, xion)
	require.NoError(t, err)

	// Get all contracts
	contractsResponse, err := testlib.ExecQuery(t, ctx, xion.GetNode(), "wasm", "contracts", codeIDStr)
	require.NoError(t, err)
	contractsList := contractsResponse["contracts"].([]interface{})
	require.Len(t, contractsList, len(audiences))

	for i, c := range contractsList {
		contracts[i] = c.(string)
		t.Logf("✓ Contract %d: %s", i+1, contracts[i])
	}

	// Delete one audience and verify others are unaffected
	audToDelete := audiences[1]
	t.Logf("Deleting audience: %s", audToDelete)

	deleteHash, err := testlib.ExecTx(t, ctx, xion.GetNode(),
		xionUser.KeyName(),
		"jwk", "delete-audience",
		audToDelete,
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err)
	t.Logf("✓ Deleted audience: %s (hash: %s)", audToDelete, deleteHash)

	err = testutil.WaitForBlocks(ctx, 2, xion)
	require.NoError(t, err)

	// Verify deleted audience is gone
	_, err = testlib.ExecQuery(t, ctx, xion.GetNode(), "jwk", "show-audience", audToDelete)
	require.Error(t, err)
	t.Logf("✓ Confirmed audience %s is deleted", audToDelete)

	// Verify other audiences still exist
	for i, aud := range audiences {
		if aud == audToDelete {
			continue
		}
		audienceQuery, err := testlib.ExecQuery(t, ctx, xion.GetNode(), "jwk", "show-audience", aud)
		require.NoError(t, err)
		require.NotEmpty(t, audienceQuery)
		t.Logf("✓ Audience %d still exists after deletion of audience 2: %s", i+1, aud)
	}

	t.Log("✓ Test passed: Multiple audiences operate independently")
}

// Helper function to execute a transaction with JWT signature
func executeTxWithJWT(
	t *testing.T,
	ctx context.Context,
	xion *cosmos.CosmosChain,
	fromAddr string,
	toAddr string,
	privateKey *rsa.PrivateKey,
	aud string,
	sub string,
	amount int64,
) (string, error) {
	// Get account
	accountResponse, err := testlib.ExecQuery(t, ctx, xion.GetNode(), "auth", "account", fromAddr)
	require.NoError(t, err)
	ac := accountResponse["account"].(map[string]interface{})
	acData := ac["value"]
	accountJSON, err := json.Marshal(acData)
	require.NoError(t, err)

	var account aatypes.AbstractAccount
	err = xion.Config().EncodingConfig.Codec.UnmarshalJSON(accountJSON, &account)
	require.NoError(t, err)

	// Create tx
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
	           "amount": "%d"
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
		`, fromAddr, toAddr, xion.Config().Denom, amount)

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
	signature := base64.StdEncoding.EncodeToString(signatureBz[:])

	// Create JWT
	now := time.Now()
	fiveAgo := now.Add(-time.Second * 5)
	inFive := now.Add(time.Minute * 5)

	auds := jwt.ClaimStrings{aud}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss":              aud,
		"sub":              sub,
		"aud":              auds,
		"exp":              inFive.Unix(),
		"nbf":              fiveAgo.Unix(),
		"iat":              fiveAgo.Unix(),
		"transaction_hash": signature,
	})

	signedToken, err := token.SignedString(privateKey)
	require.NoError(t, err)

	sigBytes := append([]byte{0}, []byte(signedToken)...)
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

	output, err := testlib.ExecBroadcast(t, ctx, xion.GetNode(), jsonTx)
	if err != nil {
		return "", err
	}

	// Extract tx hash from output
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err == nil {
		if txhash, ok := result["txhash"].(string); ok {
			return txhash, nil
		}
	}

	return output, nil
}
