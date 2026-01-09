package types

import (
	"encoding/json"

	errorsmod "cosmossdk.io/errors"
)

const (
	DefaultMaxPubKeySizeBytes uint64 = 512
)

// DefaultParams returns default module parameters.
func DefaultParams() Params {
	vkeyIdentifier := uint64(1)

	return Params{
		VkeyIdentifier:     vkeyIdentifier,
		MaxPubkeySizeBytes: DefaultMaxPubKeySizeBytes,
	}
}

// Stringer method for Params.
func (p Params) String() string {
	bz, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}

	return string(bz)
}

// Validate does the sanity check on the params.
func (p Params) Validate() error {
	if p.MaxPubkeySizeBytes <= 0 {
		return errorsmod.Wrap(ErrInvalidParams, "max_pubkey_size_bytes must be positive")
	}
	return nil
}
