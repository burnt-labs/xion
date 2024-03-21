package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"gopkg.in/yaml.v2"
)

var _ paramtypes.ParamSet = (*Params)(nil)

var (
	ParamStoreKeyTimeOffset    = []byte("TimeOffset")
	ParamStoreKeyDeploymentGas = []byte("DeploymentGas")
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams() Params {
	return Params{}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	params := NewParams()
	params.DeploymentGas = 10_000
	params.TimeOffset = 30 * 1000 // default to 30 seconds

	return params
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(ParamStoreKeyDeploymentGas, &p.DeploymentGas, validateDeploymentGas),
		paramtypes.NewParamSetPair(ParamStoreKeyTimeOffset, &p.TimeOffset, validateTimeOffset),
	}
}

func validateDeploymentGas(i interface{}) error {
	_, ok := i.(uint64)
	if !ok {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidType, "type: %T, expected uint64", i)
	}

	return nil
}

func validateTimeOffset(i interface{}) error {
	_, ok := i.(uint64)
	if !ok {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidType, "type: %T, expected uint64", i)
	}

	return nil
}

// Validate validates the set of params
func (p Params) Validate() error {
	return nil
}

// String implements the Stringer interface.
func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}
