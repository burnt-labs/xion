package e2e_jwk

import (
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
	"github.com/cosmos/cosmos-sdk/client"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	ibctest "github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/cosmos/interchaintest/v10/testutil"
	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/anypb"
)

// TestJWSJSONSerializationRejected verifies that JWS JSON serialization
// is rejected by the chain, preventing the parsing-confusion attack
// described in bug bounty report #65653.
//
// Attack theory:
//   - ValidateJWT (lestrrat-go/jwx) accepts both compact and JSON JWS
//   - The smart contract (jwt.rs) splits sig_bytes on '.' assuming compact format
//   - An attacker crafts a JWS JSON with a "garbage" field positioned so the
//     '.' split extracts a fake transaction_hash while ValidateJWT validates
//     the real (differently-hashed) payload
//
// This test proves the attack does NOT work because:
//  1. Plain JWS JSON format is rejected (contract's '.' split fails to find valid payload)
//  2. Crafted JWS JSON with garbage field and mismatched hashes is rejected
//  3. Only standard compact JWTs with correct transaction_hash succeed
func TestJWSJSONSerializationRejected(t *testing.T) {
	ctx := t.Context()
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	t.Parallel()

	// ─── Setup: chain, user, audience, AA contract ───────────────────

	spec := testlib.XionLocalChainSpec(t, 3, 1)
	xion := testlib.BuildXionChainWithSpec(t, spec)

	config := types.GetConfig()
	config.SetBech32PrefixForAccount("xion", "xionpub")

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

	// Register audience
	aud := "jws-json-test"
	sub := "jws-json-user"

	_, err = testlib.ExecTx(t, ctx, xion.GetNode(),
		xionUser.KeyName(),
		"jwk", "create-audience-claim", aud,
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err)

	_, err = testlib.ExecTx(t, ctx, xion.GetNode(),
		xionUser.KeyName(),
		"jwk", "create-audience", aud, string(testPublicKeyJSON),
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err)

	// Deploy & register AA contract
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

	// Create registration JWT (compact, standard)
	regSignature := base64.StdEncoding.EncodeToString([]byte(predictedAddr.String()))
	now := time.Now()
	regToken := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss":              aud,
		"sub":              sub,
		"aud":              jwt.ClaimStrings{aud},
		"exp":              now.Add(time.Minute * 5).Unix(),
		"nbf":              now.Add(-time.Second * 5).Unix(),
		"iat":              now.Add(-time.Second * 5).Unix(),
		"transaction_hash": regSignature,
	})
	signedRegToken, err := regToken.SignedString(privateKey)
	require.NoError(t, err)

	instantiateMsg := CreateJWTAuthenticatorMsg(AAAuthenticatorConfig{
		Subject:  sub,
		Audience: aud,
		ID:       0,
		Token:    []byte(signedRegToken),
	})
	instantiateMsgStr, err := json.Marshal(instantiateMsg)
	require.NoError(t, err)

	_, err = testlib.ExecTx(t, ctx, xion.GetNode(), xionUser.KeyName(),
		"abstract-account", "register",
		codeIDStr, string(instantiateMsgStr),
		"--salt", salt,
		"--funds", "10000uxion",
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err)
	t.Log("AA registered successfully")

	err = testutil.WaitForBlocks(ctx, 2, xion)
	require.NoError(t, err)

	// Get contract address & fund it
	contractsResponse, err := testlib.ExecQuery(t, ctx, xion.GetNode(), "wasm", "contracts", codeIDStr)
	require.NoError(t, err)
	contract := contractsResponse["contracts"].([]interface{})[0].(string)

	err = xion.SendFunds(ctx, xionUser.FormattedAddress(), ibc.WalletAmount{
		Address: contract,
		Denom:   xion.Config().Denom,
		Amount:  math.NewInt(100_000),
	})
	require.NoError(t, err)
	err = testutil.WaitForBlocks(ctx, 2, xion)
	require.NoError(t, err)

	// Helper: get current account state
	getAccount := func() aatypes.AbstractAccount {
		resp, err := testlib.ExecQuery(t, ctx, xion.GetNode(), "auth", "account", contract)
		require.NoError(t, err)
		acJSON, err := json.Marshal(resp["account"].(map[string]interface{})["value"])
		require.NoError(t, err)
		var acct aatypes.AbstractAccount
		err = xion.Config().EncodingConfig.Codec.UnmarshalJSON(acJSON, &acct)
		require.NoError(t, err)
		return acct
	}

	// Helper: build a send tx and return (txBuilder, signBytes hash as base64)
	buildSendTx := func(account aatypes.AbstractAccount, amount string, memo string) (client.TxBuilder, string) {
		sendMsg := fmt.Sprintf(`{
			"body": {
				"messages": [{
					"@type": "/cosmos.bank.v1beta1.MsgSend",
					"from_address": "%s",
					"to_address": "%s",
					"amount": [{"denom": "%s", "amount": "%s"}]
				}],
				"memo": "%s",
				"timeout_height": "0",
				"extension_options": [],
				"non_critical_extension_options": []
			},
			"auth_info": {
				"signer_infos": [],
				"fee": {"amount": [], "gas_limit": "200000", "payer": "", "granter": ""},
				"tip": null
			},
			"signatures": []
		}`, contract, xionUser.FormattedAddress(), xion.Config().Denom, amount, memo)

		tx, err := xion.Config().EncodingConfig.TxConfig.TxJSONDecoder()([]byte(sendMsg))
		require.NoError(t, err)
		txBuilder, err := xion.Config().EncodingConfig.TxConfig.WrapTxBuilder(tx)
		require.NoError(t, err)

		pubKey := account.GetPubKey()
		anyPk, err := codectypes.NewAnyWithValue(pubKey)
		require.NoError(t, err)
		signerData := txsigning.SignerData{
			Address:       account.GetAddress().String(),
			ChainID:       xion.Config().ChainID,
			AccountNumber: account.GetAccountNumber(),
			Sequence:      account.GetSequence(),
			PubKey:        &anypb.Any{TypeUrl: anyPk.TypeUrl, Value: anyPk.Value},
		}

		sig := signing.SignatureV2{
			PubKey:   account.GetPubKey(),
			Data:     &signing.SingleSignatureData{SignMode: signing.SignMode_SIGN_MODE_DIRECT},
			Sequence: account.GetSequence(),
		}
		err = txBuilder.SetSignatures(sig)
		require.NoError(t, err)

		builtTx := txBuilder.GetTx()
		adaptableTx, ok := builtTx.(authsigning.V2AdaptableTx)
		require.True(t, ok)

		signBytes, err := xion.Config().EncodingConfig.TxConfig.SignModeHandler().GetSignBytes(
			ctx,
			signingv1beta1.SignMode(signing.SignMode_SIGN_MODE_DIRECT),
			signerData, adaptableTx.GetSigningTxData())
		require.NoError(t, err)

		hash := sha256.Sum256(signBytes)
		return txBuilder, base64.StdEncoding.EncodeToString(hash[:])
	}

	// Helper: set JWT signature on txBuilder and broadcast, return (txHash, error)
	broadcastWithSig := func(txBuilder client.TxBuilder, account aatypes.AbstractAccount, sigBytes []byte) (string, error) {
		sigData := signing.SingleSignatureData{
			SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
			Signature: sigBytes,
		}
		sig := signing.SignatureV2{
			PubKey:   account.GetPubKey(),
			Data:     &sigData,
			Sequence: account.GetSequence(),
		}
		err := txBuilder.SetSignatures(sig)
		require.NoError(t, err)

		jsonTx, err := xion.Config().EncodingConfig.TxConfig.TxJSONEncoder()(txBuilder.GetTx())
		require.NoError(t, err)

		return testlib.ExecBroadcastWithFlags(t, ctx, xion.GetNode(), jsonTx, "--output", "json")
	}

	// ─── Test 1: Compact JWT with correct hash SUCCEEDS (baseline) ───

	t.Run("CompactJWT_CorrectHash_Succeeds", func(t *testing.T) {
		account := getAccount()

		balanceBefore, err := xion.GetBalance(ctx, contract, xion.Config().Denom)
		require.NoError(t, err)

		txBuilder, txHash := buildSendTx(account, "1000", "compact-correct")

		signedToken := CreateValidJWTToken(t, &JWTTestKey{
			PrivateKey: privateKey,
		}, aud, sub, txHash)

		sigBytes := append([]byte{0}, []byte(signedToken)...)
		txResult, err := broadcastWithSig(txBuilder, account, sigBytes)
		require.NoError(t, err, "compact JWT with correct hash should succeed")
		t.Logf("Compact JWT tx succeeded: %s", txResult)

		balanceAfter, err := xion.GetBalance(ctx, contract, xion.Config().Denom)
		require.NoError(t, err)
		require.True(t, balanceAfter.LT(balanceBefore), "balance should decrease")
		t.Log("PASS: compact JWT with correct transaction_hash works")
	})

	// ─── Test 2: Plain JWS JSON serialization is REJECTED ────────────

	t.Run("JWSJSONSerialization_Rejected", func(t *testing.T) {
		account := getAccount()
		txBuilder, txHash := buildSendTx(account, "1000", "jws-json-plain")

		// Create a standard JWS JSON serialization with the CORRECT hash
		now := time.Now()
		payload := map[string]interface{}{
			"iss":              aud,
			"sub":              sub,
			"aud":              []string{aud},
			"exp":              now.Add(time.Minute * 5).Unix(),
			"nbf":              now.Add(-time.Second * 5).Unix(),
			"iat":              now.Add(-time.Second * 5).Unix(),
			"transaction_hash": txHash,
		}
		payloadBytes, err := json.Marshal(payload)
		require.NoError(t, err)

		// Sign using JWS JSON Serialization (lestrrat jws.WithJSON())
		signedJWS, err := jws.Sign(
			payloadBytes,
			jws.WithJSON(),
			jws.WithKey(jwa.RS256, privateKey),
		)
		require.NoError(t, err)
		t.Logf("JWS JSON output: %s", string(signedJWS))

		// Verify this IS valid JSON (not compact)
		var jsonCheck map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(signedJWS, &jsonCheck), "should be valid JSON")
		require.Contains(t, jsonCheck, "payload", "should have payload field")

		sigBytes := append([]byte{0}, signedJWS...)
		_, err = broadcastWithSig(txBuilder, account, sigBytes)
		require.Error(t, err, "JWS JSON serialization MUST be rejected")
		t.Logf("PASS: plain JWS JSON correctly rejected: %s", err)
	})

	// ─── Test 3: Crafted JWS JSON with garbage field attack ──────────

	t.Run("JWSJSONWithGarbageField_AttackRejected", func(t *testing.T) {
		account := getAccount()

		// Build TWO transactions: one the attacker wants to authorize (target)
		// and one whose JWT was legitimately signed (source)
		txBuilder, targetTxHash := buildSendTx(account, "50000", "attack-target")

		// The attacker has a JWT signed for a DIFFERENT hash (source tx)
		sourceTxHash := "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="

		now := time.Now()
		sourcePayload := map[string]interface{}{
			"iss":              aud,
			"sub":              sub,
			"aud":              []string{aud},
			"exp":              now.Add(time.Minute * 5).Unix(),
			"nbf":              now.Add(-time.Second * 5).Unix(),
			"iat":              now.Add(-time.Second * 5).Unix(),
			"transaction_hash": sourceTxHash, // WRONG hash for target tx
		}
		sourcePayloadBytes, err := json.Marshal(sourcePayload)
		require.NoError(t, err)

		// Sign the source payload legitimately as JWS JSON
		signedJWS, err := jws.Sign(
			sourcePayloadBytes,
			jws.WithJSON(),
			jws.WithKey(jwa.RS256, privateKey),
		)
		require.NoError(t, err)

		// Parse the JWS JSON into components
		var flatMap map[string]json.RawMessage
		err = json.Unmarshal(signedJWS, &flatMap)
		require.NoError(t, err)

		// Construct the "garbage" field that positions the TARGET hash
		// so the contract's '.' split extracts it as the second segment
		fakePayload := fmt.Sprintf(`{"transaction_hash":"%s"}`, targetTxHash)
		fakePayloadB64 := base64.RawURLEncoding.EncodeToString([]byte(fakePayload))

		// Build the crafted JWS JSON:
		// Go's json.Marshal sorts keys alphabetically: "garbage" < "payload" < "signatures"
		// After '.' split:
		//   segment[0] = {"garbage":"junk
		//   segment[1] = <fakePayloadB64>    ← attacker wants contract to extract this
		//   segment[2] = punk","payload":"...","signatures":[...]}
		craftedJWS, err := json.Marshal(map[string]interface{}{
			"garbage": fmt.Sprintf("junk.%s.punk", fakePayloadB64),
			"payload": json.RawMessage(flatMap["payload"]),
			"signatures": []map[string]json.RawMessage{{
				"protected": flatMap["protected"],
				"signature": flatMap["signature"],
			}},
		})
		require.NoError(t, err)
		t.Logf("Crafted JWS: %s", string(craftedJWS))

		// Verify the structure: json keys should be ordered garbage, payload, signatures
		require.True(t, craftedJWS[0] == '{', "should start with {")

		// Log what the '.' split would produce
		parts := splitOnDot(craftedJWS)
		t.Logf("Dot-split segment count: %d", len(parts))
		for i, p := range parts {
			preview := string(p)
			if len(preview) > 80 {
				preview = preview[:80] + "..."
			}
			t.Logf("  segment[%d]: %s", i, preview)
		}
		t.Logf("Target tx hash: %s", targetTxHash)
		t.Logf("Source tx hash (in JWT): %s", sourceTxHash)

		sigBytes := append([]byte{0}, craftedJWS...)
		_, err = broadcastWithSig(txBuilder, account, sigBytes)
		require.Error(t, err, "crafted JWS JSON attack MUST be rejected")
		t.Logf("PASS: crafted JWS JSON with garbage field correctly rejected: %s", err)
	})

	// ─── Test 4: Compact JWT with WRONG hash is REJECTED ─────────────

	t.Run("CompactJWT_WrongHash_Rejected", func(t *testing.T) {
		account := getAccount()
		txBuilder, _ := buildSendTx(account, "1000", "compact-wrong-hash")

		// Use a wrong transaction hash
		wrongHash := "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
		signedToken := CreateValidJWTToken(t, &JWTTestKey{
			PrivateKey: privateKey,
		}, aud, sub, wrongHash)

		sigBytes := append([]byte{0}, []byte(signedToken)...)
		_, err := broadcastWithSig(txBuilder, account, sigBytes)
		require.Error(t, err, "compact JWT with wrong hash MUST be rejected")
		t.Logf("PASS: compact JWT with wrong transaction_hash correctly rejected: %s", err)
	})

	t.Log("ALL TESTS PASSED: JWS JSON serialization attack is not viable")
}

// splitOnDot splits a byte slice on '.' characters (for test logging only)
func splitOnDot(data []byte) [][]byte {
	var parts [][]byte
	start := 0
	for i, b := range data {
		if b == '.' {
			parts = append(parts, data[start:i])
			start = i + 1
		}
	}
	parts = append(parts, data[start:])
	return parts
}
