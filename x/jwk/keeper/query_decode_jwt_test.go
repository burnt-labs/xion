package keeper_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/stretchr/testify/require"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/burnt-labs/xion/x/jwk/types"
)

func TestDecodeJWT(t *testing.T) {
	k, ctx := setupKeeper(t)
	admin := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	// Create a symmetric key for HMAC
	secretKey := []byte("test-secret-for-decode-jwt")
	key, err := jwk.FromRaw(secretKey)
	require.NoError(t, err)
	require.NoError(t, key.Set(jwk.AlgorithmKey, jwa.HS256))

	keyJSON, err := json.Marshal(key)
	require.NoError(t, err)

	aud := "decode-test-aud"
	k.SetAudience(ctx, types.Audience{
		Aud:   aud,
		Admin: admin,
		Key:   string(keyJSON),
	})

	t.Run("nil request", func(t *testing.T) {
		resp, err := k.DecodeJWT(ctx, nil)
		require.Error(t, err)
		require.Nil(t, resp)
		require.Contains(t, err.Error(), "invalid request")
	})

	t.Run("non-existent audience", func(t *testing.T) {
		resp, err := k.DecodeJWT(ctx, &types.QueryDecodeJWTRequest{
			Aud:      "does-not-exist",
			Sub:      "user",
			SigBytes: "some.jwt.here",
		})
		require.Error(t, err)
		require.Nil(t, resp)
		require.Contains(t, err.Error(), "not found")
	})

	t.Run("empty sig_bytes", func(t *testing.T) {
		resp, err := k.DecodeJWT(ctx, &types.QueryDecodeJWTRequest{
			Aud:      aud,
			Sub:      "user",
			SigBytes: "",
		})
		require.Error(t, err)
		require.Nil(t, resp)
		require.Contains(t, err.Error(), "empty jwt")
	})

	t.Run("whitespace-only input", func(t *testing.T) {
		resp, err := k.DecodeJWT(ctx, &types.QueryDecodeJWTRequest{
			Aud:      aud,
			Sub:      "user",
			SigBytes: "  \t\n",
		})
		require.Error(t, err)
		require.Nil(t, resp)
		require.Contains(t, err.Error(), "empty jwt")
	})

	t.Run("JWS JSON serialization rejected", func(t *testing.T) {
		resp, err := k.DecodeJWT(ctx, &types.QueryDecodeJWTRequest{
			Aud:      aud,
			Sub:      "user",
			SigBytes: `{"payload":"abc","signatures":[]}`,
		})
		require.Error(t, err)
		require.Nil(t, resp)
		require.Contains(t, err.Error(), "JWS JSON serialization is not supported")
	})

	t.Run("success returns all claims", func(t *testing.T) {
		token, err := jwt.NewBuilder().
			Audience([]string{aud}).
			Subject("test-user").
			Issuer("test-issuer").
			IssuedAt(time.Unix(1000000, 0)).
			Expiration(time.Unix(9999999999, 0)).
			JwtID("jwt-id-123").
			Claim("custom_str", "hello").
			Claim("custom_num", 42).
			Claim("custom_obj", map[string]interface{}{"nested": "value"}).
			Build()
		require.NoError(t, err)

		signed, err := jwt.Sign(token, jwt.WithKey(jwa.HS256, secretKey))
		require.NoError(t, err)

		resp, err := k.DecodeJWT(ctx, &types.QueryDecodeJWTRequest{
			Aud:      aud,
			Sub:      "test-user",
			SigBytes: string(signed),
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		// Build a map for easy lookup
		claimMap := make(map[string]string)
		for _, c := range resp.Claims {
			claimMap[c.Key] = c.Value
		}

		// Standard claims
		require.Equal(t, "test-issuer", claimMap["iss"])
		require.Equal(t, "test-user", claimMap["sub"])
		require.Contains(t, claimMap["aud"], aud)
		require.Equal(t, "9999999999", claimMap["exp"])
		require.Equal(t, "1000000", claimMap["iat"])
		require.Equal(t, "jwt-id-123", claimMap["jti"])

		// Private claims
		require.Equal(t, "hello", claimMap["custom_str"])
		require.Equal(t, "42", claimMap["custom_num"])
		require.Contains(t, claimMap["custom_obj"], `"nested":"value"`)

		// Verify sorted order
		for i := 1; i < len(resp.Claims); i++ {
			require.True(t, resp.Claims[i-1].Key <= resp.Claims[i].Key,
				"claims should be sorted: %s should come before %s", resp.Claims[i-1].Key, resp.Claims[i].Key)
		}
	})
}
