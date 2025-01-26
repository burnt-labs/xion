package app

import (
	"encoding/json"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/module"
)

/*
The GenesisState of the blockchain is represented as a map of raw JSON
messages keyed by an identifier string.

This identifier determines to which module the genesis information belongs,
ensuring it is correctly routed during chain initialization.
Within this application, default genesis information is retrieved from the
ModuleBasicManager, which populates JSON from each BasicModule object
provided during initialization.
*/
type GenesisState map[string]json.RawMessage

// NewDefaultGenesisState generates the default state for the application.
func NewDefaultGenesisState(cdc codec.JSONCodec, manager module.BasicManager) GenesisState {
	return manager.DefaultGenesis(cdc)
}
