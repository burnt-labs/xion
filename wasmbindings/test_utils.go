package wasmbinding

import (
	"crypto/rsa"
	"os"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/jwk"
)

func SetupKeys() (*rsa.PrivateKey, error) {
	// CreateAudience
	privateKeyBz, err := os.ReadFile("./keys/jwtRS256.key")
	if err != nil {
		return nil, err
	}
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyBz)
	if err != nil {
		return nil, err
	}
	return privateKey, nil
}

func SetupPublicKeys(rsaFile ...string) (*rsa.PrivateKey, jwk.Key, error) {
	// CreateAudience
	if rsaFile[0] == "" {
		rsaFile[0] = "./keys/jwtRS256.key"
	}
	privateKeyBz, err := os.ReadFile(rsaFile[0])
	if err != nil {
		return nil, nil, err
	}
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyBz)
	if err != nil {
		return nil, nil, err
	}
	jwkPrivKey, err := jwk.New(privateKey)
	if err != nil {
		return nil, nil, err
	}
	publicKey, err := jwkPrivKey.PublicKey()
	if err != nil {
		return nil, nil, err
	}
	publicKey.Set("alg", "RS256")

	return privateKey, publicKey, nil
}
