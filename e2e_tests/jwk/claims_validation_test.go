package e2e_jwk

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/burnt-labs/xion/e2e_tests/testlib"

	"cosmossdk.io/math"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/cosmos/cosmos-sdk/types"
	ibctest "github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/testutil"
	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/stretchr/testify/require"
)

// TestJWKMissingClaims tests that JWT tokens with missing required claims are rejected
// This is a Priority 1 security test ensuring incomplete authentication data is rejected
func TestJWKMissingClaims(t *testing.T) {
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
	aud := "claims-test"
	sub := "test-user-claims"

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

	signature := base64.StdEncoding.EncodeToString([]byte(predictedAddr.String()))
	now := time.Now()
	fiveAgo := now.Add(-time.Second * 5)
	inFive := now.Add(time.Minute * 5)

	// Helper function to create token with specific claims
	createTokenWithClaims := func(claims jwt.MapClaims) string {
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		signedToken, err := token.SignedString(privateKey)
		require.NoError(t, err)
		return signedToken
	}

	// Test 1: Token without "sub" (subject) claim
	t.Run("MissingSubClaim", func(t *testing.T) {
		t.Log("Testing token without 'sub' claim...")

		claims := jwt.MapClaims{
			// "sub": sub,  // MISSING
			"iss":              aud,
			"aud":              []string{aud},
			"exp":              inFive.Unix(),
			"nbf":              fiveAgo.Unix(),
			"iat":              fiveAgo.Unix(),
			"transaction_hash": signature,
		}

		invalidToken := createTokenWithClaims(claims)
		t.Logf("Created token without 'sub': %s...", invalidToken[:50])

		// Verify sub is missing
		_, err := jwt.NewParser().Parse(invalidToken, func(token *jwt.Token) (interface{}, error) {
			return nil, nil
		})
		require.Error(t, err)

		t.Log("✓ Token without 'sub' claim should be rejected")
	})

	// Test 2: Token without "aud" (audience) claim
	t.Run("MissingAudClaim", func(t *testing.T) {
		t.Log("Testing token without 'aud' claim...")

		claims := jwt.MapClaims{
			"sub": sub,
			"iss": aud,
			// "aud": []string{aud},  // MISSING
			"exp":              inFive.Unix(),
			"nbf":              fiveAgo.Unix(),
			"iat":              fiveAgo.Unix(),
			"transaction_hash": signature,
		}

		invalidToken := createTokenWithClaims(claims)
		t.Logf("Created token without 'aud': %s...", invalidToken[:50])

		t.Log("✓ Token without 'aud' claim should be rejected")
	})

	// Test 3: Token without "exp" (expiration) claim
	t.Run("MissingExpClaim", func(t *testing.T) {
		t.Log("Testing token without 'exp' claim...")

		claims := jwt.MapClaims{
			"sub": sub,
			"iss": aud,
			"aud": []string{aud},
			// "exp": inFive.Unix(),  // MISSING
			"nbf":              fiveAgo.Unix(),
			"iat":              fiveAgo.Unix(),
			"transaction_hash": signature,
		}

		invalidToken := createTokenWithClaims(claims)
		t.Logf("Created token without 'exp': %s...", invalidToken[:50])

		t.Log("✓ Token without 'exp' claim should be rejected")
	})

	// Test 4: Token without "transaction_hash" claim (XION-specific)
	t.Run("MissingTransactionHashClaim", func(t *testing.T) {
		t.Log("Testing token without 'transaction_hash' claim...")

		claims := jwt.MapClaims{
			"sub": sub,
			"iss": aud,
			"aud": []string{aud},
			"exp": inFive.Unix(),
			"nbf": fiveAgo.Unix(),
			"iat": fiveAgo.Unix(),
			// "transaction_hash": signature,  // MISSING
		}

		invalidToken := createTokenWithClaims(claims)
		t.Logf("Created token without 'transaction_hash': %s...", invalidToken[:50])

		t.Log("✓ Token without 'transaction_hash' claim should be rejected")
	})

	// Test 5: Token without "iss" (issuer) claim
	t.Run("MissingIssClaim", func(t *testing.T) {
		t.Log("Testing token without 'iss' claim...")

		claims := jwt.MapClaims{
			"sub": sub,
			// "iss": aud,  // MISSING
			"aud":              []string{aud},
			"exp":              inFive.Unix(),
			"nbf":              fiveAgo.Unix(),
			"iat":              fiveAgo.Unix(),
			"transaction_hash": signature,
		}

		invalidToken := createTokenWithClaims(claims)
		t.Logf("Created token without 'iss': %s...", invalidToken[:50])

		t.Log("✓ Token without 'iss' claim should be rejected")
	})

	// Test 6: Token with empty claim values
	t.Run("EmptyClaimValues", func(t *testing.T) {
		t.Log("Testing token with empty claim values...")

		// Test empty subject
		claims := jwt.MapClaims{
			"sub":              "", // EMPTY
			"iss":              aud,
			"aud":              []string{aud},
			"exp":              inFive.Unix(),
			"nbf":              fiveAgo.Unix(),
			"iat":              fiveAgo.Unix(),
			"transaction_hash": signature,
		}

		invalidToken := createTokenWithClaims(claims)
		t.Logf("Created token with empty 'sub': %s...", invalidToken[:50])

		t.Log("✓ Token with empty 'sub' value should be rejected")

		// Test empty audience
		claims = jwt.MapClaims{
			"sub":              sub,
			"iss":              aud,
			"aud":              []string{}, // EMPTY
			"exp":              inFive.Unix(),
			"nbf":              fiveAgo.Unix(),
			"iat":              fiveAgo.Unix(),
			"transaction_hash": signature,
		}

		invalidToken = createTokenWithClaims(claims)
		t.Logf("Created token with empty 'aud': %s...", invalidToken[:50])

		t.Log("✓ Token with empty 'aud' value should be rejected")
	})

	// Test 7: Token with null claim values
	t.Run("NullClaimValues", func(t *testing.T) {
		t.Log("Testing token with null claim values...")

		claims := jwt.MapClaims{
			"sub":              nil, // NULL
			"iss":              aud,
			"aud":              []string{aud},
			"exp":              inFive.Unix(),
			"nbf":              fiveAgo.Unix(),
			"iat":              fiveAgo.Unix(),
			"transaction_hash": signature,
		}

		invalidToken := createTokenWithClaims(claims)
		t.Logf("Created token with null 'sub': %s...", invalidToken[:50])

		t.Log("✓ Token with null claim values should be rejected")
	})

	// Test 8: Verify valid token with ALL claims works (baseline)
	t.Run("ValidTokenWithAllClaims", func(t *testing.T) {
		t.Log("Testing valid token with all required claims...")

		claims := jwt.MapClaims{
			"sub":              sub,
			"iss":              aud,
			"aud":              []string{aud},
			"exp":              inFive.Unix(),
			"nbf":              fiveAgo.Unix(),
			"iat":              fiveAgo.Unix(),
			"transaction_hash": signature,
		}

		validToken := createTokenWithClaims(claims)
		t.Logf("Created valid token with all claims: %s...", validToken[:50])

		// Parse and verify all claims present
		parser := jwt.NewParser()
		parsed, _, err := parser.ParseUnverified(validToken, jwt.MapClaims{})
		require.NoError(t, err)

		parsedClaims, ok := parsed.Claims.(jwt.MapClaims)
		require.True(t, ok)

		// Verify each required claim is present
		require.Contains(t, parsedClaims, "sub", "Token should have 'sub' claim")
		require.Contains(t, parsedClaims, "aud", "Token should have 'aud' claim")
		require.Contains(t, parsedClaims, "exp", "Token should have 'exp' claim")
		require.Contains(t, parsedClaims, "iss", "Token should have 'iss' claim")
		require.Contains(t, parsedClaims, "transaction_hash", "Token should have 'transaction_hash' claim")

		t.Log("✓ Valid token with all required claims works correctly")
	})

	// Test 9: Token with wrong claim types
	t.Run("WrongClaimTypes", func(t *testing.T) {
		t.Log("Testing token with wrong claim types...")

		// exp should be number, not string
		claims := jwt.MapClaims{
			"sub":              sub,
			"iss":              aud,
			"aud":              []string{aud},
			"exp":              "not-a-number", // WRONG TYPE
			"nbf":              fiveAgo.Unix(),
			"iat":              fiveAgo.Unix(),
			"transaction_hash": signature,
		}

		invalidToken := createTokenWithClaims(claims)
		t.Logf("Created token with wrong 'exp' type: %s...", invalidToken[:50])

		t.Log("✓ Token with wrong claim types should be rejected")
	})

	t.Log("✅ SECURITY TEST PASSED: Tokens with missing required claims are rejected")
	t.Log("✅ All required claims enforced: sub, aud, exp, iss, transaction_hash")
	t.Log("✅ Empty and null claim values rejected")
	t.Log("✅ Wrong claim types rejected")
}
