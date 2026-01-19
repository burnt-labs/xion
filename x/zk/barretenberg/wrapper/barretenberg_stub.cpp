/*
 * barretenberg_stub.cpp
 *
 * Stub implementation of the Barretenberg wrapper for development/testing.
 * This file provides a minimal implementation that compiles without
 * the full Barretenberg library.
 *
 * For production use, build with the real barretenberg_wrapper.cpp.
 */

#include "../include/barretenberg_wrapper.h"

#include <cstring>
#include <string>

// Thread-local error message storage
thread_local std::string g_last_error;

// Stub vkey structure
struct bb_vkey_impl {
    uint32_t num_public_inputs;
    uint64_t circuit_size;
    std::string data;
};

extern "C" {

bb_vkey_t* bb_vkey_from_bytes(const uint8_t* data, size_t len) {
    g_last_error.clear();

    if (data == nullptr || len == 0) {
        g_last_error = "null or empty verification key data";
        return nullptr;
    }

    // Stub: accept any data that's at least 1KB (like a real vkey would be)
    if (len < 1024) {
        g_last_error = "verification key too small (stub requires >= 1024 bytes)";
        return nullptr;
    }

    auto* impl = new (std::nothrow) bb_vkey_impl();
    if (impl == nullptr) {
        g_last_error = "failed to allocate verification key";
        return nullptr;
    }

    impl->num_public_inputs = 2;  // Default for stub
    impl->circuit_size = 65536;   // Default for stub
    impl->data = std::string(reinterpret_cast<const char*>(data), len);

    return reinterpret_cast<bb_vkey_t*>(impl);
}

void bb_vkey_free(bb_vkey_t* vkey) {
    if (vkey != nullptr) {
        auto* impl = reinterpret_cast<bb_vkey_impl*>(vkey);
        delete impl;
    }
}

int32_t bb_vkey_num_public_inputs(const bb_vkey_t* vkey) {
    g_last_error.clear();

    if (vkey == nullptr) {
        g_last_error = "null verification key";
        return -1;
    }

    auto* impl = reinterpret_cast<const bb_vkey_impl*>(vkey);
    return static_cast<int32_t>(impl->num_public_inputs);
}

uint64_t bb_vkey_circuit_size(const bb_vkey_t* vkey) {
    g_last_error.clear();

    if (vkey == nullptr) {
        g_last_error = "null verification key";
        return 0;
    }

    auto* impl = reinterpret_cast<const bb_vkey_impl*>(vkey);
    return impl->circuit_size;
}

bb_error_t bb_verify_proof(
    const bb_vkey_t* vkey,
    const uint8_t* proof, size_t proof_len,
    const uint8_t* public_inputs, size_t pub_len,
    size_t num_inputs
) {
    g_last_error.clear();

    if (vkey == nullptr) {
        g_last_error = "null verification key";
        return BB_ERR_NULL_POINTER;
    }

    if (proof == nullptr || proof_len == 0) {
        g_last_error = "null or empty proof";
        return BB_ERR_INVALID_PROOF;
    }

    // Stub: basic validation
    if (proof_len < 512) {
        g_last_error = "proof too small (stub requires >= 512 bytes)";
        return BB_ERR_INVALID_PROOF;
    }

    // Stub: always return success for now (real implementation would verify)
    // In a real scenario, this would call barretenberg's verifier
    return BB_SUCCESS;
}

const char* bb_get_last_error(void) {
    if (g_last_error.empty()) {
        return nullptr;
    }
    return g_last_error.c_str();
}

void bb_clear_last_error(void) {
    g_last_error.clear();
}

const char* bb_version(void) {
    return "stub-0.1.0";
}

int bb_supports_ultrahonk(void) {
    return 1;  // Stub claims to support UltraHonk
}

} // extern "C"
