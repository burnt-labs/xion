package barretenberg

import (
	"encoding/hex"
	"fmt"
)

// VerificationKey represents an UltraHonk verification key.
// It wraps the native Barretenberg verification key with a safe Go interface.
type VerificationKey struct {
	handle *vkeyHandle
	raw    []byte // Original bytes for serialization
}

// ParseVerificationKey parses a verification key from binary data.
// The data should be in the Barretenberg UltraHonk verification key format.
//
// The caller must call Close() when done with the verification key to release
// native resources. Alternatively, the key will be cleaned up by the garbage
// collector, but this may delay resource release.
func ParseVerificationKey(data []byte) (*VerificationKey, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("%w: empty data", ErrInvalidVKey)
	}

	handle, err := newVKeyHandle(data)
	if err != nil {
		return nil, err
	}

	// Make a copy of the data for potential re-serialization
	rawCopy := make([]byte, len(data))
	copy(rawCopy, data)

	return &VerificationKey{
		handle: handle,
		raw:    rawCopy,
	}, nil
}

// ParseVerificationKeyHex parses a verification key from a hex-encoded string.
func ParseVerificationKeyHex(hexStr string) (*VerificationKey, error) {
	data, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid hex encoding: %v", ErrInvalidVKey, err)
	}
	return ParseVerificationKey(data)
}

// Close releases native resources associated with the verification key.
// After Close is called, the verification key cannot be used.
// It is safe to call Close multiple times.
func (vk *VerificationKey) Close() {
	if vk.handle != nil {
		vk.handle.Close()
	}
}

// NumPublicInputs returns the number of public inputs expected by this
// verification key.
func (vk *VerificationKey) NumPublicInputs() (int, error) {
	if vk.handle == nil {
		return 0, ErrClosed
	}
	return vk.handle.numPublicInputs()
}

// CircuitSize returns the circuit size from the verification key.
func (vk *VerificationKey) CircuitSize() (uint64, error) {
	if vk.handle == nil {
		return 0, ErrClosed
	}
	return vk.handle.circuitSize()
}

// Bytes returns the raw verification key bytes.
// The returned slice should not be modified.
func (vk *VerificationKey) Bytes() []byte {
	return vk.raw
}

// Hex returns the verification key as a hex-encoded string.
func (vk *VerificationKey) Hex() string {
	return hex.EncodeToString(vk.raw)
}

// IsClosed returns true if the verification key has been closed.
func (vk *VerificationKey) IsClosed() bool {
	if vk.handle == nil {
		return true
	}
	vk.handle.mu.RLock()
	defer vk.handle.mu.RUnlock()
	return vk.handle.closed
}

// ValidateVerificationKeyBytes performs basic validation on verification key bytes
// without fully parsing them. This is useful for quick validation before storage.
func ValidateVerificationKeyBytes(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("%w: empty data", ErrInvalidVKey)
	}

	// Minimum reasonable size for an UltraHonk verification key
	// This is a heuristic based on the expected structure
	const minVKeySize = 1024
	if len(data) < minVKeySize {
		return fmt.Errorf("%w: data too small (%d bytes, minimum %d)", ErrInvalidVKey, len(data), minVKeySize)
	}

	// Try to parse to validate fully
	vk, err := ParseVerificationKey(data)
	if err != nil {
		return err
	}
	vk.Close()

	return nil
}
