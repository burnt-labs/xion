package types

import (
	"fmt"
)

// NewGenesisState creates a new GenesisState object
func NewGenesisState(vkeys []VKeyWithID) *GenesisState {
	return &GenesisState{
		Vkeys: vkeys,
	}
}

// DefaultGenesisState returns a default genesis state
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Vkeys: []VKeyWithID{},
	}
}

// Validate performs basic genesis state validation
func (gs GenesisState) Validate() error {
	// Check for duplicate vkey IDs
	seenIDs := make(map[uint64]bool)
	seenNames := make(map[string]bool)

	for i, vkeyWithID := range gs.Vkeys {
		// Check for duplicate IDs
		if seenIDs[vkeyWithID.Id] {
			return fmt.Errorf("duplicate vkey ID %d at index %d", vkeyWithID.Id, i)
		}
		seenIDs[vkeyWithID.Id] = true

		// Check for duplicate names
		if seenNames[vkeyWithID.Vkey.Name] {
			return fmt.Errorf("duplicate vkey name '%s' at index %d", vkeyWithID.Vkey.Name, i)
		}
		seenNames[vkeyWithID.Vkey.Name] = true

		// Validate the vkey itself
		if vkeyWithID.Vkey.Name == "" {
			return fmt.Errorf("vkey at index %d has empty name", i)
		}

		if len(vkeyWithID.Vkey.KeyBytes) == 0 {
			return fmt.Errorf("vkey '%s' at index %d has empty key_bytes", vkeyWithID.Vkey.Name, i)
		}

		// Validate the key bytes
		if err := ValidateVKeyBytes(vkeyWithID.Vkey.KeyBytes); err != nil {
			return fmt.Errorf("vkey '%s' at index %d has invalid key_bytes: %w", vkeyWithID.Vkey.Name, i, err)
		}
	}

	return nil
}
