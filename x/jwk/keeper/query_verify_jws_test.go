package keeper_test

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"testing"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/stretchr/testify/require"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/burnt-labs/xion/x/jwk/types"
)

func TestVerifyJWS(t *testing.T) {
	k, ctx := setupKeeper(t)
	admin := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	// Generate a real RSA key pair
	rsaPriv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	jwkPriv, err := jwk.FromRaw(rsaPriv)
	require.NoError(t, err)
	require.NoError(t, jwkPriv.Set("alg", "RS256"))

	jwkPub, err := jwkPriv.PublicKey()
	require.NoError(t, err)

	pubJSON, err := json.Marshal(jwkPub)
	require.NoError(t, err)

	// Register audience with the public key
	aud := "jws-verify-test"
	k.SetAudience(ctx, types.Audience{
		Admin: admin,
		Aud:   aud,
		Key:   string(pubJSON),
	})

	t.Run("nil request", func(t *testing.T) {
		resp, err := k.VerifyJWS(ctx, nil)
		require.Error(t, err)
		require.Nil(t, resp)
		require.Contains(t, err.Error(), "invalid request")
	})

	t.Run("non-existent audience", func(t *testing.T) {
		resp, err := k.VerifyJWS(ctx, &types.QueryVerifyJWSRequest{
			Aud:      "does-not-exist",
			SigBytes: "some.jws.here",
		})
		require.Error(t, err)
		require.Nil(t, resp)
		require.Contains(t, err.Error(), "not found")
	})

	t.Run("empty sig_bytes", func(t *testing.T) {
		resp, err := k.VerifyJWS(ctx, &types.QueryVerifyJWSRequest{
			Aud:      aud,
			SigBytes: "",
		})
		require.Error(t, err)
		require.Nil(t, resp)
		require.Contains(t, err.Error(), "empty jws")
	})

	t.Run("whitespace-only input", func(t *testing.T) {
		resp, err := k.VerifyJWS(ctx, &types.QueryVerifyJWSRequest{
			Aud:      aud,
			SigBytes: "  \t\n",
		})
		require.Error(t, err)
		require.Nil(t, resp)
		require.Contains(t, err.Error(), "empty jws")
	})

	t.Run("JWS JSON serialization rejected", func(t *testing.T) {
		payload := []byte(`{"hello":"world"}`)
		signed, err := jws.Sign(payload, jws.WithJSON(), jws.WithKey(jwa.RS256, rsaPriv))
		require.NoError(t, err)

		require.Equal(t, byte('{'), signed[0])

		resp, err := k.VerifyJWS(ctx, &types.QueryVerifyJWSRequest{
			Aud:      aud,
			SigBytes: string(signed),
		})
		require.Error(t, err)
		require.Nil(t, resp)
		require.Contains(t, err.Error(), "JWS JSON serialization is not supported")
	})

	t.Run("JWS JSON with leading whitespace rejected", func(t *testing.T) {
		payload := []byte(`{"hello":"world"}`)
		signed, err := jws.Sign(payload, jws.WithJSON(), jws.WithKey(jwa.RS256, rsaPriv))
		require.NoError(t, err)

		// Prepend whitespace
		withWS := "\t \n" + string(signed)

		resp, err := k.VerifyJWS(ctx, &types.QueryVerifyJWSRequest{
			Aud:      aud,
			SigBytes: withWS,
		})
		require.Error(t, err)
		require.Nil(t, resp)
		require.Contains(t, err.Error(), "JWS JSON serialization is not supported")
	})

	t.Run("invalid signature rejected", func(t *testing.T) {
		resp, err := k.VerifyJWS(ctx, &types.QueryVerifyJWSRequest{
			Aud:      aud,
			SigBytes: "aGVhZGVy.cGF5bG9hZA.aW52YWxpZA",
		})
		require.Error(t, err)
		require.Nil(t, resp)
	})

	t.Run("compact JWS success", func(t *testing.T) {
		payload := []byte(`{"action":"transfer","amount":"100"}`)
		signed, err := jws.Sign(payload, jws.WithKey(jwa.RS256, rsaPriv))
		require.NoError(t, err)

		resp, err := k.VerifyJWS(ctx, &types.QueryVerifyJWSRequest{
			Aud:      aud,
			SigBytes: string(signed),
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, payload, resp.Payload)
	})

	t.Run("non-JSON payload success", func(t *testing.T) {
		payload := []byte("arbitrary binary data: \x00\x01\x02")
		signed, err := jws.Sign(payload, jws.WithKey(jwa.RS256, rsaPriv))
		require.NoError(t, err)

		resp, err := k.VerifyJWS(ctx, &types.QueryVerifyJWSRequest{
			Aud:      aud,
			SigBytes: string(signed),
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, payload, resp.Payload)
	})

	t.Run("invalid JWK key format rejected", func(t *testing.T) {
		// Register an audience with an invalid (non-JWK) key so that
		// jwk.ParseKey fails inside VerifyJWS.
		invalidAud := "invalid-key-format"
		k.SetAudience(ctx, types.Audience{
			Admin: admin,
			Aud:   invalidAud,
			Key:   `{"not":"a-valid-jwk"}`,
		})

		payload := []byte(`{"hello":"world"}`)
		signed, err := jws.Sign(payload, jws.WithKey(jwa.RS256, rsaPriv))
		require.NoError(t, err)

		resp, err := k.VerifyJWS(ctx, &types.QueryVerifyJWSRequest{
			Aud:      invalidAud,
			SigBytes: string(signed),
		})
		require.Error(t, err)
		require.Nil(t, resp)
		require.Contains(t, err.Error(), "parse")
	})
}
