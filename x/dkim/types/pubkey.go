package types

import (
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"

	"cosmossdk.io/errors"
)

// ParseRSAPublicKey parses PKIX or PKCS#1-encoded RSA public key bytes.
func ParseRSAPublicKey(pubKeyBytes []byte) (*rsa.PublicKey, error) {
	pub, err := x509.ParsePKIXPublicKey(pubKeyBytes)
	if err == nil {
		rsaPub, ok := pub.(*rsa.PublicKey)
		if !ok {
			return nil, ErrNotRSAKey
		}
		return rsaPub, nil
	}

	rsaPub, err := x509.ParsePKCS1PublicKey(pubKeyBytes)
	if err != nil {
		return nil, errors.Wrapf(ErrInvalidPubKey, "failed to parse public key: %s", err)
	}

	return rsaPub, nil
}

// CanonicalizeRSAPublicKey returns a canonical, base64-encoded hash of the given RSA
// public key.
//
// The function:
//   - Marshals the RSA public key using PKCS#1 via x509.MarshalPKCS1PublicKey to obtain
//     a stable, canonical byte representation of the key, independent of the original
//     input encoding (for example, PKIX vs PKCS#1).
//   - Hashes those bytes using SHA-256.
//   - Encodes the resulting 32-byte SHA-256 digest using standard base64 encoding.
//
// This canonical base64-encoded SHA-256(PKCS#1(pubkey)) identifier is used for
// operations such as revocation tracking and duplicate detection, ensuring that the
// same logical RSA key always maps to the same identifier even if provided in
// different encodings.
func CanonicalizeRSAPublicKey(pubKey *rsa.PublicKey) (string, error) {
	keyBz := x509.MarshalPKCS1PublicKey(pubKey)
	keyHashBz := sha256.Sum256(keyBz)

	return base64.StdEncoding.EncodeToString(keyHashBz[:]), nil
}
