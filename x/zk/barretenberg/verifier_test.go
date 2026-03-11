//go:build cgo
// +build cgo

package barretenberg

import (
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

const (
	testdataDir = "testdata/statics"
	vkeyFile    = "vk"
	proofFile   = "proof"
	inputsFile  = "public_inputs"
)

// loadTestVector loads a test vector file from testdata directory.
func loadTestVector(t *testing.T, filename string) []byte {
	t.Helper()
	path := filepath.Join(testdataDir, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("test vector %s not found: %v", filename, err)
	}
	return data
}

// loadTestInputs loads public inputs from the binary public_inputs file.
func loadTestInputs(t *testing.T) [][]byte {
	t.Helper()
	data := loadTestVector(t, inputsFile)
	if len(data)%FieldElementSize != 0 {
		t.Fatalf("public_inputs file size %d is not a multiple of %d", len(data), FieldElementSize)
	}
	count := len(data) / FieldElementSize
	elements := make([][]byte, count)
	for i := range count {
		elements[i] = data[i*FieldElementSize : (i+1)*FieldElementSize]
	}
	return elements
}

// TestParseProofEmpty tests parsing empty proof.
func TestParseProofEmpty(t *testing.T) {
	_, err := ParseProof(nil)
	if err == nil {
		t.Error("expected error for nil data")
	}

	_, err = ParseProof([]byte{})
	if err == nil {
		t.Error("expected error for empty data")
	}
}

// TestParseProofTooSmall tests parsing proof that is too small.
func TestParseProofTooSmall(t *testing.T) {
	smallData := make([]byte, 100) // Less than minimum proof size
	_, err := ParseProof(smallData)
	if err == nil {
		t.Error("expected error for small proof")
	}

	if !errors.Is(err, ErrInvalidProof) {
		t.Errorf("expected ErrInvalidProof, got %v", err)
	}
}

// TestParseProofValid tests parsing valid proof structure.
func TestParseProofValid(t *testing.T) {
	// Create data that meets minimum size requirements
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i % 256)
	}

	proof, err := ParseProof(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if proof.Size() != len(data) {
		t.Errorf("proof size = %d, expected %d", proof.Size(), len(data))
	}

	// Check that hex encoding works
	hexStr := proof.Hex()
	decoded, err := hex.DecodeString(hexStr)
	if err != nil {
		t.Fatalf("failed to decode hex: %v", err)
	}

	if string(decoded) != string(data) {
		t.Error("round-trip through hex failed")
	}
}

// TestPublicInputsEmpty tests creating empty public inputs.
func TestPublicInputsEmpty(t *testing.T) {
	pi, err := NewPublicInputs(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pi.Count() != 0 {
		t.Errorf("expected 0 inputs, got %d", pi.Count())
	}

	if pi.Bytes() != nil {
		t.Error("expected nil bytes for empty inputs")
	}
}

// TestPublicInputsInvalidSize tests creating public inputs with wrong size.
func TestPublicInputsInvalidSize(t *testing.T) {
	// Element with wrong size
	elements := [][]byte{
		make([]byte, 16), // Should be 32 bytes
	}

	_, err := NewPublicInputs(elements)
	if err == nil {
		t.Error("expected error for invalid element size")
	}

	if !errors.Is(err, ErrInvalidPublicInputs) {
		t.Errorf("expected ErrInvalidPublicInputs, got %v", err)
	}
}

// TestPublicInputsValid tests creating valid public inputs.
func TestPublicInputsValid(t *testing.T) {
	elem1 := make([]byte, 32)
	elem1[31] = 0x42

	elem2 := make([]byte, 32)
	elem2[31] = 0x43

	pi, err := NewPublicInputs([][]byte{elem1, elem2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pi.Count() != 2 {
		t.Errorf("expected 2 inputs, got %d", pi.Count())
	}

	bytes := pi.Bytes()
	if len(bytes) != 64 {
		t.Errorf("expected 64 bytes, got %d", len(bytes))
	}

	// Check individual elements
	e, err := pi.Element(0)
	if err != nil {
		t.Fatalf("failed to get element 0: %v", err)
	}
	if e[31] != 0x42 {
		t.Errorf("element 0 last byte = %x, expected 0x42", e[31])
	}

	e, err = pi.Element(1)
	if err != nil {
		t.Fatalf("failed to get element 1: %v", err)
	}
	if e[31] != 0x43 {
		t.Errorf("element 1 last byte = %x, expected 0x43", e[31])
	}
}

// TestPublicInputsElementOutOfRange tests accessing out of range element.
func TestPublicInputsElementOutOfRange(t *testing.T) {
	elem := make([]byte, 32)
	pi, err := NewPublicInputs([][]byte{elem})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = pi.Element(1)
	if err == nil {
		t.Error("expected error for out of range index")
	}

	_, err = pi.Element(-1)
	if err == nil {
		t.Error("expected error for negative index")
	}
}

// TestParsePublicInputsFromStrings tests parsing public inputs from strings.
func TestParsePublicInputsFromStrings(t *testing.T) {
	tests := []struct {
		name    string
		inputs  []string
		wantErr bool
	}{
		{
			name:    "empty",
			inputs:  []string{},
			wantErr: false,
		},
		{
			name:    "decimal",
			inputs:  []string{"42"},
			wantErr: false,
		},
		{
			name:    "hex with prefix",
			inputs:  []string{"0x2a"},
			wantErr: false,
		},
		{
			name:    "multiple",
			inputs:  []string{"1", "2", "0x3"},
			wantErr: false,
		},
		{
			name:    "invalid string",
			inputs:  []string{"not a number"},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParsePublicInputsFromStrings(tc.inputs)
			if tc.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestNewVerifierNilVKey tests creating verifier with nil verification key.
func TestNewVerifierNilVKey(t *testing.T) {
	_, err := NewVerifier(nil)
	if err == nil {
		t.Error("expected error for nil vkey")
	}
}

// TestVerifierClosedOperations tests operations on closed verifier.
func TestVerifierClosedOperations(t *testing.T) {
	// Create a mock closed verifier
	v := &Verifier{vkey: nil}

	_, err := v.Verify(nil, nil)
	if err != ErrClosed {
		t.Errorf("Verify: expected ErrClosed, got %v", err)
	}

	_, err = v.VerifyWithBytes(nil, nil)
	if err != ErrClosed {
		t.Errorf("VerifyWithBytes: expected ErrClosed, got %v", err)
	}

	_, err = v.VerifyWithHexInputs(nil, nil)
	if err != ErrClosed {
		t.Errorf("VerifyWithHexInputs: expected ErrClosed, got %v", err)
	}

	_, err = v.NumPublicInputs()
	if err != ErrClosed {
		t.Errorf("NumPublicInputs: expected ErrClosed, got %v", err)
	}

	_, err = v.CircuitSize()
	if err != ErrClosed {
		t.Errorf("CircuitSize: expected ErrClosed, got %v", err)
	}
}

// TestVerifierIsClosed tests IsClosed method.
func TestVerifierIsClosed(t *testing.T) {
	v := &Verifier{vkey: nil}
	if !v.IsClosed() {
		t.Error("expected IsClosed() = true for nil vkey")
	}
}

// TestVerifierConcurrentUse tests thread safety of verifier.
func TestVerifierConcurrentUse(t *testing.T) {
	// Create a mock verifier
	v := &Verifier{vkey: nil}

	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 100 {
				_ = v.IsClosed()
				_, _ = v.Verify(nil, nil)
				_, _ = v.NumPublicInputs()
			}
		}()
	}
	wg.Wait()
}

// TestVerifyValidProof tests verification with valid test vectors.
// This test requires test vectors to be present in testdata/ AND the real barretenberg library.
// It is skipped with the stub library (which does not perform cryptographic verification).
func TestVerifyValidProof(t *testing.T) {
	if strings.HasPrefix(Version(), "stub") {
		t.Skip("stub library does not perform real verification; regenerate testdata with bb@4.0.4 and build real library")
	}
	vkeyData := loadTestVector(t, vkeyFile)
	proofData := loadTestVector(t, proofFile)
	inputs := loadTestInputs(t)
	// inputs := [][]byte{}

	vkey, err := ParseVerificationKey(vkeyData)
	if err != nil {
		t.Fatalf("failed to parse vkey: %v", err)
	}
	defer vkey.Close()

	proof, err := ParseProof(proofData)
	if err != nil {
		t.Fatalf("failed to parse proof: %v", err)
	}

	verifier, err := NewVerifier(vkey)
	if err != nil {
		t.Fatalf("failed to create verifier: %v", err)
	}
	defer verifier.Close()

	valid, err := verifier.VerifyWithBytes(proof, inputs)
	if err != nil {
		t.Fatalf("verification error: %v", err)
	}

	if !valid {
		t.Error("expected valid proof to verify")
	}
}

// TestVerifyInvalidProof tests verification with tampered proof.
func TestVerifyInvalidProof(t *testing.T) {
	if strings.HasPrefix(Version(), "stub") {
		t.Skip("stub library does not perform real verification")
	}

	vkeyData := loadTestVector(t, vkeyFile)
	proofData := loadTestVector(t, proofFile)

	// Tamper with proof data
	tamperedProof := make([]byte, len(proofData))
	copy(tamperedProof, proofData)
	if len(tamperedProof) > 100 {
		tamperedProof[100] ^= 0xFF // Flip some bits
	}

	vkey, err := ParseVerificationKey(vkeyData)
	if err != nil {
		t.Fatalf("failed to parse vkey: %v", err)
	}
	defer vkey.Close()

	proof, err := ParseProof(tamperedProof)
	if err != nil {
		t.Fatalf("failed to parse tampered proof: %v", err)
	}

	verifier, err := NewVerifier(vkey)
	if err != nil {
		t.Fatalf("failed to create verifier: %v", err)
	}
	defer verifier.Close()

	inputs := loadTestInputs(t)
	valid, err := verifier.VerifyWithBytes(proof, inputs)
	// Either verification fails (valid = false) or we get an error
	if valid {
		t.Error("expected tampered proof to fail verification")
	}
}

// BenchmarkVerify benchmarks proof verification.
func BenchmarkVerify(b *testing.B) {
	vkeyData, err := os.ReadFile(filepath.Join(testdataDir, vkeyFile))
	if err != nil {
		b.Skip("test vectors not available")
	}

	proofData, err := os.ReadFile(filepath.Join(testdataDir, proofFile))
	if err != nil {
		b.Skip("test vectors not available")
	}

	inputsData, err := os.ReadFile(filepath.Join(testdataDir, inputsFile))
	if err != nil {
		b.Skip("test vectors not available")
	}
	if len(inputsData)%FieldElementSize != 0 {
		b.Fatalf("public_inputs file size %d is not a multiple of %d", len(inputsData), FieldElementSize)
	}
	count := len(inputsData) / FieldElementSize
	inputs := make([][]byte, count)
	for i := range count {
		inputs[i] = inputsData[i*FieldElementSize : (i+1)*FieldElementSize]
	}

	vkey, err := ParseVerificationKey(vkeyData)
	if err != nil {
		b.Fatalf("failed to parse vkey: %v", err)
	}
	defer vkey.Close()

	proof, err := ParseProof(proofData)
	if err != nil {
		b.Fatalf("failed to parse proof: %v", err)
	}

	verifier, err := NewVerifier(vkey)
	if err != nil {
		b.Fatalf("failed to create verifier: %v", err)
	}
	defer verifier.Close()

	b.ResetTimer()

	for b.Loop() {
		_, _ = verifier.VerifyWithBytes(proof, inputs)
	}
}

// BenchmarkParseProof benchmarks proof parsing.
func BenchmarkParseProof(b *testing.B) {
	proofData, err := os.ReadFile(filepath.Join(testdataDir, proofFile))
	if err != nil {
		b.Skip("test vectors not available")
	}

	b.ResetTimer()

	for b.Loop() {
		_, err := ParseProof(proofData)
		if err != nil {
			b.Fatalf("failed to parse proof: %v", err)
		}
	}
}
