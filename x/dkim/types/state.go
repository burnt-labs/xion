package types

import (
	"net/url"

	"cosmossdk.io/errors"
	sdkError "github.com/cosmos/cosmos-sdk/types/errors"
)

// ValidateBasic does a sanity check on the provided data.
func (pubKey *DkimPubKey) Validate() error {
	// url pass the pubkey domain
	if _, err := url.ParseRequestURI(pubKey.Domain); err != nil {
		return errors.Wrap(sdkError.ErrInvalidRequest, err.Error())
	}
	// TODO: pass the public key - what kind of public key can be passed here?
	return nil
}
