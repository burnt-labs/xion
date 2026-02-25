package barretenberg

import (
	"encoding/hex"
	"fmt"
)

// Validation limits for Barretenberg verification keys (heuristics to reject
// obviously invalid or abusive vkeys without relying only on native parse).
const (
	// MinVKeySizeBytes is a heuristic minimum; smaller payloads are rejected
	// before calling into the native library (avoids C round-trip for garbage).
	MinVKeySizeBytes = 256
	// MinCircuitSize is the minimum circuit size; native returns 0 on error.
	MinCircuitSize = 1
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

// ValidateVerificationKeyBytes validates verification key bytes so they are
// appropriate and usable as an UltraHonk vkey. It enforces: non-empty data,
// optional max size, minimum size heuristic, successful native parse, and
// post-parse semantic checks (num public inputs and circuit size).
// If maxSizeBytes is > 0, len(data) must not exceed it (same as Groth16's
// ValidateVKeyByteSize / ValidateVKeyBytes).
func ValidateVerificationKeyBytes(data []byte, maxSizeBytes uint64) error {
	if len(data) == 0 {
		return fmt.Errorf("%w: empty data", ErrInvalidVKey)
	}
	if maxSizeBytes > 0 && uint64(len(data)) > maxSizeBytes {
		return fmt.Errorf("%w: vkey size %d exceeds max %d", ErrInvalidVKey, len(data), maxSizeBytes)
	}
	if len(data) < MinVKeySizeBytes {
		return fmt.Errorf("%w: data too small (%d bytes, minimum %d)", ErrInvalidVKey, len(data), MinVKeySizeBytes)
	}

	vk, err := ParseVerificationKey(data)
	if err != nil {
		return err
	}
	defer vk.Close()

	nPub, err := vk.NumPublicInputs()
	if err != nil {
		return fmt.Errorf("%w: num public inputs: %v", ErrInvalidVKey, err)
	}
	if nPub < 0 {
		return fmt.Errorf("%w: invalid num public inputs: %d", ErrInvalidVKey, nPub)
	}

	circuitSize, err := vk.CircuitSize()
	if err != nil {
		return fmt.Errorf("%w: circuit size: %v", ErrInvalidVKey, err)
	}
	if circuitSize < MinCircuitSize {
		return fmt.Errorf("%w: invalid circuit size: %d (minimum %d)", ErrInvalidVKey, circuitSize, MinCircuitSize)
	}

	return nil
}
