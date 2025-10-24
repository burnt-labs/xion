// types/vkey.go
package types

import (
	"encoding/json"
	"fmt"

	"github.com/vocdoni/circom2gnark/parser"
)

// ValidateVKeyBytes validates that the bytes can be unmarshaled into a CircomVerificationKey
func ValidateVKeyBytes(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("empty vkey data")
	}

	// Validate by attempting to unmarshal
	_, err := parser.UnmarshalCircomVerificationKeyJSON(data)
	if err != nil {
		return fmt.Errorf("invalid verification key JSON: %w", err)
	}
	// TODO: verify circomverification key

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
	// Validate the bytes
	if err := ValidateVKeyBytes(keyBytes); err != nil {
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
