package types

import (
	"errors"
	"fmt"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

// Parameter keys
var (
	KeyPlatformPercentage = []byte("platform-percentage")
)

var _ paramtypes.ParamSet = &Params{}

// ParamKeyTable returns the parameter key table.
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// DefaultParams returns default oracle parameters
func DefaultParams() Params {
	return Params{
		PlatformPercentage: 500,
	}
}

// ParamSetPairs returns the parameter set pairs.
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyPlatformPercentage, &p.PlatformPercentage, validatePlatformPercentage),
	}
}

// ValidateBasic performs basic validation on oracle parameters.
func (p *Params) ValidateBasic() error {
	if err := validatePlatformPercentage(p.PlatformPercentage); err != nil {
		return err
	}
	return nil
}

func validatePlatformPercentage(i interface{}) error {
	percentage, ok := i.(uint32)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if percentage > 10000 {
		return errors.New("platform fee can not exceed 100%")
	}

	return nil
}
