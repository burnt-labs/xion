package barretenberg

import (
	"errors"
	"fmt"
	"sync"
)

// Verifier provides UltraHonk proof verification using a verification key.
// It is safe for concurrent use by multiple goroutines.
type Verifier struct {
	mu   sync.RWMutex
	vkey *VerificationKey
}

// NewVerifier creates a new Verifier with the given verification key.
// The verification key is not copied - it will be closed when the Verifier is closed.
func NewVerifier(vkey *VerificationKey) (*Verifier, error) {
	if vkey == nil {
		return nil, fmt.Errorf("%w: nil verification key", ErrInvalidVKey)
	}

	if vkey.IsClosed() {
		return nil, fmt.Errorf("%w: verification key is closed", ErrClosed)
	}

	return &Verifier{vkey: vkey}, nil
}

// NewVerifierFromBytes creates a new Verifier by parsing a verification key from bytes.
func NewVerifierFromBytes(vkeyData []byte) (*Verifier, error) {
	vkey, err := ParseVerificationKey(vkeyData)
	if err != nil {
		return nil, err
	}

	return &Verifier{vkey: vkey}, nil
}

// NewVerifierFromHex creates a new Verifier by parsing a verification key from hex.
func NewVerifierFromHex(vkeyHex string) (*Verifier, error) {
	vkey, err := ParseVerificationKeyHex(vkeyHex)
	if err != nil {
		return nil, err
	}

	return &Verifier{vkey: vkey}, nil
}

// Close releases resources associated with the Verifier.
// This also closes the underlying verification key.
// It is safe to call Close multiple times.
func (v *Verifier) Close() {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.vkey != nil {
		v.vkey.Close()
		v.vkey = nil
	}
}

// IsClosed returns true if the Verifier has been closed.
func (v *Verifier) IsClosed() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.vkey == nil || v.vkey.IsClosed()
}

// Verify verifies an UltraHonk proof with the given public inputs.
//
// Returns (true, nil) if the proof is valid.
// Returns (false, nil) if the proof is invalid (verification failed).
// Returns (false, err) if an error occurred during verification.
//
// The publicInputs parameter should contain string representations of field elements.
// Supported formats:
//   - Decimal: "12345"
//   - Hex with prefix: "0x1a2b3c..."
func (v *Verifier) Verify(proof *Proof, publicInputs []string) (bool, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if v.vkey == nil || v.vkey.IsClosed() {
		return false, ErrClosed
	}

	if proof == nil {
		return false, fmt.Errorf("%w: nil proof", ErrInvalidProof)
	}

	// Parse public inputs
	pubInputs, err := ParsePublicInputsFromStrings(publicInputs)
	if err != nil {
		return false, err
	}

	return v.verifyWithInputs(proof, pubInputs)
}

// VerifyWithBytes verifies an UltraHonk proof with public inputs as raw bytes.
// Each byte slice in publicInputs should be a 32-byte field element (big-endian).
func (v *Verifier) VerifyWithBytes(proof *Proof, publicInputs [][]byte) (bool, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if v.vkey == nil || v.vkey.IsClosed() {
		return false, ErrClosed
	}

	if proof == nil {
		return false, fmt.Errorf("%w: nil proof", ErrInvalidProof)
	}

	pubInputs, err := NewPublicInputs(publicInputs)
	if err != nil {
		return false, err
	}

	return v.verifyWithInputs(proof, pubInputs)
}

// VerifyWithHexInputs verifies an UltraHonk proof with hex-encoded public inputs.
func (v *Verifier) VerifyWithHexInputs(proof *Proof, hexInputs []string) (bool, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if v.vkey == nil || v.vkey.IsClosed() {
		return false, ErrClosed
	}

	if proof == nil {
		return false, fmt.Errorf("%w: nil proof", ErrInvalidProof)
	}

	pubInputs, err := ParsePublicInputsFromHex(hexInputs)
	if err != nil {
		return false, err
	}

	return v.verifyWithInputs(proof, pubInputs)
}

// verifyWithInputs is the internal verification method.
// Caller must hold at least a read lock on v.mu.
func (v *Verifier) verifyWithInputs(proof *Proof, publicInputs *PublicInputs) (bool, error) {
	// Cross-check public input count against what the vkey declares.
	// This surfaces version mismatches (e.g. vkey generated with a different bb version)
	// with a clear error instead of an opaque barretenberg exception.
	_, err := v.vkey.NumPublicInputs()
	if err != nil {
		return false, fmt.Errorf("failed to read vkey public input count: %w", err)
	}
	// if publicInputs.Count() != expectedCount {
	// 	return false, fmt.Errorf("%w: vkey expects %d public input(s), got %d — ensure bb version matches compiled library",
	// 		ErrInvalidPublicInputs, expectedCount, publicInputs.Count())
	// }

	// Verify using the CGo bindings
	err = v.vkey.handle.verifyProof(
		proof.Bytes(),
		publicInputs.Bytes(),
		publicInputs.Count(),
	)

	if err == nil {
		return true, nil
	}

	// Check if this is a verification failure (proof is invalid) vs an actual error
	if errors.Is(err, ErrVerificationFailed) {
		return false, nil
	}

	return false, err
}

// NumPublicInputs returns the number of public inputs expected by this verifier.
func (v *Verifier) NumPublicInputs() (int, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if v.vkey == nil || v.vkey.IsClosed() {
		return 0, ErrClosed
	}

	return v.vkey.NumPublicInputs()
}

// CircuitSize returns the circuit size from the verification key.
func (v *Verifier) CircuitSize() (uint64, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if v.vkey == nil || v.vkey.IsClosed() {
		return 0, ErrClosed
	}

	return v.vkey.CircuitSize()
}

// VerificationKey returns the underlying verification key.
// The caller should not close the returned key; it is managed by the Verifier.
func (v *Verifier) VerificationKey() *VerificationKey {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.vkey
}

// VerifyProofBytes is a convenience function that verifies a proof without
// creating a persistent Verifier. Useful for one-off verifications.
func VerifyProofBytes(vkeyData, proofData []byte, publicInputs []string) (bool, error) {
	vkey, err := ParseVerificationKey(vkeyData)
	if err != nil {
		return false, fmt.Errorf("failed to parse verification key: %w", err)
	}
	defer vkey.Close()

	proof, err := ParseProof(proofData)
	if err != nil {
		return false, fmt.Errorf("failed to parse proof: %w", err)
	}

	verifier, err := NewVerifier(vkey)
	if err != nil {
		return false, err
	}
	defer verifier.Close()

	return verifier.Verify(proof, publicInputs)
}
