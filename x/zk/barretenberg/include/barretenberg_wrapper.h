/*
 * barretenberg_wrapper.h
 *
 * C API wrapper for Barretenberg's UltraHonk verification.
 * This header provides a C-compatible interface for use with CGo.
 */

#ifndef BARRETENBERG_WRAPPER_H
#define BARRETENBERG_WRAPPER_H

#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

/* Error codes returned by bb_ functions */
typedef enum {
    BB_SUCCESS = 0,
    BB_ERR_INVALID_VKEY = 1,
    BB_ERR_INVALID_PROOF = 2,
    BB_ERR_INVALID_PUBLIC_INPUTS = 3,
    BB_ERR_VERIFICATION_FAILED = 4,
    BB_ERR_INTERNAL = 5,
    BB_ERR_NULL_POINTER = 6,
    BB_ERR_ALLOCATION_FAILED = 7,
    BB_ERR_DESERIALIZATION_FAILED = 8
} bb_error_t;

/* Opaque handle to a verification key */
typedef struct bb_vkey_t bb_vkey_t;

/*
 * Parse a verification key from binary data.
 *
 * @param data      Pointer to the binary verification key data
 * @param len       Length of the data in bytes
 * @return          Pointer to the verification key handle, or NULL on failure
 *
 * On failure, use bb_get_last_error() to retrieve the error message.
 * The caller must free the returned handle using bb_vkey_free().
 */
bb_vkey_t* bb_vkey_from_bytes(const uint8_t* data, size_t len);

/*
 * Free a verification key handle.
 *
 * @param vkey      Pointer to the verification key handle (may be NULL)
 */
void bb_vkey_free(bb_vkey_t* vkey);

/*
 * Get the number of public inputs expected by the verification key.
 *
 * @param vkey      Pointer to the verification key handle
 * @return          Number of public inputs, or -1 on error
 */
int32_t bb_vkey_num_public_inputs(const bb_vkey_t* vkey);

/*
 * Get the circuit size from the verification key.
 *
 * @param vkey      Pointer to the verification key handle
 * @return          Circuit size, or 0 on error
 */
uint64_t bb_vkey_circuit_size(const bb_vkey_t* vkey);

/*
 * Verify an UltraHonk proof.
 *
 * @param vkey          Pointer to the verification key handle
 * @param proof         Pointer to the proof data
 * @param proof_len     Length of the proof data in bytes
 * @param public_inputs Pointer to serialized public inputs (32-byte field elements concatenated)
 * @param pub_len       Total length of public inputs data in bytes
 * @param num_inputs    Number of public input field elements
 * @return              BB_SUCCESS if verification succeeds, error code otherwise
 *
 * Note: public_inputs should contain num_inputs field elements, each serialized
 * as a 32-byte big-endian value. The total length pub_len should equal
 * num_inputs * 32.
 */
bb_error_t bb_verify_proof(
    const bb_vkey_t* vkey,
    const uint8_t* proof, size_t proof_len,
    const uint8_t* public_inputs, size_t pub_len,
    size_t num_inputs
);

/*
 * Get the last error message from a failed operation.
 *
 * @return          Pointer to a null-terminated error string, or NULL if no error
 *
 * The returned string is valid until the next bb_ function call on the same thread.
 * The caller must NOT free the returned string.
 */
const char* bb_get_last_error(void);

/*
 * Clear the last error message for the current thread.
 */
void bb_clear_last_error(void);

/*
 * Get the Barretenberg library version string.
 *
 * @return          Pointer to a null-terminated version string
 *
 * The returned string is statically allocated and must NOT be freed.
 */
const char* bb_version(void);

/*
 * Check if the library was built with UltraHonk support.
 *
 * @return          1 if UltraHonk is supported, 0 otherwise
 */
int bb_supports_ultrahonk(void);

#ifdef __cplusplus
}
#endif

#endif /* BARRETENBERG_WRAPPER_H */
