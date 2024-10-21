package types

import (
	"encoding/json"
	"errors"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types"
)

// Validate performs basic validation of supply genesis data returning an
// error for any failed validation criteria.
func (gs GenesisState) Validate() error {
	if gs.PlatformPercentage > 10000 {
		return errors.New("unable to set platform percentage to greater than 100%")
	}

	return nil
}

// NewGenesisState creates a new genesis state.
func NewGenesisState(platformPercentage uint32, platformMinimums types.Coins) *GenesisState {
	rv := &GenesisState{
		PlatformPercentage: platformPercentage,
		PlatformMinimums:   platformMinimums,
	}
	return rv
}

// DefaultGenesisState returns a default bank module genesis state.
func DefaultGenesisState() *GenesisState {
	return NewGenesisState(0, types.Coins{})
}

// GetGenesisStateFromAppState returns x/bank GenesisState given raw application
// genesis state.
func GetGenesisStateFromAppState(cdc codec.JSONCodec, appState map[string]json.RawMessage) *GenesisState {
	var genesisState GenesisState

	if appState[ModuleName] != nil {
		cdc.MustUnmarshalJSON(appState[ModuleName], &genesisState)
	}

	return &genesisState
}
