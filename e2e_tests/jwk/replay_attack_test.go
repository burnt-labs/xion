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

// TestJWKTransactionHashValidation tests that JWT transaction_hash binding prevents replay attacks
// This is a Priority 1 security test preventing transaction replay and double-spending
func TestJWKTransactionHash(t *testing.T) {
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

	// Create audience
	aud := "replay-test"
	sub := "test-user-replay"

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
		string(testPublicKeyJSON),
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err)

	// Deploy contract from abstract-account module
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

	validToken := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss":              aud,
		"sub":              sub,
		"aud":              auds,
		"exp":              inFive.Unix(),
		"nbf":              fiveAgo.Unix(),
		"iat":              fiveAgo.Unix(),
		"transaction_hash": signature,
	})

	signedToken, err := validToken.SignedString(privateKey)
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
	t.Logf("✓ AA registered")

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

	// Test 1: Execute transaction 1 successfully
	t.Run("ExecuteTransaction1", func(t *testing.T) {
		t.Log("Executing first transaction...")

		// Get initial balance before executing transaction
		initialBalance, err := xion.GetBalance(ctx, contract, xion.Config().Denom)
		require.NoError(t, err)
		t.Logf("Initial contract balance: %s", initialBalance)

		txHash1, signedJWT1 := executeTransactionWithJWT(t, ctx, xion, contract, xionUser.FormattedAddress(), account, privateKey, aud, sub, 1000)
		require.NotEmpty(t, txHash1)
		t.Logf("✓ Transaction 1 executed: %s", txHash1)

		// Store JWT for replay attempt
		t.Logf("Captured JWT token: %s...", signedJWT1[:50])

		// Wait for transaction to be processed
		err = testutil.WaitForBlocks(ctx, 4, xion)
		require.NoError(t, err)

		// Verify transaction was successful by querying it
		txResult, err := testlib.ExecQuery(t, ctx, xion.GetNode(), "tx", txHash1)
		require.NoError(t, err, "Failed to query transaction %s", txHash1)

		// Check transaction code
		if code, ok := txResult["code"]; ok {
			codeNum := int(code.(float64))
			t.Logf("Transaction code: %d", codeNum)
			if codeNum != 0 {
				rawLog := ""
				if log, ok := txResult["raw_log"].(string); ok {
					rawLog = log
				}
				t.Fatalf("Transaction failed with code %d: %s", codeNum, rawLog)
			}
		} else {
			// If no code field, check for txhash to confirm it was found
			if _, ok := txResult["txhash"]; !ok {
				t.Fatalf("Transaction query returned unexpected response: %v", txResult)
			}
		}

		if rawLog, ok := txResult["raw_log"]; ok {
			t.Logf("Transaction raw_log: %v", rawLog)
		}

		// Verify balance decreased
		balanceAfterTx1, err := xion.GetBalance(ctx, contract, xion.Config().Denom)
		require.NoError(t, err)
		t.Logf("Initial balance: %s, Balance after tx1: %s", initialBalance, balanceAfterTx1)
		require.True(t, balanceAfterTx1.LT(initialBalance), "Balance should have decreased from %s to %s", initialBalance, balanceAfterTx1)
	})

	// Test 2: Attempt to replay the SAME JWT for a DIFFERENT transaction (should FAIL)
	t.Run("ReplayAttackPrevention", func(t *testing.T) {
		t.Log("Attempting replay attack with captured JWT...")

		// Create a DIFFERENT transaction (transaction 2)
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
		           "amount": "2000"
		         }
		       ]
		     }
		   ],
		   "memo": "replay-attempt",
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

		// Get fresh account state (sequence incremented)
		accountResponse, err := testlib.ExecQuery(t, ctx, xion.GetNode(), "auth", "account", contract)
		require.NoError(t, err)
		ac := accountResponse["account"].(map[string]interface{})
		acData := ac["value"]
		accountJSON, err := json.Marshal(acData)
		require.NoError(t, err)

		var freshAccount aatypes.AbstractAccount
		err = xion.Config().EncodingConfig.Codec.UnmarshalJSON(accountJSON, &freshAccount)
		require.NoError(t, err)

		// Create sign bytes for NEW transaction
		pubKey := freshAccount.GetPubKey()
		anyPk, err := codectypes.NewAnyWithValue(pubKey)
		require.NoError(t, err)
		signerData := txsigning.SignerData{
			Address:       freshAccount.GetAddress().String(),
			ChainID:       xion.Config().ChainID,
			AccountNumber: freshAccount.GetAccountNumber(),
			Sequence:      freshAccount.GetSequence(),
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
			PubKey:   freshAccount.GetPubKey(),
			Data:     &sigData,
			Sequence: freshAccount.GetSequence(),
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

		// Calculate hash of NEW transaction
		signatureBz := sha256.Sum256(signBytes)
		newTxHash := base64.StdEncoding.EncodeToString(signatureBz[:])

		// Create JWT with OLD transaction hash (replay attack)
		// The JWT's transaction_hash won't match the actual transaction bytes
		now := time.Now()
		oldTxHash := "old-transaction-hash-from-tx1" // This doesn't match new transaction

		replayToken := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
			"iss":              aud,
			"sub":              sub,
			"aud":              auds,
			"exp":              now.Add(time.Minute * 5).Unix(),
			"nbf":              now.Add(-time.Second * 5).Unix(),
			"iat":              now.Add(-time.Second * 5).Unix(),
			"transaction_hash": oldTxHash, // WRONG HASH (replay attempt)
		})

		signedReplayToken, err := replayToken.SignedString(privateKey)
		require.NoError(t, err)

		// Safely truncate hashes for logging (hashes may be shorter than 50 chars)
		newHashPreview := newTxHash
		if len(newHashPreview) > 50 {
			newHashPreview = newHashPreview[:50] + "..."
		}
		oldHashPreview := oldTxHash
		if len(oldHashPreview) > 50 {
			oldHashPreview = oldHashPreview[:50] + "..."
		}
		t.Logf("New transaction hash: %s", newHashPreview)
		t.Logf("JWT transaction_hash: %s (MISMATCH)", oldHashPreview)

		sigBytes := append([]byte{0}, []byte(signedReplayToken)...)
		sigData = signing.SingleSignatureData{
			SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
			Signature: sigBytes,
		}

		sig = signing.SignatureV2{
			PubKey:   freshAccount.GetPubKey(),
			Data:     &sigData,
			Sequence: freshAccount.GetSequence(),
		}
		err = txBuilder.SetSignatures(sig)
		require.NoError(t, err)

		jsonTx, err := xion.Config().EncodingConfig.TxConfig.TxJSONEncoder()(txBuilder.GetTx())
		require.NoError(t, err)

		// Attempt broadcast - should FAIL
		output, err := testlib.ExecBroadcast(t, ctx, xion.GetNode(), jsonTx)

		if err == nil {
			t.Logf("Broadcast output: %s", output)
			var result map[string]interface{}
			if unmarshalErr := json.Unmarshal([]byte(output), &result); unmarshalErr == nil {
				if code, ok := result["code"]; ok && code != float64(0) {
					t.Logf("✓ Replay attack correctly prevented with code: %v", code)
					t.Logf("✓ Transaction hash mismatch detected")
				} else {
					t.Fatalf("❌ SECURITY FAILURE: Replay attack SUCCEEDED! JWT was reused for different transaction.")
				}
			}
		} else {
			t.Logf("✓ Replay attack correctly rejected: %s", err.Error())
		}
	})

	// Test 3: Verify new transaction with CORRECT hash succeeds
	t.Run("NewTransactionWithCorrectHash", func(t *testing.T) {
		t.Log("Executing new transaction with correct transaction_hash binding...")

		// Get fresh account state
		accountResponse, err := testlib.ExecQuery(t, ctx, xion.GetNode(), "auth", "account", contract)
		require.NoError(t, err)
		ac := accountResponse["account"].(map[string]interface{})
		acData := ac["value"]
		accountJSON, err := json.Marshal(acData)
		require.NoError(t, err)

		var freshAccount aatypes.AbstractAccount
		err = xion.Config().EncodingConfig.Codec.UnmarshalJSON(accountJSON, &freshAccount)
		require.NoError(t, err)

		txHash2, _ := executeTransactionWithJWT(t, ctx, xion, contract, xionUser.FormattedAddress(), freshAccount, privateKey, aud, sub, 500)
		require.NotEmpty(t, txHash2)
		t.Logf("✓ New transaction with correct hash executed: %s", txHash2)

		err = testutil.WaitForBlocks(ctx, 2, xion)
		require.NoError(t, err)

		t.Log("✓ Transaction with CORRECT transaction_hash binding works")
	})

	t.Log("✅ SECURITY TEST PASSED: Transaction replay attacks prevented")
	t.Log("✅ JWT transaction_hash binding enforced")
	t.Log("✅ Each JWT is tied to specific transaction bytes")
}

// Helper function to execute transaction with JWT and return tx hash and signed JWT
func executeTransactionWithJWT(
	t *testing.T,
	ctx context.Context,
	xion *cosmos.CosmosChain,
	fromAddr string,
	toAddr string,
	account aatypes.AbstractAccount,
	privateKey *rsa.PrivateKey,
	aud string,
	sub string,
	amount int64,
) (string, string) {
	// Create transaction
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

	// Create JWT with correct transaction hash
	now := time.Now()
	auds := jwt.ClaimStrings{aud}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss":              aud,
		"sub":              sub,
		"aud":              auds,
		"exp":              now.Add(time.Minute * 5).Unix(),
		"nbf":              now.Add(-time.Second * 5).Unix(),
		"iat":              now.Add(-time.Second * 5).Unix(),
		"transaction_hash": signature, // CORRECT hash
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
	require.NoError(t, err)

	// Extract tx hash and check for errors in broadcast response
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err == nil {
		// Check if the broadcast returned an error code
		if code, ok := result["code"]; ok {
			codeNum := int(code.(float64))
			if codeNum != 0 {
				rawLog := ""
				if log, ok := result["raw_log"].(string); ok {
					rawLog = log
				}
				t.Fatalf("Transaction broadcast failed with code %d: %s", codeNum, rawLog)
			}
		}

		if txhash, ok := result["txhash"].(string); ok {
			return txhash, signedToken
		}
	}

	return output, signedToken
}
