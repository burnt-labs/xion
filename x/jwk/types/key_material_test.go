package types_test

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"strings"
	"testing"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/stretchr/testify/require"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/burnt-labs/xion/x/jwk/types"
)

func jwkJSON(t *testing.T, raw interface{}, alg jwa.KeyAlgorithm) string {
	t.Helper()
	key, err := jwk.FromRaw(raw)
	require.NoError(t, err)
	require.NoError(t, key.Set(jwk.AlgorithmKey, alg))
	bz, err := json.Marshal(key)
	require.NoError(t, err)
	return string(bz)
}

func parseJWK(t *testing.T, s string) jwk.Key {
	t.Helper()
	key, err := jwk.ParseKey([]byte(s))
	require.NoError(t, err)
	return key
}

func TestValidateJWKKeySizeRejectsPrivateAndUnexportableKeys(t *testing.T) {
	rsaPriv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	ecPriv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	_, edPriv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	cases := []struct {
		name    string
		key     string
		wantMsg string
	}{
		{"rsa private", jwkJSON(t, rsaPriv, jwa.RS256), "RSA private keys must not be stored on-chain"},
		{"ecdsa private", jwkJSON(t, ecPriv, jwa.ES256), "ECDSA private keys must not be stored on-chain"},
		{"ed25519 private", jwkJSON(t, edPriv, jwa.EdDSA), "Ed25519 private keys must not be stored on-chain"},
		{"unknown curve", `{"kty":"EC","crv":"P-999","x":"AQ","y":"AQ","alg":"ES256"}`, "unable to extract raw key"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := types.ValidateJWKKeySize(parseJWK(t, tc.key))
			require.ErrorIs(t, err, types.ErrInvalidJWK)
			require.Contains(t, err.Error(), tc.wantMsg)
		})
	}
}

func TestMsgCreateAudienceRejectsEdDSAOnNonOKPKey(t *testing.T) {
	rsaPriv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	admin := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	msg := types.NewMsgCreateAudience(admin, "aud", jwkJSON(t, &rsaPriv.PublicKey, jwa.EdDSA))
	err = msg.ValidateBasic()
	require.ErrorIs(t, err, types.ErrInvalidJWK)
	require.Contains(t, err.Error(), "requires kty=OKP")
}

func TestMsgUpdateAudienceSizeLimits(t *testing.T) {
	admin := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	t.Run("key too large", func(t *testing.T) {
		msg := types.NewMsgUpdateAudience(admin, admin, "aud", "", strings.Repeat("k", types.MaxJWKKeySize+1))
		err := msg.ValidateBasic()
		require.ErrorIs(t, err, types.ErrInvalidJWK)
		require.Contains(t, err.Error(), "key size")
	})

	t.Run("aud too large", func(t *testing.T) {
		msg := types.NewMsgUpdateAudience(admin, admin, strings.Repeat("a", types.MaxAudSize+1), "", "{}")
		err := msg.ValidateBasic()
		require.Error(t, err)
		require.Contains(t, err.Error(), "aud length")
	})
}

func TestMsgDeleteAudienceClaimRejectsShortHash(t *testing.T) {
	admin := authtypes.NewModuleAddress(govtypes.ModuleName)
	msg := types.NewMsgDeleteAudienceClaim(admin, []byte{1, 2, 3})
	err := msg.ValidateBasic()
	require.Error(t, err)
	require.Contains(t, err.Error(), "got 3 bytes")
}

func TestGenesisStateValidateRejectsWeakOrMismatchedKeys(t *testing.T) {
	admin := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	rsaPriv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	t.Run("symmetric algorithm", func(t *testing.T) {
		gs := types.GenesisState{
			Params:       types.DefaultParams(),
			AudienceList: []types.Audience{{Aud: "hmac", Admin: admin, Key: jwkJSON(t, []byte("secret"), jwa.HS256)}},
		}
		err := gs.Validate()
		require.ErrorIs(t, err, types.ErrInvalidJWK)
		require.Contains(t, err.Error(), "invalid algorithm HS256")
	})

	t.Run("missing algorithm", func(t *testing.T) {
		key, err := jwk.FromRaw(&rsaPriv.PublicKey)
		require.NoError(t, err)
		bz, err := json.Marshal(key)
		require.NoError(t, err)
		gs := types.GenesisState{
			Params:       types.DefaultParams(),
			AudienceList: []types.Audience{{Aud: "no-alg", Admin: admin, Key: string(bz)}},
		}
		err = gs.Validate()
		require.ErrorIs(t, err, types.ErrInvalidJWK)
		require.Contains(t, err.Error(), "invalid algorithm in genesis JWK")
	})

	t.Run("private key material", func(t *testing.T) {
		gs := types.GenesisState{
			Params:       types.DefaultParams(),
			AudienceList: []types.Audience{{Aud: "private", Admin: admin, Key: jwkJSON(t, rsaPriv, jwa.RS256)}},
		}
		err := gs.Validate()
		require.ErrorIs(t, err, types.ErrInvalidJWK)
		require.Contains(t, err.Error(), "invalid key material in genesis JWK")
	})

	t.Run("algorithm does not match key type", func(t *testing.T) {
		gs := types.GenesisState{
			Params:       types.DefaultParams(),
			AudienceList: []types.Audience{{Aud: "mismatch", Admin: admin, Key: jwkJSON(t, &rsaPriv.PublicKey, jwa.ES256)}},
		}
		err := gs.Validate()
		require.ErrorIs(t, err, types.ErrInvalidJWK)
		require.Contains(t, err.Error(), "key type/algorithm mismatch")
	})
}
