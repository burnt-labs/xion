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

// CanonicalizeRSAPublicKey returns a base64-encoded hash of the marshaled public key.
func CanonicalizeRSAPublicKey(pubKey *rsa.PublicKey) (string, error) {
	keyBz := x509.MarshalPKCS1PublicKey(pubKey)
	keyHashBz := sha256.Sum256(keyBz)

	return base64.StdEncoding.EncodeToString(keyHashBz[:]), nil
}
