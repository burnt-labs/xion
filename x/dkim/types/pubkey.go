package types

import (
	"crypto/rsa"
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

// CanonicalizeRSAPublicKey returns base64-encoded PKIX and PKCS#1 encodings.
func CanonicalizeRSAPublicKey(pubKey *rsa.PublicKey) (string, string, error) {
	pkixDER, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return "", "", errors.Wrapf(ErrInvalidPubKey, "failed to marshal public key: %s", err)
	}

	pkcs1DER := x509.MarshalPKCS1PublicKey(pubKey)

	return base64.StdEncoding.EncodeToString(pkixDER), base64.StdEncoding.EncodeToString(pkcs1DER), nil
}
