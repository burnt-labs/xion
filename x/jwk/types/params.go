package types

import (
	errorsmod "cosmossdk.io/errors"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
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
func NewParams(timeOffset, deploymentGas uint64) Params {
	return Params{
		TimeOffset:    timeOffset,
		DeploymentGas: deploymentGas,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	deploymentGas := uint64(10_000)
	timeOffset := uint64(30 * 1000) // default to 30 seconds

	return NewParams(timeOffset, deploymentGas)
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
		return errorsmod.Wrapf(sdkerrors.ErrInvalidType, "type: %T, expected uint64", i)
	}

	return nil
}

func validateTimeOffset(i interface{}) error {
	_, ok := i.(uint64)
	if !ok {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidType, "type: %T, expected uint64", i)
	}

	return nil
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := validateDeploymentGas(p.DeploymentGas); err != nil {
		return err
	}

	return validateTimeOffset(p.TimeOffset)
}
