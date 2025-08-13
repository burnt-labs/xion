package types

import (
	"encoding/json"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"
)

// NewGenesisState - Create a new genesis state
func NewGenesisState(params Params) *GenesisState {
	return &GenesisState{
		Params: params,
	}
}

// DefaultGenesisState - Return a default genesis state
func DefaultGenesisState() *GenesisState {
	return NewGenesisState(DefaultParams())
}

// GetGenesisStateFromAppState returns x/globalfee GenesisState given raw application
// genesis state.
func GetGenesisStateFromAppState(cdc codec.Codec, appState map[string]json.RawMessage) *GenesisState {
	var genesisState GenesisState

	if appState[ModuleName] != nil {
		cdc.MustUnmarshalJSON(appState[ModuleName], &genesisState)
		return &genesisState
	}

	// If no genesis state is provided, return the default genesis state
	return DefaultGenesisState()
}

func ValidateGenesis(data GenesisState) error {
	if err := data.Params.ValidateBasic(); err != nil {
		return errorsmod.Wrap(err, "globalfee params")
	}

	return nil
}
