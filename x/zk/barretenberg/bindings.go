package barretenberg

/*
#cgo CFLAGS: -I${SRCDIR}/include
#include "barretenberg_wrapper.h"
#include <stdlib.h>
*/
import "C"

import (
	"runtime"
	"sync"
	"unsafe"
)

// vkeyHandle wraps the C verification key pointer with thread-safe access.
type vkeyHandle struct {
	mu     sync.RWMutex
	ptr    *C.bb_vkey_t
	closed bool
}

// newVKeyHandle creates a new vkeyHandle from raw bytes.
// The caller is responsible for calling Close() when done.
func newVKeyHandle(data []byte) (*vkeyHandle, error) {
	if len(data) == 0 {
		return nil, ErrInvalidVKey
	}

	// Pin the data slice to prevent GC from moving it during the C call
	var pinner runtime.Pinner
	pinner.Pin(&data[0])
	defer pinner.Unpin()

	ptr := C.bb_vkey_from_bytes(
		(*C.uint8_t)(unsafe.Pointer(&data[0])),
		C.size_t(len(data)),
	)

	if ptr == nil {
		return nil, getLastError(errCodeInvalidVKey)
	}

	h := &vkeyHandle{ptr: ptr}

	// Set finalizer to clean up if Close() is not called
	runtime.SetFinalizer(h, func(h *vkeyHandle) {
		h.Close()
	})

	return h, nil
}

// Close releases the verification key resources.
func (h *vkeyHandle) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed || h.ptr == nil {
		return
	}

	C.bb_vkey_free(h.ptr)
	h.ptr = nil
	h.closed = true

	// Remove finalizer since we've cleaned up
	runtime.SetFinalizer(h, nil)
}

// numPublicInputs returns the number of public inputs expected by the verification key.
func (h *vkeyHandle) numPublicInputs() (int, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.closed || h.ptr == nil {
		return 0, ErrClosed
	}

	n := C.bb_vkey_num_public_inputs(h.ptr)
	if n < 0 {
		return 0, getLastError(errCodeInvalidVKey)
	}

	return int(n), nil
}

// circuitSize returns the circuit size from the verification key.
func (h *vkeyHandle) circuitSize() (uint64, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.closed || h.ptr == nil {
		return 0, ErrClosed
	}

	size := C.bb_vkey_circuit_size(h.ptr)
	return uint64(size), nil
}

// verifyProof verifies a proof against this verification key.
// publicInputs should be serialized field elements (32 bytes each, big-endian).
func (h *vkeyHandle) verifyProof(proof, publicInputs []byte, numInputs int) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.closed || h.ptr == nil {
		return ErrClosed
	}

	if len(proof) == 0 {
		return ErrInvalidProof
	}

	// Validate public inputs size
	expectedLen := numInputs * 32
	if numInputs > 0 && len(publicInputs) != expectedLen {
		return ErrInvalidPublicInputs
	}

	// Pin memory to prevent GC from moving it during the C call
	var pinner runtime.Pinner
	pinner.Pin(&proof[0])
	defer pinner.Unpin()

	var pubPtr *C.uint8_t
	if numInputs > 0 && len(publicInputs) > 0 {
		pinner.Pin(&publicInputs[0])
		pubPtr = (*C.uint8_t)(unsafe.Pointer(&publicInputs[0]))
	}

	result := C.bb_verify_proof(
		h.ptr,
		(*C.uint8_t)(unsafe.Pointer(&proof[0])),
		C.size_t(len(proof)),
		pubPtr,
		C.size_t(len(publicInputs)),
		C.size_t(numInputs),
	)

	if result != C.BB_SUCCESS {
		return getLastError(int(result))
	}

	return nil
}

// getLastError retrieves the last error message from the C library.
func getLastError(code int) error {
	errStr := C.bb_get_last_error()
	C.bb_clear_last_error()

	var detail string
	if errStr != nil {
		detail = C.GoString(errStr)
	}

	return errorFromCode(code, detail)
}

// Version returns the Barretenberg library version string.
func Version() string {
	return C.GoString(C.bb_version())
}

// SupportsUltraHonk returns true if the library supports UltraHonk proofs.
func SupportsUltraHonk() bool {
	return C.bb_supports_ultrahonk() != 0
}
