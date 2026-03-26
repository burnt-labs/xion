package types

// DefaultGenesisState returns the default genesis state for the tee module.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{}
}

// Validate performs basic genesis state validation. The tee module is stateless.
func (gs GenesisState) Validate() error {
	return nil
}
