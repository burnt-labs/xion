/*
 * barretenberg_stub.c
 *
 * Pure-C stub implementation of the Barretenberg wrapper for CI/testing.
 * No C++ stdlib required — compiles with plain gcc, no -lc++ or -lstdc++ needed.
 *
 * For production use, build with the real barretenberg_wrapper.cpp.
 */

#include "../include/barretenberg_wrapper.h"

#include <stdlib.h>
#include <string.h>

/* Thread-local error message storage — plain C, no std::string */
#ifdef _MSC_VER
static __declspec(thread) char g_last_error[512];
#else
static _Thread_local char g_last_error[512];
#endif

/* Stub vkey structure — no C++ members */
typedef struct {
    uint32_t num_public_inputs;
    uint64_t circuit_size;
} bb_vkey_impl_t;

bb_vkey_t* bb_vkey_from_bytes(const uint8_t* data, size_t len) {
    g_last_error[0] = '\0';

    if (data == NULL || len == 0) {
        strncpy(g_last_error, "null or empty verification key data", sizeof(g_last_error) - 1);
        return NULL;
    }

    if (len < 1024) {
        strncpy(g_last_error, "verification key too small (stub requires >= 1024 bytes)", sizeof(g_last_error) - 1);
        return NULL;
    }

    bb_vkey_impl_t* impl = (bb_vkey_impl_t*)malloc(sizeof(bb_vkey_impl_t));
    if (impl == NULL) {
        strncpy(g_last_error, "failed to allocate verification key", sizeof(g_last_error) - 1);
        return NULL;
    }

    impl->num_public_inputs = 2; /* default for stub */
    impl->circuit_size = 65536;  /* default for stub */

    return (bb_vkey_t*)impl;
}

void bb_vkey_free(bb_vkey_t* vkey) {
    if (vkey != NULL) {
        free(vkey);
    }
}

int32_t bb_vkey_num_public_inputs(const bb_vkey_t* vkey) {
    g_last_error[0] = '\0';

    if (vkey == NULL) {
        strncpy(g_last_error, "null verification key", sizeof(g_last_error) - 1);
        return -1;
    }

    const bb_vkey_impl_t* impl = (const bb_vkey_impl_t*)vkey;
    return (int32_t)impl->num_public_inputs;
}

uint64_t bb_vkey_circuit_size(const bb_vkey_t* vkey) {
    g_last_error[0] = '\0';

    if (vkey == NULL) return 0;

    const bb_vkey_impl_t* impl = (const bb_vkey_impl_t*)vkey;
    return impl->circuit_size;
}

bb_error_t bb_verify_proof(
    const bb_vkey_t* vkey,
    const uint8_t* proof, size_t proof_len,
    const uint8_t* public_inputs, size_t pub_len,
    size_t num_inputs
) {
    g_last_error[0] = '\0';

    if (vkey == NULL) {
        strncpy(g_last_error, "null verification key", sizeof(g_last_error) - 1);
        return BB_ERR_NULL_POINTER;
    }

    if (proof == NULL || proof_len == 0) {
        strncpy(g_last_error, "null or empty proof", sizeof(g_last_error) - 1);
        return BB_ERR_INVALID_PROOF;
    }

    if (proof_len < 512) {
        strncpy(g_last_error, "proof too small (stub requires >= 512 bytes)", sizeof(g_last_error) - 1);
        return BB_ERR_INVALID_PROOF;
    }

    const bb_vkey_impl_t* impl = (const bb_vkey_impl_t*)vkey;
    if ((size_t)impl->num_public_inputs != num_inputs) {
        strncpy(g_last_error, "public input count mismatch", sizeof(g_last_error) - 1);
        return BB_ERR_INVALID_PUBLIC_INPUTS;
    }

    if (num_inputs > 0) {
        if (public_inputs == NULL) {
            strncpy(g_last_error, "null public inputs with non-zero count", sizeof(g_last_error) - 1);
            return BB_ERR_INVALID_PUBLIC_INPUTS;
        }
        if (pub_len != num_inputs * 32) {
            strncpy(g_last_error, "invalid public inputs length", sizeof(g_last_error) - 1);
            return BB_ERR_INVALID_PUBLIC_INPUTS;
        }
    }

    return BB_SUCCESS;
}

const char* bb_get_last_error(void) {
    return g_last_error[0] ? g_last_error : NULL;
}

void bb_clear_last_error(void) {
    g_last_error[0] = '\0';
}

const char* bb_version(void) {
    return "stub-0.1.0";
}

int bb_supports_ultrahonk(void) {
    return 1;
}
