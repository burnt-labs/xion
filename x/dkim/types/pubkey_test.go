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
