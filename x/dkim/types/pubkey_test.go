package types_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/burnt-labs/xion/x/dkim/types"
)

func TestCanonicalizeRSAPublicKeyEncodingInvariant(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	pkixBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	require.NoError(t, err)
	pkcs1Bytes := x509.MarshalPKCS1PublicKey(&key.PublicKey)

	pkixB64 := base64.StdEncoding.EncodeToString(pkixBytes)
	pkcs1B64 := base64.StdEncoding.EncodeToString(pkcs1Bytes)

	pkixDecoded, err := types.DecodePubKey(pkixB64)
	require.NoError(t, err)
	pkcs1Decoded, err := types.DecodePubKey(pkcs1B64)
	require.NoError(t, err)

	pkixParsed, err := types.ParseRSAPublicKey(pkixDecoded)
	require.NoError(t, err)
	pkcs1Parsed, err := types.ParseRSAPublicKey(pkcs1Decoded)
	require.NoError(t, err)

	pkixHash, err := types.CanonicalizeRSAPublicKey(pkixParsed)
	require.NoError(t, err)
	pkcs1Hash, err := types.CanonicalizeRSAPublicKey(pkcs1Parsed)
	require.NoError(t, err)

	require.Equal(t, pkixHash, pkcs1Hash)
	require.NotEmpty(t, pkixHash)
}

func TestParseRSAPublicKeyAcceptsSmallKeys(t *testing.T) {
	// ParseRSAPublicKey should parse without enforcing minimum key size.
	// This is needed for genesis/state-loading of legacy keys (e.g. Yahoo s1024).
	key, err := rsa.GenerateKey(rand.Reader, 1024) //nolint:gosec // G403: intentionally testing legacy 1024-bit key
	require.NoError(t, err)

	pkixBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	require.NoError(t, err)

	parsed, err := types.ParseRSAPublicKey(pkixBytes)
	require.NoError(t, err)
	require.Equal(t, 1024, parsed.N.BitLen())
}

func TestValidateRSAKeySize(t *testing.T) {
	t.Run("rejects nil key", func(t *testing.T) {
		err := types.ValidateRSAKeySize(nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "RSA public key is nil")
	})

	t.Run("rejects key with nil N", func(t *testing.T) {
		key := &rsa.PublicKey{}
		err := types.ValidateRSAKeySize(key)
		require.Error(t, err)
		require.Contains(t, err.Error(), "RSA public key is nil")
	})

	t.Run("rejects 1024-bit key", func(t *testing.T) {
		key, err := rsa.GenerateKey(rand.Reader, 1024) //nolint:gosec // G403: intentionally testing legacy 1024-bit key
		require.NoError(t, err)
		err = types.ValidateRSAKeySize(&key.PublicKey)
		require.Error(t, err)
		require.Contains(t, err.Error(), "below minimum")
	})

	t.Run("accepts 2048-bit key", func(t *testing.T) {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)
		err = types.ValidateRSAKeySize(&key.PublicKey)
		require.NoError(t, err)
	})

	t.Run("accepts 4096-bit key", func(t *testing.T) {
		key, err := rsa.GenerateKey(rand.Reader, 4096)
		require.NoError(t, err)
		err = types.ValidateRSAKeySize(&key.PublicKey)
		require.NoError(t, err)
	})
}

