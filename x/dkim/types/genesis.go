package types

import errorsmod "cosmossdk.io/errors"

// this line is used by starport scaffolding # genesis/types/import

// DefaultIndex is the default global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		// this line is used by starport scaffolding # genesis/types/default
		Params: DefaultParams(),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// this line is used by starport scaffolding # genesis/types/validate
	if err := gs.Params.Validate(); err != nil {
		return errorsmod.Wrap(err, "params")
	}
	for _, dkimPubKey := range gs.DkimPubkeys {
		if err := dkimPubKey.Validate(); err != nil {
			return err
		}
	}
	return nil
}
