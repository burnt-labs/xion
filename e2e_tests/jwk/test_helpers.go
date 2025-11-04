package e2e_jwk

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/burnt-labs/xion/e2e_tests/testlib"

	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/stretchr/testify/require"
)

// JWTTestKey represents a test JWT key pair
type JWTTestKey struct {
	PrivateKey      *rsa.PrivateKey
	PublicKeyJSON   []byte
	PrivateKeyBytes []byte
}

// LoadJWTTestKey loads the default test JWT key from testdata
func LoadJWTTestKey(t *testing.T) *JWTTestKey {
	privateKeyBz, err := os.ReadFile(testlib.IntegrationTestPath("testdata", "keys", "jwtRS256.key"))
	require.NoError(t, err, "Failed to read JWT test key")

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyBz)
	require.NoError(t, err, "Failed to parse JWT private key")

	testKey, err := jwk.ParseKey(privateKeyBz, jwk.WithPEM(true))
	require.NoError(t, err, "Failed to parse JWK")

	err = testKey.Set("alg", "RS256")
	require.NoError(t, err, "Failed to set algorithm")

	testKeyPublic, err := testKey.PublicKey()
	require.NoError(t, err, "Failed to get public key")

	publicKeyJSON, err := json.Marshal(testKeyPublic)
	require.NoError(t, err, "Failed to marshal public key")

	return &JWTTestKey{
		PrivateKey:      privateKey,
		PublicKeyJSON:   publicKeyJSON,
		PrivateKeyBytes: privateKeyBz,
	}
}

// JWTClaims represents common JWT claims for testing
type JWTClaims struct {
	Issuer          string
	Subject         string
	Audience        []string
	ExpirationTime  time.Time
	NotBefore       time.Time
	IssuedAt        time.Time
	TransactionHash string
}

// CreateJWTToken creates a signed JWT token with the given claims
func CreateJWTToken(t *testing.T, key *JWTTestKey, claims JWTClaims) string {
	auds := jwt.ClaimStrings(claims.Audience)

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss":              claims.Issuer,
		"sub":              claims.Subject,
		"aud":              auds,
		"exp":              claims.ExpirationTime.Unix(),
		"nbf":              claims.NotBefore.Unix(),
		"iat":              claims.IssuedAt.Unix(),
		"transaction_hash": claims.TransactionHash,
	})

	signedToken, err := token.SignedString(key.PrivateKey)
	require.NoError(t, err, "Failed to sign JWT token")

	return signedToken
}

// CreateExpiredJWTToken creates an expired JWT token for testing
func CreateExpiredJWTToken(t *testing.T, key *JWTTestKey, aud, sub, txHash string) string {
	now := time.Now()
	return CreateJWTToken(t, key, JWTClaims{
		Issuer:          aud,
		Subject:         sub,
		Audience:        []string{aud},
		ExpirationTime:  now.Add(-time.Minute * 5), // Expired 5 minutes ago
		NotBefore:       now.Add(-time.Minute * 10),
		IssuedAt:        now.Add(-time.Minute * 10),
		TransactionHash: txHash,
	})
}

// CreateValidJWTToken creates a valid (non-expired) JWT token for testing
func CreateValidJWTToken(t *testing.T, key *JWTTestKey, aud, sub, txHash string) string {
	now := time.Now()
	return CreateJWTToken(t, key, JWTClaims{
		Issuer:          aud,
		Subject:         sub,
		Audience:        []string{aud},
		ExpirationTime:  now.Add(time.Minute * 5), // Valid for 5 minutes
		NotBefore:       now.Add(-time.Second * 5),
		IssuedAt:        now.Add(-time.Second * 5),
		TransactionHash: txHash,
	})
}

// CreateJWTTokenWithCustomExpiry creates a JWT token with custom expiration
func CreateJWTTokenWithCustomExpiry(t *testing.T, key *JWTTestKey, aud, sub, txHash string, expiresAt time.Time) string {
	now := time.Now()
	return CreateJWTToken(t, key, JWTClaims{
		Issuer:          aud,
		Subject:         sub,
		Audience:        []string{aud},
		ExpirationTime:  expiresAt,
		NotBefore:       now.Add(-time.Second * 5),
		IssuedAt:        now.Add(-time.Second * 5),
		TransactionHash: txHash,
	})
}

// AudienceSetup represents a configured audience in the JWK module
type AudienceSetup struct {
	Name           string
	ClaimTxHash    string
	AudienceTxHash string
	PublicKey      []byte
}

// SetupJWKAudience creates an audience claim and audience in the JWK module
func SetupJWKAudience(
	t *testing.T,
	ctx context.Context,
	xion *cosmos.CosmosChain,
	userKeyName string,
	audienceName string,
	publicKeyJSON []byte,
) *AudienceSetup {
	// Create audience claim
	claimHash, err := testlib.ExecTx(t, ctx, xion.GetNode(),
		userKeyName,
		"jwk", "create-audience-claim",
		audienceName,
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err, "Failed to create audience claim")
	t.Logf("Created audience claim for %s: %s", audienceName, claimHash)

	// Create audience with public key
	audHash, err := testlib.ExecTx(t, ctx, xion.GetNode(),
		userKeyName,
		"jwk", "create-audience",
		audienceName,
		string(publicKeyJSON),
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err, "Failed to create audience")
	t.Logf("Created audience %s: %s", audienceName, audHash)

	return &AudienceSetup{
		Name:           audienceName,
		ClaimTxHash:    claimHash,
		AudienceTxHash: audHash,
		PublicKey:      publicKeyJSON,
	}
}

// UpdateJWKAudience updates an existing audience with a new public key
func UpdateJWKAudience(
	t *testing.T,
	ctx context.Context,
	xion *cosmos.CosmosChain,
	userKeyName string,
	audienceName string,
	newPublicKeyJSON []byte,
) string {
	updateHash, err := testlib.ExecTx(t, ctx, xion.GetNode(),
		userKeyName,
		"jwk", "update-audience",
		audienceName,
		string(newPublicKeyJSON),
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err, "Failed to update audience")
	t.Logf("Updated audience %s: %s", audienceName, updateHash)

	return updateHash
}

// DeleteJWKAudience deletes an audience from the JWK module
func DeleteJWKAudience(
	t *testing.T,
	ctx context.Context,
	xion *cosmos.CosmosChain,
	userKeyName string,
	audienceName string,
) string {
	deleteHash, err := testlib.ExecTx(t, ctx, xion.GetNode(),
		userKeyName,
		"jwk", "delete-audience",
		audienceName,
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err, "Failed to delete audience")
	t.Logf("Deleted audience %s: %s", audienceName, deleteHash)

	return deleteHash
}

// VerifyJWKAudienceExists checks that an audience exists and has correct data
func VerifyJWKAudienceExists(
	t *testing.T,
	ctx context.Context,
	xion *cosmos.CosmosChain,
	audienceName string,
) map[string]interface{} {
	audienceQuery, err := testlib.ExecQuery(t, ctx, xion.GetNode(), "jwk", "show-audience", audienceName)
	require.NoError(t, err, "Failed to query audience")
	require.NotEmpty(t, audienceQuery, "Audience query returned empty")
	require.Contains(t, audienceQuery, "audience", "Audience query missing 'audience' field")

	audience, ok := audienceQuery["audience"].(map[string]interface{})
	require.True(t, ok, "Audience field is not a map")
	require.Equal(t, audienceName, audience["aud"].(string), "Audience name mismatch")

	return audience
}

// VerifyJWKAudienceDeleted checks that an audience was successfully deleted
func VerifyJWKAudienceDeleted(
	t *testing.T,
	ctx context.Context,
	xion *cosmos.CosmosChain,
	audienceName string,
) {
	_, err := testlib.ExecQuery(t, ctx, xion.GetNode(), "jwk", "show-audience", audienceName)
	require.Error(t, err, "Audience should not exist but query succeeded")
	t.Logf("Confirmed audience %s is deleted", audienceName)
}

// AAAuthenticatorConfig represents the configuration for an Abstract Account JWT authenticator
type AAAuthenticatorConfig struct {
	Subject  string
	Audience string
	ID       int
	Token    []byte
}

// CreateJWTAuthenticatorMsg creates the instantiate message for a JWT-authenticated AA
func CreateJWTAuthenticatorMsg(config AAAuthenticatorConfig) map[string]interface{} {
	authenticatorDetails := map[string]interface{}{
		"sub": config.Subject,
		"aud": config.Audience,
		"id":  config.ID,
	}

	if config.Token != nil {
		authenticatorDetails["token"] = config.Token
	}

	authenticator := map[string]interface{}{
		"Jwt": authenticatorDetails,
	}

	return map[string]interface{}{
		"authenticator": authenticator,
	}
}
