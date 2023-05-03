package types

import (
	"errors"
)

// NewParams creates a new parameter configuration for the xion module
func NewParams() Params {
	return Params{
		PlatformPercentage: 0,
	}
}

// DefaultParams is the default parameter configuration for the bank module
func DefaultParams() Params {
	return Params{
		PlatformPercentage: 500,
	}
}

// Validate all bank module parameters
func (p Params) Validate() error {
	if p.PlatformPercentage > 10000 {
		return errors.New("platform fee cannot exceed 100%")
	}
	return nil
}
