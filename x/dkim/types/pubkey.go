package types

import (
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"

	errorsmod "cosmossdk.io/errors"
)

// MinRSAKeyBits is the hardcoded fallback minimum RSA key size used in
// ValidateDkimPubKeysWithRevocation when params.MinRsaKeyBits is unset (zero).
// The governance-configurable minimum is params.MinRsaKeyBits (default 1024,
// defined as DefaultMinRSAKeyBits in params.go).
const MinRSAKeyBits = 2048

// ParseRSAPublicKey parses PKIX or PKCS#1-encoded RSA public key bytes.
// It does NOT enforce a minimum key size — use ValidateRSAKeySize for that.
func ParseRSAPublicKey(pubKeyBytes []byte) (*rsa.PublicKey, error) {
	var rsaPub *rsa.PublicKey

	pub, err := x509.ParsePKIXPublicKey(pubKeyBytes)
	if err == nil {
		key, ok := pub.(*rsa.PublicKey)
		if !ok {
			return nil, ErrNotRSAKey
		}
		rsaPub = key
	} else {
		key, err := x509.ParsePKCS1PublicKey(pubKeyBytes)
		if err != nil {
			return nil, errorsmod.Wrapf(ErrInvalidPubKey, "failed to parse public key: %s", err)
		}
		rsaPub = key
	}

	return rsaPub, nil
}

// ValidateRSAKeySize checks that the RSA key meets the hardcoded minimum bit length
// (MinRSAKeyBits = 2048). This is the fallback used by ValidateDkimPubKeysWithRevocation
// when params.MinRsaKeyBits is unset. It is NOT called from any ValidateBasic path.
func ValidateRSAKeySize(key *rsa.PublicKey) error {
	if key == nil || key.N == nil {
		return errorsmod.Wrap(ErrInvalidPubKey, "RSA public key is nil")
	}
	if key.N.BitLen() < MinRSAKeyBits {
		return errorsmod.Wrapf(ErrInvalidPubKey, "RSA key size %d bits is below minimum %d", key.N.BitLen(), MinRSAKeyBits)
	}
	return nil
}

// CanonicalizeRSAPublicKey returns a canonical, base64-encoded hash of the given RSA
// public key.
//
// The function:
//   - Marshals the RSA public key using PKCS#1 via x509.MarshalPKCS1PublicKey to obtain
//     a stable, canonical byte representation of the key, independent of the original
//     input encoding (for example, PKIX vs PKCS#1)
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
