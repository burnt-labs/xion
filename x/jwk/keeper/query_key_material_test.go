package keeper_test

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/stretchr/testify/require"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/burnt-labs/xion/x/jwk/keeper"
	"github.com/burnt-labs/xion/x/jwk/types"
)

// shortEd25519JWK parses and passes the stored-key checks, but its 3-byte
// public key makes the ed25519 verifier panic.
const shortEd25519JWK = `{"kty":"OKP","crv":"Ed25519","x":"AQID","alg":"EdDSA"}`

// eddsaCompactToken is a syntactically valid compact JWS with an EdDSA header.
const eddsaCompactToken = "eyJhbGciOiJFZERTQSJ9.eyJhdWQiOiJwYW5pYy1hdWQiLCJzdWIiOiJ1c2VyIn0.AQID" //nolint:gosec // G101: unsigned fixture, not a credential

func hmacAudience(t *testing.T) ([]byte, string) {
	t.Helper()
	secret := []byte("key-material-secret")
	key, err := jwk.FromRaw(secret)
	require.NoError(t, err)
	require.NoError(t, key.Set(jwk.AlgorithmKey, jwa.HS256))
	keyJSON, err := json.Marshal(key)
	require.NoError(t, err)
	return secret, string(keyJSON)
}

func rsaPrivateJWK(t *testing.T) string {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	key, err := jwk.FromRaw(priv)
	require.NoError(t, err)
	require.NoError(t, key.Set(jwk.AlgorithmKey, jwa.RS256))
	keyJSON, err := json.Marshal(key)
	require.NoError(t, err)
	return string(keyJSON)
}

func TestQueriesRejectUnusableStoredKeys(t *testing.T) {
	k, ctx := setupKeeper(t)
	admin := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	k.SetAudience(ctx, types.Audience{Aud: "unparseable", Admin: admin, Key: "not a jwk"})
	k.SetAudience(ctx, types.Audience{Aud: "private", Admin: admin, Key: rsaPrivateJWK(t)})

	t.Run("decode jwt", func(t *testing.T) {
		_, err := k.DecodeJWT(ctx, &types.QueryDecodeJWTRequest{Aud: "unparseable", Sub: "user", SigBytes: "a.b.c"})
		require.Error(t, err)
		require.NotContains(t, err.Error(), "stored key validation failed")

		_, err = k.DecodeJWT(ctx, &types.QueryDecodeJWTRequest{Aud: "private", Sub: "user", SigBytes: "a.b.c"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "stored key validation failed")
	})

	t.Run("validate jwt", func(t *testing.T) {
		_, err := k.ValidateJWT(ctx, &types.QueryValidateJWTRequest{Aud: "private", Sub: "user", SigBytes: "a.b.c"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "stored key validation failed")
	})

	t.Run("verify jws", func(t *testing.T) {
		_, err := k.VerifyJWS(ctx, &types.QueryVerifyJWSRequest{Aud: "unparseable", SigBytes: "a.b.c"})
		require.Error(t, err)
		require.NotContains(t, err.Error(), "stored key validation failed")

		_, err = k.VerifyJWS(ctx, &types.QueryVerifyJWSRequest{Aud: "private", SigBytes: "a.b.c"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "stored key validation failed")
	})
}

func TestQueriesRecoverFromVerifierPanic(t *testing.T) {
	k, ctx := setupKeeper(t)
	admin := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	k.SetAudience(ctx, types.Audience{Aud: "panic-aud", Admin: admin, Key: shortEd25519JWK})

	t.Run("decode jwt", func(t *testing.T) {
		res, err := k.DecodeJWT(ctx, &types.QueryDecodeJWTRequest{Aud: "panic-aud", Sub: "user", SigBytes: eddsaCompactToken})
		require.Error(t, err)
		require.Nil(t, res)
		require.Contains(t, err.Error(), "panic during jwt parse")
	})

	t.Run("validate jwt", func(t *testing.T) {
		res, err := k.ValidateJWT(ctx, &types.QueryValidateJWTRequest{Aud: "panic-aud", Sub: "user", SigBytes: eddsaCompactToken})
		require.Error(t, err)
		require.Nil(t, res)
		require.Contains(t, err.Error(), "panic during jwt parse")
	})

	t.Run("verify jws", func(t *testing.T) {
		res, err := k.VerifyJWS(ctx, &types.QueryVerifyJWSRequest{Aud: "panic-aud", SigBytes: eddsaCompactToken})
		require.Error(t, err)
		require.Nil(t, res)
		require.Contains(t, err.Error(), "panic during jws verify")
	})
}

func TestDecodeJWTClaimHandling(t *testing.T) {
	k, ctx := setupKeeper(t)
	admin := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	secret, keyJSON := hmacAudience(t)
	k.SetAudience(ctx, types.Audience{Aud: "claims-aud", Admin: admin, Key: keyJSON})

	sign := func(t *testing.T, b *jwt.Builder) string {
		t.Helper()
		tok, err := b.Build()
		require.NoError(t, err)
		signed, err := jwt.Sign(tok, jwt.WithKey(jwa.HS256, secret))
		require.NoError(t, err)
		return string(signed)
	}

	t.Run("expired token is rejected", func(t *testing.T) {
		signed := sign(t, jwt.NewBuilder().Audience([]string{"claims-aud"}).Subject("user").Expiration(ctx.BlockTime().Add(-time.Hour)))
		res, err := k.DecodeJWT(ctx, &types.QueryDecodeJWTRequest{Aud: "claims-aud", Sub: "user", SigBytes: signed})
		require.Error(t, err)
		require.Nil(t, res)
		require.Contains(t, err.Error(), "exp")
	})

	t.Run("not-before and numeric claims are returned", func(t *testing.T) {
		nbf := ctx.BlockTime().Add(-time.Minute).Truncate(time.Second)
		signed := sign(t, jwt.NewBuilder().
			Audience([]string{"claims-aud"}).
			Subject("user").
			NotBefore(nbf).
			Expiration(ctx.BlockTime().Add(time.Hour)).
			Claim("count", 42).
			Claim("flag", true))
		res, err := k.DecodeJWT(ctx, &types.QueryDecodeJWTRequest{Aud: "claims-aud", Sub: "user", SigBytes: signed})
		require.NoError(t, err)

		claims := map[string]string{}
		for _, c := range res.Claims {
			claims[c.Key] = c.Value
		}
		require.Equal(t, "42", claims["count"])
		require.Equal(t, "true", claims["flag"])
		require.Equal(t, nbf.Unix(), parseInt64(t, claims["nbf"]))

		vres, err := k.ValidateJWT(ctx, &types.QueryValidateJWTRequest{Aud: "claims-aud", Sub: "user", SigBytes: signed})
		require.NoError(t, err)
		private := map[string]string{}
		for _, c := range vres.PrivateClaims {
			private[c.Key] = c.Value
		}
		require.Equal(t, "42", private["count"])
		require.Equal(t, "true", private["flag"])
	})
}

func parseInt64(t *testing.T, s string) int64 {
	t.Helper()
	v, err := strconv.ParseInt(s, 10, 64)
	require.NoError(t, err)
	return v
}

func TestUpdateAudienceRejectsInvalidNewAdmin(t *testing.T) {
	k, ctx := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)
	admin := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	_, keyJSON := hmacAudience(t)
	k.SetAudience(ctx, types.Audience{Aud: "admin-aud", Admin: admin, Key: keyJSON})

	_, err := srv.UpdateAudience(ctx, &types.MsgUpdateAudience{
		Admin:    admin,
		NewAdmin: "not-a-bech32-address",
		Aud:      "admin-aud",
		Key:      keyJSON,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid new admin address")

	stored, found := k.GetAudience(ctx, "admin-aud")
	require.True(t, found)
	require.Equal(t, admin, stored.Admin)
}
