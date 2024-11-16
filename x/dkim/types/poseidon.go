package types

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"

	"github.com/iden3/go-iden3-crypto/poseidon"

	"cosmossdk.io/errors"

	sdkError "github.com/cosmos/cosmos-sdk/types/errors"
)

// PubkeyHasher emulates the PubkeyHasher logic from Circom
func PreparePubkeyForHashing(pubkey []*big.Int, n, k int) []*big.Int {
	// Step 1: Calculate k2_chunked_size
	k2ChunkedSize := k >> 1
	if k%2 == 1 {
		k2ChunkedSize++
	}

	// Step 2: Prepare pubkey_hash_input with correct size
	pubkeyHashInput := make([]*big.Int, k2ChunkedSize)

	// Step 3: Populate pubkey_hash_input based on conditions
	for i := 0; i < k2ChunkedSize; i++ {
		if i == k2ChunkedSize-1 && k2ChunkedSize%2 == 1 {
			// Last element, odd case: only one element at this index
			pubkeyHashInput[i] = pubkey[2*i]
		} else {
			// Combine two elements from pubkey with shift
			term1 := new(big.Int).Set(pubkey[2*i])
			term2 := new(big.Int).Lsh(pubkey[2*i+1], uint(n)) //nolint:gosec // disable G115
			pubkeyHashInput[i] = new(big.Int).Add(term1, term2)
		}
	}

	return pubkeyHashInput
}

const (
	CircomBigintN = 121
	CircomBigintK = 17
)

func BigIntToChunkedBytes(num *big.Int, bytesPerChunk, numChunks int) []*big.Int {
	res := make([]*big.Int, 0, numChunks)

	// Define the mask as (1 << (8 * bytesPerChunk)) - 1
	msk := new(big.Int).Lsh(big.NewInt(1), uint(bytesPerChunk)) //nolint:gosec // disable G115
	msk.Sub(msk, big.NewInt(1))

	for i := 0; i < numChunks; i++ {
		// Shift the number to the right by i * bytesPerChunk * 8 bits
		shifted := new(big.Int).Rsh(num, uint(i*bytesPerChunk)) //nolint:gosec // disable G115
		// Mask to get the chunk and convert to string
		chunk := new(big.Int).And(shifted, msk)
		res = append(res, chunk)
	}

	return res
}

func ConvertStringArrayToBigInt(arr []string) ([]*big.Int, error) {
	res := make([]*big.Int, len(arr))

	for i := 0; i < len(arr); i++ {
		val, isSet := new(big.Int).SetString(arr[i], 10)
		if !isSet {
			return nil, errors.Wrap(sdkError.ErrInvalidRequest, "failed to set big.Int")
		}
		res[i] = val
	}
	return res, nil
}

// Converts a base64 encoded string `pk` into a PEM format public key
func FormatPublicKey(pk string) string {
	// Determine the necessary padding for base64 encoding
	pad := (4 - (len(pk) % 4)) % 4
	pkPadded := pk + string(bytes.Repeat([]byte("="), pad))

	// Insert newline every 64 characters
	var formattedPk bytes.Buffer
	for i := 0; i < len(pkPadded); i += 64 {
		end := i + 64
		if end > len(pkPadded) {
			end = len(pkPadded)
		}
		formattedPk.WriteString(pkPadded[i:end] + "\n")
	}

	// Wrap in PEM format
	pemKey := fmt.Sprintf("-----BEGIN PUBLIC KEY-----\n%s-----END PUBLIC KEY-----\n", formattedPk.String())
	return pemKey
}

// compute the poseidon hash of a x509 encoded public key
func ComputePoseidonHash(pub string) (*big.Int, error) {
	// make sure the public key is base64 encoded
	if _, err := base64.StdEncoding.DecodeString(pub); err != nil {
		return nil, errors.Wrap(sdkError.ErrInvalidRequest, err.Error())
	}
	// write pubkey to PEM
	pemKey := FormatPublicKey(pub)
	block, _ := pem.Decode([]byte(pemKey))
	if block == nil {
		return nil, errors.Wrap(sdkError.ErrInvalidRequest, "public key is invalid")
	}
	pkInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	publicKey, ok := pkInterface.(*rsa.PublicKey)
	if !ok {
		return nil, errors.Wrap(sdkError.ErrInvalidRequest, "the public key is not a rsa.PublicKey")
	}
	modulus := publicKey.N
	// convert modulus to circom bigint bytes
	modulusBytes := BigIntToChunkedBytes(modulus, CircomBigintN, CircomBigintK)
	// prepare the pubkey for hashing
	pubKeyInputBigInt := PreparePubkeyForHashing(modulusBytes, CircomBigintN, CircomBigintK)
	hash, err := poseidon.Hash(pubKeyInputBigInt)
	if err != nil {
		return nil, errors.Wrap(sdkError.ErrInvalidRequest, err.Error())
	}
	return hash, nil
}
