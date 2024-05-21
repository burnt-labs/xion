package wasmbinding

import (
	"crypto/rsa"
	"os"

	"github.com/lestrrat-go/jwx/v2/jwk"
)

func SetupKeys() (*rsa.PrivateKey, error) {
	// CreateAudience
	privateKeyBz, err := os.ReadFile("./keys/jwtRS256.key")
	if err != nil {
		return nil, err
	}
	jwKey, err := jwk.ParseKey(privateKeyBz, jwk.WithPEM(true))
	if err != nil {
		return nil, err
	}
	var privateKey rsa.PrivateKey
	err = jwKey.Raw(&privateKey)
	if err != nil {
		return nil, err
	}
	return &privateKey, nil
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
	jwKey, err := jwk.ParseKey(privateKeyBz, jwk.WithPEM(true))
	if err != nil {
		return nil, nil, err
	}

	var privateKey rsa.PrivateKey
	err = jwKey.Raw(&privateKey)
	if err != nil {
		return nil, nil, err
	}

	return &privateKey, nil, nil
}
