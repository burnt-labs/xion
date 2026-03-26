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

// ProofSystemGroth16 and ProofSystemUltraHonk are typed aliases for the ProofSystem enum.
const (
	ProofSystemGroth16   = ProofSystem_PROOF_SYSTEM_GROTH16
	ProofSystemUltraHonk = ProofSystem_PROOF_SYSTEM_ULTRA_HONK_ZK
)

const (
	// MaxVKeyNameLen is the maximum allowed length (in bytes) for a verification key name.
	MaxVKeyNameLen = 128
	// MaxVKeyDescLen is the maximum allowed length (in bytes) for a verification key description.
	MaxVKeyDescLen = 1024
)

func (m *MsgAddVKey) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}

	if m.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	if len(m.Name) > MaxVKeyNameLen {
		return fmt.Errorf("name length %d bytes exceeds maximum %d bytes", len(m.Name), MaxVKeyNameLen)
	}

	if len(m.Description) > MaxVKeyDescLen {
		return fmt.Errorf("description length %d bytes exceeds maximum %d bytes", len(m.Description), MaxVKeyDescLen)
	}

	if len(m.VkeyBytes) == 0 {
		return fmt.Errorf("vkey_bytes cannot be empty")
	}

	proofSystem := m.GetProofSystem()
	if proofSystem != ProofSystem_PROOF_SYSTEM_UNSPECIFIED &&
		proofSystem != ProofSystem_PROOF_SYSTEM_GROTH16 &&
		proofSystem != ProofSystem_PROOF_SYSTEM_ULTRA_HONK_ZK {
		return fmt.Errorf("unsupported proof_system: %v", proofSystem)
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

	if len(m.Name) > MaxVKeyNameLen {
		return fmt.Errorf("name length %d bytes exceeds maximum %d bytes", len(m.Name), MaxVKeyNameLen)
	}

	if len(m.Description) > MaxVKeyDescLen {
		return fmt.Errorf("description length %d bytes exceeds maximum %d bytes", len(m.Description), MaxVKeyDescLen)
	}

	if len(m.VkeyBytes) == 0 {
		return fmt.Errorf("vkey_bytes cannot be empty")
	}

	proofSystem := m.GetProofSystem()
	if proofSystem != ProofSystem_PROOF_SYSTEM_UNSPECIFIED &&
		proofSystem != ProofSystem_PROOF_SYSTEM_GROTH16 &&
		proofSystem != ProofSystem_PROOF_SYSTEM_ULTRA_HONK_ZK {
		return fmt.Errorf("unsupported proof_system: %v", proofSystem)
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

	// Backfill newly-added Groth16 and UltraHonk size params for older clients that don't specify them.
	return m.Params.WithMaxLimitDefaults().Validate()
}

// GetSigners returns the expected signers for a MsgUpdateParams message.
func (m *MsgUpdateParams) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{addr}
}
