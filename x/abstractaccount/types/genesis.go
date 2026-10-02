package types

import (
	"encoding/binary"
	"errors"
	"fmt"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func NewGenesisState(nextAccountID uint64, params *Params) *GenesisState {
	return &GenesisState{
		NextAccountId: nextAccountID,
		Params:        params,
	}
}

func DefaultGenesisState() *GenesisState {
	return NewGenesisState(1, DefaultParams())
}

func (gs *GenesisState) Validate() error {
	if gs.Params == nil {
		return errors.New("params cannot be nil")
	}
	if err := gs.Params.Validate(); err != nil {
		return err
	}
	if len(gs.AccountAddresses) > 0 && !gs.Params.RegistrationConfigured() {
		return errors.New("account address registry requires configured address derivation")
	}
	return validateAccountAddresses(gs.AccountAddresses)
}

func validateAccountAddresses(entries []*AccountAddress) error {
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if entry == nil {
			return errors.New("account address registry entry cannot be nil")
		}
		sender, err := sdk.AccAddressFromBech32(entry.Sender)
		if err != nil {
			return fmt.Errorf("invalid account address registry sender: %w", err)
		}
		if _, err := sdk.AccAddressFromBech32(entry.Address); err != nil {
			return fmt.Errorf("invalid account address registry address: %w", err)
		}
		if err := wasmtypes.ValidateSalt(entry.Salt); err != nil {
			return fmt.Errorf("invalid account address registry salt: %w", err)
		}

		keyBytes := make([]byte, 8+len(sender)+len(entry.Salt))
		binary.BigEndian.PutUint64(keyBytes[:8], uint64(len(sender)))
		copy(keyBytes[8:], sender)
		copy(keyBytes[8+len(sender):], entry.Salt)
		key := string(keyBytes)
		if _, ok := seen[key]; ok {
			return errors.New("duplicate account address registry entry")
		}
		seen[key] = struct{}{}
	}

	return nil
}
