//go:build cgo
// +build cgo

package barretenberg

import (
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

const (
	vkeyTestdataDir = "testdata/statics"
	vkeyTestFile    = "vk"
)

// loadVkeyTestVector loads a vkey test vector from testdata; skips test if missing.
func loadVkeyTestVector(t *testing.T, filename string) []byte {
	t.Helper()
	path := filepath.Join(vkeyTestdataDir, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("test vector %s not found: %v", filename, err)
	}
	return data
}

// TestParseVerificationKeyEmpty tests parsing empty verification key.
func TestParseVerificationKeyEmpty(t *testing.T) {
	_, err := ParseVerificationKey(nil)
	if err == nil {
		t.Error("expected error for nil data")
	}

	_, err = ParseVerificationKey([]byte{})
	if err == nil {
		t.Error("expected error for empty data")
	}
}

// TestParseVerificationKeyInvalid tests parsing invalid verification key.
func TestParseVerificationKeyInvalid(t *testing.T) {
	invalidData := []byte("this is not a valid verification key")
	_, err := ParseVerificationKey(invalidData)
	if err == nil {
		t.Error("expected error for invalid data")
	}

	if !errors.Is(err, ErrInvalidVKey) {
		t.Errorf("expected ErrInvalidVKey, got %v", err)
	}
}

// TestParseVerificationKeyHexInvalid tests parsing invalid hex string.
func TestParseVerificationKeyHexInvalid(t *testing.T) {
	_, err := ParseVerificationKeyHex("not valid hex!")
	if err == nil {
		t.Error("expected error for invalid hex")
	}
}

// TestValidateVerificationKeyBytesEmpty tests validation with empty data.
func TestValidateVerificationKeyBytesEmpty(t *testing.T) {
	err := ValidateVerificationKeyBytes(nil, 0)
	if err == nil {
		t.Error("expected error for nil data")
	}

	err = ValidateVerificationKeyBytes([]byte{}, 0)
	if err == nil {
		t.Error("expected error for empty data")
	}
}

// TestValidateVerificationKeyBytesTooSmall tests validation with small data.
func TestValidateVerificationKeyBytesTooSmall(t *testing.T) {
	smallData := make([]byte, 100)
	err := ValidateVerificationKeyBytes(smallData, 0)
	if err == nil {
		t.Error("expected error for small data")
	}
}

// TestValidateVerificationKeyBytesMaxSizeExceeded tests that validation fails when size exceeds max.
func TestValidateVerificationKeyBytesMaxSizeExceeded(t *testing.T) {
	// Use testdata vkey; if missing, skip. We only need something > 100 bytes.
	vkeyData := loadVkeyTestVector(t, vkeyTestFile)
	if len(vkeyData) <= 100 {
		t.Skip("testdata vkey too small for this test")
	}
	err := ValidateVerificationKeyBytes(vkeyData, 100)
	if err == nil {
		t.Error("expected error when max size is smaller than vkey size")
	}
}

// TestValidateVerificationKeyBytesFromTestdata checks that the vkey file in testdata
// passes ValidateVerificationKeyBytes. Requires test vectors in testdata/statics/.
func TestValidateVerificationKeyBytesFromTestdata(t *testing.T) {
	vkeyData := loadVkeyTestVector(t, vkeyTestFile)

	err := ValidateVerificationKeyBytes(vkeyData, 0)
	if err != nil {
		t.Fatalf("testdata vkey should pass validation: %v", err)
	}

	const maxSize = 256 * 1024 // 256 KiB
	err = ValidateVerificationKeyBytes(vkeyData, maxSize)
	if err != nil {
		t.Fatalf("testdata vkey should pass validation with max size %d: %v", maxSize, err)
	}
}

// TestVerificationKey_Bytes_roundTrip tests that Bytes() returns the original data.
func TestVerificationKey_Bytes_roundTrip(t *testing.T) {
	vkeyData := loadVkeyTestVector(t, vkeyTestFile)
	vk, err := ParseVerificationKey(vkeyData)
	if err != nil {
		t.Fatalf("parse vkey: %v", err)
	}
	defer vk.Close()

	got := vk.Bytes()
	if len(got) != len(vkeyData) {
		t.Errorf("Bytes() length = %d, want %d", len(got), len(vkeyData))
	}
	for i := range vkeyData {
		if got[i] != vkeyData[i] {
			t.Errorf("Bytes()[%d] = %d, want %d", i, got[i], vkeyData[i])
		}
	}
}

// TestVerificationKey_Hex_roundTrip tests Hex() and ParseVerificationKeyHex round-trip.
func TestVerificationKey_Hex_roundTrip(t *testing.T) {
	vkeyData := loadVkeyTestVector(t, vkeyTestFile)
	vk, err := ParseVerificationKey(vkeyData)
	if err != nil {
		t.Fatalf("parse vkey: %v", err)
	}
	defer vk.Close()

	hexStr := vk.Hex()
	decoded, err := hex.DecodeString(hexStr)
	if err != nil {
		t.Fatalf("decode hex: %v", err)
	}
	if len(decoded) != len(vkeyData) {
		t.Errorf("round-trip length = %d, want %d", len(decoded), len(vkeyData))
	}
	for i := range vkeyData {
		if decoded[i] != vkeyData[i] {
			t.Errorf("round-trip[%d] = %d, want %d", i, decoded[i], vkeyData[i])
		}
	}

	// ParseVerificationKeyHex(hexStr) should yield equivalent vkey
	vk2, err := ParseVerificationKeyHex(hexStr)
	if err != nil {
		t.Fatalf("ParseVerificationKeyHex: %v", err)
	}
	defer vk2.Close()
	if len(vk2.Bytes()) != len(vkeyData) {
		t.Errorf("ParseVerificationKeyHex Bytes() length = %d, want %d", len(vk2.Bytes()), len(vkeyData))
	}
}

// TestVerificationKey_Close_idempotent tests that Close can be called multiple times.
func TestVerificationKey_Close_idempotent(t *testing.T) {
	vkeyData := loadVkeyTestVector(t, vkeyTestFile)
	vk, err := ParseVerificationKey(vkeyData)
	if err != nil {
		t.Fatalf("parse vkey: %v", err)
	}
	vk.Close()
	vk.Close()
	vk.Close()
	// IsClosed should be true and methods should return ErrClosed
	if !vk.IsClosed() {
		t.Error("expected IsClosed() = true after Close")
	}
	_, err = vk.NumPublicInputs()
	if !errors.Is(err, ErrClosed) {
		t.Errorf("NumPublicInputs after Close: want ErrClosed, got %v", err)
	}
	_, err = vk.CircuitSize()
	if !errors.Is(err, ErrClosed) {
		t.Errorf("CircuitSize after Close: want ErrClosed, got %v", err)
	}
}

// TestVerificationKey_IsClosed_beforeAndAfter tests IsClosed before and after Close.
func TestVerificationKey_IsClosed_beforeAndAfter(t *testing.T) {
	vkeyData := loadVkeyTestVector(t, vkeyTestFile)
	vk, err := ParseVerificationKey(vkeyData)
	if err != nil {
		t.Fatalf("parse vkey: %v", err)
	}
	if vk.IsClosed() {
		t.Error("expected IsClosed() = false before Close")
	}
	vk.Close()
	if !vk.IsClosed() {
		t.Error("expected IsClosed() = true after Close")
	}
}

// TestVerificationKey_NumPublicInputs_CircuitSize_fromTestdata tests NumPublicInputs and CircuitSize with testdata vkey.
func TestVerificationKey_NumPublicInputs_CircuitSize_fromTestdata(t *testing.T) {
	vkeyData := loadVkeyTestVector(t, vkeyTestFile)
	vk, err := ParseVerificationKey(vkeyData)
	if err != nil {
		t.Fatalf("parse vkey: %v", err)
	}
	defer vk.Close()

	nPub, err := vk.NumPublicInputs()
	if err != nil {
		t.Fatalf("NumPublicInputs: %v", err)
	}
	if nPub < 0 {
		t.Errorf("NumPublicInputs = %d, want >= 0", nPub)
	}

	circuitSize, err := vk.CircuitSize()
	if err != nil {
		t.Fatalf("CircuitSize: %v", err)
	}
	if circuitSize < MinCircuitSize {
		t.Errorf("CircuitSize = %d, want >= %d", circuitSize, MinCircuitSize)
	}
}

// BenchmarkParseVerificationKey benchmarks verification key parsing.
func BenchmarkParseVerificationKey(b *testing.B) {
	vkeyData, err := os.ReadFile(filepath.Join(vkeyTestdataDir, vkeyTestFile))
	if err != nil {
		b.Skip("test vectors not available")
	}

	b.ResetTimer()

	for b.Loop() {
		vkey, err := ParseVerificationKey(vkeyData)
		if err != nil {
			b.Fatalf("failed to parse vkey: %v", err)
		}
		vkey.Close()
	}
}
