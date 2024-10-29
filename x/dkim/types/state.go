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
		return errors.Wrap(sdkError.ErrInvalidRequest, err.Error())
	}
	// make sure the public key is base64 encoded
	if _, err := base64.StdEncoding.DecodeString(pubKey.PubKey); err != nil {
		return errors.Wrap(sdkError.ErrInvalidRequest, err.Error())
	}
	return nil
}
