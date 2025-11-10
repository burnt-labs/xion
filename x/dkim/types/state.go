package types

import (
	"encoding/base64"
	"net/url"

	"cosmossdk.io/errors"

	sdkError "github.com/cosmos/cosmos-sdk/types/errors"
)

// ValidateBasic does a sanity check on the provided data.
func (pubKey *DkimPubKey) Validate() error {
	// url pass the pubkey domain
	if _, err := url.Parse(pubKey.Domain); err != nil {
		return errors.Wrap(sdkError.ErrInvalidRequest, "dkim url key parsing failed "+err.Error())
	}
	// make sure the public key is base64 encoded
	if _, err := base64.StdEncoding.DecodeString(pubKey.PubKey); err != nil {
		return errors.Wrap(sdkError.ErrInvalidRequest, "dkim public key decoding failed "+err.Error())
	}
	/*
		// validate that the poseidon hash
		expectedHash, err := ComputePoseidonHash(pubKey.PubKey)
		if err != nil {
			return err
		}
		hash, isSet := new(big.Int).SetString(string(pubKey.PoseidonHash), 10)
		if !isSet {
			return errors.Wrap(sdkError.ErrInvalidRequest, "failed to set poseidon hash")
		}
		if hash.Cmp(expectedHash) != 0 {
			return errors.Wrapf(sdkError.ErrInvalidRequest, "poseidon hash does not match ")
		}
	*/
	return nil
}
