package types_test

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/burnt-labs/xion/x/jwk/types"
)

func TestValidateJWKKeySize(t *testing.T) {
	admin := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	validKey := `{"kty":"RSA","use":"sig","kid":"test","alg":"RS256","n":"0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw","e":"AQAB"}`

	t.Run("valid 2048-bit RSA key", func(t *testing.T) {
		msg := types.NewMsgCreateAudience(admin, "https://test.example.com", validKey)
		err := msg.ValidateBasic()
		require.NoError(t, err)
	})

	t.Run("oversized RSA key rejected in CreateAudience", func(t *testing.T) {
		oversizedKey := generateOversizedRSAJWK(t, 8192)
		msg := types.NewMsgCreateAudience(admin, "https://test.example.com", oversizedKey)
		err := msg.ValidateBasic()
		require.Error(t, err)
		require.Contains(t, err.Error(), "exceeds maximum allowed")
	})

	t.Run("oversized RSA key rejected in UpdateAudience", func(t *testing.T) {
		oversizedKey := generateOversizedRSAJWK(t, 8192)
		msg := types.NewMsgUpdateAudience(admin, admin, "https://test.example.com", "", oversizedKey)
		err := msg.ValidateBasic()
		require.Error(t, err)
		require.Contains(t, err.Error(), "exceeds maximum allowed")
	})

	t.Run("4096-bit RSA key accepted", func(t *testing.T) {
		key4096 := generateOversizedRSAJWK(t, 4096)
		msg := types.NewMsgCreateAudience(admin, "https://test.example.com", key4096)
		err := msg.ValidateBasic()
		require.NoError(t, err)
	})
}

// generateOversizedRSAJWK creates a JWK JSON string with the given RSA key size
func generateOversizedRSAJWK(t *testing.T, bits int) string {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	require.NoError(t, err)

	n := base64.RawURLEncoding.EncodeToString(privateKey.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1}) // 65537

	return `{"kty":"RSA","use":"sig","kid":"test-oversized","alg":"RS256","n":"` + n + `","e":"` + e + `"}`
}
