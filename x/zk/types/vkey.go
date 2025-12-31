// types/vkey.go
package types

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"

	errorsmod "cosmossdk.io/errors"
	"github.com/vocdoni/circom2gnark/parser"
)

// ValidateVKeyBytes enforces base64 encoding, decodes, and validates the resulting verification key.
func ValidateVKeyBytes(data []byte) error {
	_, err := DecodeAndValidateVKeyBytes(data, DefaultMaxVKeySizeBytes)
	return err
}

// DecodeAndValidateVKeyBytes decodes base64 vkey bytes, enforces size/whitespace limits,
// and validates the resulting verification key JSON.
func DecodeAndValidateVKeyBytes(data []byte, maxDecodedSize uint64) ([]byte, error) {
	decoded, err := NormalizeVKeyBytes(data, maxDecodedSize)
	if err != nil {
		return nil, err
	}

	// Validate by attempting to unmarshal
	vk, err := parser.UnmarshalCircomVerificationKeyJSON(decoded)
	if err != nil {
		return nil, fmt.Errorf("invalid verification key JSON: %w", err)
	}

	// Validate the unmarshaled verification key
	if err := validateCircomVerificationKey(vk); err != nil {
		return nil, err
	}

	return decoded, nil
}

// NormalizeVKeyBytes decodes base64-encoded vkey bytes after size/whitespace checks.
func NormalizeVKeyBytes(data []byte, maxDecodedSize uint64) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty vkey data")
	}

	return decodeBase64VKeyBytes(data, maxDecodedSize)
}

func decodeBase64VKeyBytes(data []byte, maxDecodedSize uint64) ([]byte, error) {
	if err := rejectBase64VKeyWhitespace(data); err != nil {
		return nil, err
	}

	maxEncodedLen, err := maxBase64EncodedLen(maxDecodedSize)
	if err != nil {
		return nil, err
	}

	if maxDecodedSize > 0 && len(data) > maxEncodedLen {
		return nil, errorsmod.Wrapf(ErrVKeyTooLarge, "encoded vkey length %d exceeds max %d", len(data), maxEncodedLen)
	}

	decoded, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return nil, errorsmod.Wrap(ErrInvalidVKey, err.Error())
	}

	if maxDecodedSize > 0 && uint64(len(decoded)) > maxDecodedSize {
		return nil, errorsmod.Wrapf(ErrVKeyTooLarge, "decoded vkey size %d exceeds max %d", len(decoded), maxDecodedSize)
	}

	return decoded, nil
}

func rejectBase64VKeyWhitespace(data []byte) error {
	for _, b := range data {
		switch b {
		case ' ', '\n', '\r', '\t':
			return errorsmod.Wrap(ErrInvalidVKey, "base64 vkey contains whitespace")
		}
	}

	return nil
}

func maxBase64EncodedLen(maxDecodedSize uint64) (int, error) {
	if maxDecodedSize > math.MaxInt {
		return 0, errorsmod.Wrapf(ErrVKeyTooLarge, "max_vkey_size_bytes %d exceeds supported range", maxDecodedSize)
	}

	return base64.StdEncoding.EncodedLen(int(maxDecodedSize)), nil
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
	decoded, err := DecodeAndValidateVKeyBytes(keyBytes, DefaultMaxVKeySizeBytes)
	if err != nil {
		return nil, err
	}

	return &VKey{
		KeyBytes:    decoded,
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
