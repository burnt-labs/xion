package types

import (
	fmt "fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	_ sdk.Msg = &MsgAddVKey{}
	_ sdk.Msg = &MsgUpdateVKey{}
	_ sdk.Msg = &MsgRemoveVKey{}
	_ sdk.Msg = &MsgUpdateParams{}
)

// types/msgs.go

// ProofSystemGroth16 and ProofSystemUltraHonk are the allowed proof_system values.
const (
	ProofSystemGroth16   = "groth16"
	ProofSystemUltraHonk = "ultrahonk"
)

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

	proofSystem := m.GetProofSystem()
	if proofSystem != "" && proofSystem != ProofSystemGroth16 && proofSystem != ProofSystemUltraHonk {
		return fmt.Errorf("proof_system must be %q or %q", ProofSystemGroth16, ProofSystemUltraHonk)
	}
	if err := ValidateVKeyForProofSystem(m.VkeyBytes, DefaultMaxVKeySizeBytes, proofSystem); err != nil {
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

	proofSystem := m.GetProofSystem()
	if proofSystem != "" && proofSystem != ProofSystemGroth16 && proofSystem != ProofSystemUltraHonk {
		return fmt.Errorf("proof_system must be %q or %q", ProofSystemGroth16, ProofSystemUltraHonk)
	}
	if err := ValidateVKeyForProofSystem(m.VkeyBytes, DefaultMaxVKeySizeBytes, proofSystem); err != nil {
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

// ValidateBasic performs basic validation on MsgUpdateParams.
func (m *MsgUpdateParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}

	return m.Params.Validate()
}

// GetSigners returns the expected signers for a MsgUpdateParams message.
func (m *MsgUpdateParams) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{addr}
}
