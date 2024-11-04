package types

import (
	"encoding/base64"
	"math/big"
	"net/url"

	"cosmossdk.io/errors"

	sdkError "github.com/cosmos/cosmos-sdk/types/errors"
)

// ValidateBasic does a sanity check on the provided data.
func (pubKey *DkimPubKey) Validate() error {
	// url pass the pubkey domain
	if _, err := url.Parse(pubKey.Domain); err != nil {
		return errors.Wrap(sdkError.ErrInvalidRequest, err.Error())
	}
	// make sure the public key is base64 encoded
	if _, err := base64.StdEncoding.DecodeString(pubKey.PubKey); err != nil {
		return errors.Wrap(sdkError.ErrInvalidRequest, err.Error())
	}
	// validate that the poseidon hash
	expectedHash, err := ComputePoseidonHash(pubKey.PubKey)
	if err != nil {
		return err
	}
	hash := new(big.Int).SetBytes(pubKey.PoseidonHash)
	if hash.Cmp(expectedHash) != 0 {
		return errors.Wrap(sdkError.ErrInvalidRequest, "poseidon hash does not match")
	}
	return nil
}
