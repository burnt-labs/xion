package keeper_test

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/stretchr/testify/require"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/burnt-labs/xion/x/jwk/types"
)

// TestValidateJWTRejectsJWSJSON verifies that the JWS JSON serialization
// parsing-confusion attack (bug bounty #65653) is blocked at the keeper level.
//
// Attack theory: lestrrat-go/jwx accepts both compact and JSON JWS. An attacker
// crafts a JWS JSON object with an extra "garbage" field containing dots, so that
// the contract's '.' split extracts a different payload than what was verified.
//
// Fix: ValidateJWT rejects any SigBytes starting with '{'.
func TestValidateJWTRejectsJWSJSON(t *testing.T) {
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
	aud := "jws-json-test"
	sub := "jws-json-user"
	k.SetAudience(ctx, types.Audience{
		Admin: admin,
		Aud:   aud,
		Key:   string(pubJSON),
	})

	// Helper: create a valid compact JWT
	makeCompactJWT := func() string {
		now := time.Now()
		tok, err := jwt.NewBuilder().
			Audience([]string{aud}).
			Subject(sub).
			Issuer(aud).
			IssuedAt(now.Add(-time.Second)).
			NotBefore(now.Add(-time.Second)).
			Expiration(now.Add(time.Hour)).
			Claim("transaction_hash", "test-hash").
			Build()
		require.NoError(t, err)

		signed, err := jwt.Sign(tok, jwt.WithKey(jwa.RS256, rsaPriv))
		require.NoError(t, err)
		return string(signed)
	}

	t.Run("compact JWT accepted", func(t *testing.T) {
		resp, err := k.ValidateJWT(ctx, &types.QueryValidateJWTRequest{
			Aud:      aud,
			Sub:      sub,
			SigBytes: makeCompactJWT(),
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		// Verify transaction_hash claim returned
		found := false
		for _, pc := range resp.PrivateClaims {
			if pc.Key == "transaction_hash" {
				found = true
				require.Equal(t, "test-hash", pc.Value)
			}
		}
		require.True(t, found)
	})

	t.Run("flattened JWS JSON rejected", func(t *testing.T) {
		payload, _ := json.Marshal(map[string]interface{}{
			"aud": []string{aud}, "sub": sub, "iss": aud,
			"exp": 9999999999, "iat": 1000000000, "nbf": 1000000000,
			"transaction_hash": "legit-hash",
		})
		signed, err := jws.Sign(payload, jws.WithJSON(), jws.WithKey(jwa.RS256, rsaPriv))
		require.NoError(t, err)

		// Flattened JWS starts with '{'
		require.Equal(t, byte('{'), signed[0])

		resp, err := k.ValidateJWT(ctx, &types.QueryValidateJWTRequest{
			Aud:      aud,
			Sub:      sub,
			SigBytes: string(signed),
		})
		require.Error(t, err)
		require.Nil(t, resp)
		require.Contains(t, err.Error(), "JWS JSON serialization is not supported")
	})

	t.Run("newline byte in JWS JSON rejected", func(t *testing.T) {
		payload, _ := json.Marshal(map[string]interface{}{
			"aud": []string{aud}, "sub": sub, "iss": aud,
			"exp": 9999999999, "iat": 1000000000, "nbf": 1000000000,
			"transaction_hash": "legit-hash",
		})
		signed, err := jws.Sign(payload, jws.WithJSON(), jws.WithKey(jwa.RS256, rsaPriv))
		require.NoError(t, err)

		// add a newline at the start to test that the first byte check is not fooled by whitespace
		signed = append([]byte("\t"), signed...)

		resp, err := k.ValidateJWT(ctx, &types.QueryValidateJWTRequest{
			Aud:      aud,
			Sub:      sub,
			SigBytes: string(signed),
		})
		require.Error(t, err)
		require.Nil(t, resp)
		require.Contains(t, err.Error(), "JWS JSON serialization is not supported")
	})

	t.Run("crafted JWS JSON with garbage field rejected", func(t *testing.T) {
		// This is the actual attack: sign a payload with WRONG hash,
		// then craft a JWS JSON with a garbage field that positions
		// the CORRECT hash where the contract's '.' split would find it.
		wrongHash := "WRONG_HASH"
		correctHash := "CORRECT_HASH_TARGET"

		payload, _ := json.Marshal(map[string]interface{}{
			"aud": []string{aud}, "sub": sub, "iss": aud,
			"exp": 9999999999, "iat": 1000000000, "nbf": 1000000000,
			"transaction_hash": wrongHash,
		})

		signed, err := jws.Sign(payload, jws.WithJSON(), jws.WithKey(jwa.RS256, rsaPriv))
		require.NoError(t, err)

		var flat map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(signed, &flat))

		// Craft the fake payload base64
		fakePayload := fmt.Sprintf(`{"transaction_hash":"%s"}`, correctHash)
		fakeB64 := base64.RawURLEncoding.EncodeToString([]byte(fakePayload))

		// Build crafted general JWS with garbage field
		crafted, err := json.Marshal(map[string]interface{}{
			"garbage":    fmt.Sprintf("junk.%s.punk", fakeB64),
			"payload":    flat["payload"],
			"signatures": []map[string]json.RawMessage{{"protected": flat["protected"], "signature": flat["signature"]}},
		})
		require.NoError(t, err)

		// Starts with '{'
		require.Equal(t, byte('{'), crafted[0])

		resp, err := k.ValidateJWT(ctx, &types.QueryValidateJWTRequest{
			Aud:      aud,
			Sub:      sub,
			SigBytes: string(crafted),
		})
		require.Error(t, err)
		require.Nil(t, resp)
		require.Contains(t, err.Error(), "JWS JSON serialization is not supported")
	})

	t.Run("crafted JWS JSON with garbage field and leading whitespaces rejected", func(t *testing.T) {
		// This is the actual attack: sign a payload with WRONG hash,
		// then craft a JWS JSON with a garbage field that positions
		// the CORRECT hash where the contract's '.' split would find it.
		wrongHash := "WRONG_HASH"
		correctHash := "CORRECT_HASH_TARGET"

		payload, _ := json.Marshal(map[string]interface{}{
			"aud": []string{aud}, "sub": sub, "iss": aud,
			"exp": 9999999999, "iat": 1000000000, "nbf": 1000000000,
			"transaction_hash": wrongHash,
		})

		signed, err := jws.Sign(payload, jws.WithJSON(), jws.WithKey(jwa.RS256, rsaPriv))
		require.NoError(t, err)

		var flat map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(signed, &flat))

		// Craft the fake payload base64
		fakePayload := fmt.Sprintf(`{"transaction_hash":"%s"}`, correctHash)
		fakeB64 := base64.RawURLEncoding.EncodeToString([]byte(fakePayload))

		// Build crafted general JWS with garbage field
		crafted, err := json.Marshal(map[string]interface{}{
			"garbage":    fmt.Sprintf("junk.%s.punk", fakeB64),
			"payload":    flat["payload"],
			"signatures": []map[string]json.RawMessage{{"protected": flat["protected"], "signature": flat["signature"]}},
		})
		require.NoError(t, err)

		// Add leading whitespace to test that the trimmed first-byte check is not fooled
		crafted = append([]byte("\t"), crafted...)

		resp, err := k.ValidateJWT(ctx, &types.QueryValidateJWTRequest{
			Aud:      aud,
			Sub:      sub,
			SigBytes: string(crafted),
		})
		require.Error(t, err)
		require.Nil(t, resp)
		require.Contains(t, err.Error(), "JWS JSON serialization is not supported")
	})

	t.Run("JSON array rejected", func(t *testing.T) {
		// Edge case: JSON array
		resp, err := k.ValidateJWT(ctx, &types.QueryValidateJWTRequest{
			Aud:      aud,
			Sub:      sub,
			SigBytes: `[{"not":"a jwt"}]`,
		})
		require.Error(t, err)
		require.Nil(t, resp)
		// This won't hit our check (starts with '['), but jwt.Parse should reject it
	})

	t.Run("whitespace-prefixed JSON rejected", func(t *testing.T) {
		// jwt.Parse() calls bytes.TrimSpace() internally which uses unicode.IsSpace.
		// This trims \t \n \v \f \r, space, U+0085 (NEL), and U+00A0 (NBSP).
		// Verify our TrimLeftFunc(unicode.IsSpace) catches all of these.
		whitespaceVariants := []struct {
			name  string
			input string
		}{
			{"space", ` {"payload":"abc"}`},
			{"tab", "\t{\"payload\":\"abc\"}"},
			{"newline", "\n{\"payload\":\"abc\"}"},
			{"vertical tab", "\v{\"payload\":\"abc\"}"},
			{"form feed", "\f{\"payload\":\"abc\"}"},
			{"carriage return", "\r{\"payload\":\"abc\"}"},
			{"mixed whitespace", " \t\n\v\f\r{\"payload\":\"abc\"}"},
			{"NEL U+0085", "\u0085{\"payload\":\"abc\"}"},
			{"NBSP U+00A0", "\u00A0{\"payload\":\"abc\"}"},
			{"NEL then NBSP", "\u0085\u00A0{\"payload\":\"abc\"}"},
		}
		for _, wv := range whitespaceVariants {
			t.Run(wv.name, func(t *testing.T) {
				resp, err := k.ValidateJWT(ctx, &types.QueryValidateJWTRequest{
					Aud:      aud,
					Sub:      sub,
					SigBytes: wv.input,
				})
				require.Error(t, err)
				require.Nil(t, resp)
				require.Contains(t, err.Error(), "JWS JSON serialization is not supported")
			})
		}
	})

	t.Run("whitespace-only rejected", func(t *testing.T) {
		resp, err := k.ValidateJWT(ctx, &types.QueryValidateJWTRequest{
			Aud:      aud,
			Sub:      sub,
			SigBytes: "  \t\n",
		})
		require.Error(t, err)
		require.Nil(t, resp)
		require.Contains(t, err.Error(), "empty jwt")
	})
}
