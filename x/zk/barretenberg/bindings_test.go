//go:build cgo
// +build cgo

package barretenberg

import (
	"runtime"
	"sync"
	"testing"
)

// TestVersion tests that the library version is returned correctly.
func TestVersion(t *testing.T) {
	version := Version()
	if version == "" {
		t.Error("Version() returned empty string")
	}
	t.Logf("Barretenberg version: %s", version)
}

// TestSupportsUltraHonk tests that UltraHonk support is reported correctly.
func TestSupportsUltraHonk(t *testing.T) {
	if !SupportsUltraHonk() {
		t.Error("SupportsUltraHonk() returned false, expected true")
	}
}

// TestVKeyHandleNilData tests that creating a vkey handle with nil data fails.
func TestVKeyHandleNilData(t *testing.T) {
	_, err := newVKeyHandle(nil)
	if err == nil {
		t.Error("expected error for nil data, got nil")
	}
}

// TestVKeyHandleEmptyData tests that creating a vkey handle with empty data fails.
func TestVKeyHandleEmptyData(t *testing.T) {
	_, err := newVKeyHandle([]byte{})
	if err == nil {
		t.Error("expected error for empty data, got nil")
	}
}

// TestVKeyHandleInvalidData tests that creating a vkey handle with invalid data fails.
func TestVKeyHandleInvalidData(t *testing.T) {
	invalidData := []byte("this is not a valid verification key")
	_, err := newVKeyHandle(invalidData)
	if err == nil {
		t.Error("expected error for invalid data, got nil")
	}
}

// TestVKeyHandleClose tests that closing a vkey handle works correctly.
func TestVKeyHandleClose(t *testing.T) {
	// This test uses invalid data, but we're testing the Close behavior
	// which should work even if the handle was never valid
	h := &vkeyHandle{}
	h.Close() // Should not panic

	// Close again should also not panic
	h.Close()
}

// TestVKeyHandleClosedOperations tests that operations on a closed handle fail.
func TestVKeyHandleClosedOperations(t *testing.T) {
	h := &vkeyHandle{closed: true}

	_, err := h.numPublicInputs()
	if err != ErrClosed {
		t.Errorf("expected ErrClosed, got %v", err)
	}

	_, err = h.circuitSize()
	if err != ErrClosed {
		t.Errorf("expected ErrClosed, got %v", err)
	}

	err = h.verifyProof([]byte{0x00}, nil, 0)
	if err != ErrClosed {
		t.Errorf("expected ErrClosed, got %v", err)
	}
}

// TestVKeyHandleVerifyNilPtr tests that verifying with nil handle ptr fails.
func TestVKeyHandleVerifyNilPtr(t *testing.T) {
	h := &vkeyHandle{} // Not closed, but no valid ptr

	// A handle with nil ptr should return ErrClosed
	err := h.verifyProof([]byte{0x00}, nil, 0)
	if err != ErrClosed {
		t.Errorf("expected ErrClosed for nil ptr handle, got %v", err)
	}
}

// TestMemoryCleanup tests that multiple alloc/free cycles don't leak memory.
// This test is run with the race detector to help catch issues.
func TestMemoryCleanup(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping memory test in short mode")
	}

	// We can't test with valid data without the library built,
	// but we can test that invalid data doesn't cause leaks
	invalidData := []byte("not a valid vkey")

	for i := 0; i < 1000; i++ {
		_, _ = newVKeyHandle(invalidData)
	}

	// Force GC to run finalizers
	runtime.GC()
	runtime.GC()
}

// TestConcurrentVKeyOperations tests that vkey operations are thread-safe.
func TestConcurrentVKeyOperations(t *testing.T) {
	h := &vkeyHandle{closed: true} // Use closed handle for safe testing

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_, _ = h.numPublicInputs()
				_, _ = h.circuitSize()
				_ = h.verifyProof([]byte{0x00}, nil, 0)
			}
		}()
	}
	wg.Wait()
}

// TestErrorFromCode tests error code conversion.
func TestErrorFromCode(t *testing.T) {
	tests := []struct {
		code     int
		expected error
	}{
		{errCodeSuccess, nil},
		{errCodeInvalidVKey, ErrInvalidVKey},
		{errCodeInvalidProof, ErrInvalidProof},
		{errCodeInvalidPublicInputs, ErrInvalidPublicInputs},
		{errCodeVerificationFailed, ErrVerificationFailed},
		{errCodeNullPointer, ErrNullPointer},
		{errCodeAllocationFailed, ErrAllocationFailed},
		{errCodeDeserializationFailed, ErrDeserializationFailed},
		{999, ErrInternal}, // Unknown code
	}

	for _, tc := range tests {
		result := errorFromCode(tc.code, "")
		if result != tc.expected {
			t.Errorf("errorFromCode(%d) = %v, expected %v", tc.code, result, tc.expected)
		}
	}
}

// TestErrorFromCodeWithDetail tests error code conversion with detail string.
func TestErrorFromCodeWithDetail(t *testing.T) {
	err := errorFromCode(errCodeInvalidVKey, "test detail")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	expectedMsg := "barretenberg: invalid verification key: test detail"
	if err.Error() != expectedMsg {
		t.Errorf("error message = %q, expected %q", err.Error(), expectedMsg)
	}
}

// TestWrappedErrorUnwrap tests that wrapped errors can be unwrapped.
func TestWrappedErrorUnwrap(t *testing.T) {
	err := errorFromCode(errCodeInvalidProof, "some detail")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	we, ok := err.(*wrappedError)
	if !ok {
		t.Fatalf("expected *wrappedError, got %T", err)
	}

	unwrapped := we.Unwrap()
	if unwrapped != ErrInvalidProof {
		t.Errorf("Unwrap() = %v, expected %v", unwrapped, ErrInvalidProof)
	}
}
