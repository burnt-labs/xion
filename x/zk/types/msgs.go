package types

import (
	fmt "fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	_ sdk.Msg = &MsgAddVKey{}
	_ sdk.Msg = &MsgUpdateVKey{}
	_ sdk.Msg = &MsgRemoveVKey{}
)

// types/msgs.go

func (m *MsgAddVKey) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}

	if m.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	if len(m.VkeyBytes) == 0 {
		return fmt.Errorf("vkey_bytes cannot be empty")
	}

	// Validate using the parser library
	if err := ValidateVKeyBytes(m.VkeyBytes); err != nil {
		return fmt.Errorf("invalid vkey_bytes: %w", err)
	}

	return nil
}

// ValidateBasic performs basic validation on MsgUpdateVKey
func (m *MsgUpdateVKey) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}

	if m.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	if len(m.VkeyBytes) == 0 {
		return fmt.Errorf("vkey_bytes cannot be empty")
	}

	// Validate using the parser library
	if err := ValidateVKeyBytes(m.VkeyBytes); err != nil {
		return fmt.Errorf("invalid vkey_bytes: %w", err)
	}

	return nil
}

// ValidateBasic performs basic validation on MsgRemoveVKey
func (m *MsgRemoveVKey) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}

	if m.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	return nil
}
