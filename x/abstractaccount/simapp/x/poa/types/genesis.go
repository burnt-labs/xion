package types

import (
	abci "github.com/cometbft/cometbft/abci/types"
)

func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Validators: []abci.ValidatorUpdate{},
	}
}

func (gs GenesisState) Validate() error {
	if len(gs.Validators) == 0 {
		return ErrEmptyValidatorSet
	}

	return nil
}
