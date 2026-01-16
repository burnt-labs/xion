// types/vkey.go
package types

import (
	"encoding/json"
	"fmt"

	"github.com/vocdoni/circom2gnark/parser"

	errorsmod "cosmossdk.io/errors"
)

func ValidateVKeyByteSize(data []byte, maxSizeBytes uint64) error {
	if maxSizeBytes > 0 && uint64(len(data)) > maxSizeBytes {
		return errorsmod.Wrapf(ErrVKeyTooLarge, "vkey size %d exceeds max %d", len(data), maxSizeBytes)
	}
	return nil
}

// ValidateVKeyBytes enforces that the vkey bytes represent a valid CircomVerificationKey JSON structure.
func ValidateVKeyBytes(data []byte, maxDecodedSize uint64) error {
	if err := ValidateVKeyByteSize(data, maxDecodedSize); err != nil {
		return err
	}
	// Validate by attempting to unmarshal
	vk, err := parser.UnmarshalCircomVerificationKeyJSON(data)
	if err != nil {
		return errorsmod.Wrapf(ErrInvalidVKey, "failed to unmarshal vkey bytes as circom vkey: %v", err)
	}

	// Validate the unmarshaled verification key
	if err := validateCircomVerificationKey(vk); err != nil {
		return errorsmod.Wrapf(ErrInvalidVKey, "invalid circom verification key: %v", err)
	}
	return nil
}

// validateCircomVerificationKey validates the structure and fields of a CircomVerificationKey
func validateCircomVerificationKey(vk *parser.CircomVerificationKey) error {
	// Validate protocol (only groth16 is supported for now)
	if vk.Protocol != "groth16" {
		return fmt.Errorf("unsupported protocol: %s (only 'groth16' is supported)", vk.Protocol)
	}

	// Validate NPublic (should be greater than 0)
	if vk.NPublic <= 0 {
		return fmt.Errorf("invalid nPublic: %d (must be greater than 0)", vk.NPublic)
	}

	// Validate VkAlpha1 (G1 point: affine form with 2 coordinates, extended affine with 3)
	// The parser only uses the first 2 coordinates, so we accept both formats
	if len(vk.VkAlpha1) < 2 {
		return fmt.Errorf("invalid VkAlpha1: expected at least 2 coordinates, got %d", len(vk.VkAlpha1))
	}

	// Validate VkBeta2 (G2 point: affine form with 2 rows, projective with 3 rows)
	// Both forms are acceptable, but we need at least 2 rows with 2 columns each
	if len(vk.VkBeta2) < 2 {
		return fmt.Errorf("invalid VkBeta2: expected at least 2 rows for G2 point, got %d", len(vk.VkBeta2))
	}
	// For projective form, we may have 3 rows (x, y, z), but only check the first 2 rows
	// as the parser only uses the first 2 rows
	for i := 0; i < 2 && i < len(vk.VkBeta2); i++ {
		if len(vk.VkBeta2[i]) != 2 {
			return fmt.Errorf("invalid VkBeta2: row %d should have 2 columns, got %d", i, len(vk.VkBeta2[i]))
		}
	}

	// Validate VkGamma2 (G2 point: same structure as VkBeta2)
	if len(vk.VkGamma2) < 2 {
		return fmt.Errorf("invalid VkGamma2: expected at least 2 rows for G2 point, got %d", len(vk.VkGamma2))
	}
	for i := 0; i < 2 && i < len(vk.VkGamma2); i++ {
		if len(vk.VkGamma2[i]) != 2 {
			return fmt.Errorf("invalid VkGamma2: row %d should have 2 columns, got %d", i, len(vk.VkGamma2[i]))
		}
	}

	// Validate VkDelta2 (G2 point: same structure as VkBeta2)
	if len(vk.VkDelta2) < 2 {
		return fmt.Errorf("invalid VkDelta2: expected at least 2 rows for G2 point, got %d", len(vk.VkDelta2))
	}
	for i := 0; i < 2 && i < len(vk.VkDelta2); i++ {
		if len(vk.VkDelta2[i]) != 2 {
			return fmt.Errorf("invalid VkDelta2: row %d should have 2 columns, got %d", i, len(vk.VkDelta2[i]))
		}
	}

	// Validate IC (array of G1 points for public inputs)
	// IC should have length = NPublic + 1 (constant term + each public input)
	expectedICLen := vk.NPublic + 1
	if len(vk.IC) != expectedICLen {
		return fmt.Errorf("invalid IC length: expected %d points (nPublic + 1), got %d", expectedICLen, len(vk.IC))
	}

	// Validate each IC point has at least 2 coordinates (affine or extended affine form)
	for i, icPoint := range vk.IC {
		if len(icPoint) < 2 {
			return fmt.Errorf("invalid IC[%d]: expected at least 2 coordinates, got %d", i, len(icPoint))
		}
	}

	return nil
}

// UnmarshalVKey unmarshals VKey.key_bytes into a parser.CircomVerificationKey
func UnmarshalVKey(vkey *VKey) (*parser.CircomVerificationKey, error) {
	if vkey == nil {
		return nil, fmt.Errorf("nil vkey")
	}

	if len(vkey.KeyBytes) == 0 {
		return nil, fmt.Errorf("empty key_bytes")
	}

	return parser.UnmarshalCircomVerificationKeyJSON(vkey.KeyBytes)
}

// MarshalVKey marshals a parser.CircomVerificationKey into bytes for storage
func MarshalVKey(vk *parser.CircomVerificationKey) ([]byte, error) {
	if vk == nil {
		return nil, fmt.Errorf("nil verification key")
	}

	return json.Marshal(vk)
}

// NewVKeyFromBytes creates a VKey from raw JSON bytes with validation
func NewVKeyFromBytes(keyBytes []byte, name, description string) (*VKey, error) {
	if err := ValidateVKeyBytes(keyBytes, 0); err != nil {
		return nil, err
	}

	return &VKey{
		KeyBytes:    keyBytes,
		Name:        name,
		Description: description,
	}, nil
}

// NewVKeyFromCircom creates a VKey from a parser.CircomVerificationKey
func NewVKeyFromCircom(vk *parser.CircomVerificationKey, name, description string) (*VKey, error) {
	keyBytes, err := MarshalVKey(vk)
	if err != nil {
		return nil, err
	}

	return &VKey{
		KeyBytes:    keyBytes,
		Name:        name,
		Description: description,
	}, nil
}
