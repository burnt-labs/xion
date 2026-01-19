/*
 * barretenberg_wrapper.cpp
 *
 * C++ implementation of the Barretenberg wrapper for UltraHonk verification.
 */

#include "../include/barretenberg_wrapper.h"

#include <cstring>
#include <memory>
#include <mutex>
#include <sstream>
#include <string>
#include <vector>

// Barretenberg headers
#include "barretenberg/honk/proof_system/types/proof.hpp"
#include "barretenberg/stdlib_circuit_builders/ultra_flavor.hpp"
#include "barretenberg/ultra_honk/ultra_verifier.hpp"

// Version information
#ifndef BB_VERSION
#define BB_VERSION "0.82.0"
#endif

namespace {

// Thread-local error message storage
thread_local std::string g_last_error;

void set_last_error(const std::string& msg) {
    g_last_error = msg;
}

void clear_last_error() {
    g_last_error.clear();
}

// Wrapper struct for verification key
struct bb_vkey_impl {
    std::shared_ptr<bb::UltraFlavor::VerificationKey> vk;
    uint32_t num_public_inputs;
    uint64_t circuit_size;
};

} // anonymous namespace

extern "C" {

bb_vkey_t* bb_vkey_from_bytes(const uint8_t* data, size_t len) {
    clear_last_error();

    if (data == nullptr || len == 0) {
        set_last_error("null or empty verification key data");
        return nullptr;
    }

    try {
        // Create a vector from the input data
        std::vector<uint8_t> vk_bytes(data, data + len);

        // Deserialize the verification key
        // UltraHonk verification keys are serialized in msgpack format
        auto vk = std::make_shared<bb::UltraFlavor::VerificationKey>();

        // Use barretenberg's deserialization
        // The verification key should be deserialized from the binary format
        auto it = vk_bytes.begin();
        bb::serialize::read(it, *vk);

        // Allocate the wrapper struct
        auto* impl = new (std::nothrow) bb_vkey_impl();
        if (impl == nullptr) {
            set_last_error("failed to allocate verification key wrapper");
            return nullptr;
        }

        impl->vk = vk;
        impl->num_public_inputs = vk->num_public_inputs;
        impl->circuit_size = vk->circuit_size;

        return reinterpret_cast<bb_vkey_t*>(impl);

    } catch (const std::exception& e) {
        std::ostringstream oss;
        oss << "failed to parse verification key: " << e.what();
        set_last_error(oss.str());
        return nullptr;
    } catch (...) {
        set_last_error("unknown error parsing verification key");
        return nullptr;
    }
}

void bb_vkey_free(bb_vkey_t* vkey) {
    if (vkey != nullptr) {
        auto* impl = reinterpret_cast<bb_vkey_impl*>(vkey);
        delete impl;
    }
}

int32_t bb_vkey_num_public_inputs(const bb_vkey_t* vkey) {
    clear_last_error();

    if (vkey == nullptr) {
        set_last_error("null verification key");
        return -1;
    }

    auto* impl = reinterpret_cast<const bb_vkey_impl*>(vkey);
    return static_cast<int32_t>(impl->num_public_inputs);
}

uint64_t bb_vkey_circuit_size(const bb_vkey_t* vkey) {
    clear_last_error();

    if (vkey == nullptr) {
        set_last_error("null verification key");
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
    clear_last_error();

    // Validate inputs
    if (vkey == nullptr) {
        set_last_error("null verification key");
        return BB_ERR_NULL_POINTER;
    }

    if (proof == nullptr || proof_len == 0) {
        set_last_error("null or empty proof data");
        return BB_ERR_INVALID_PROOF;
    }

    // Validate public inputs size
    constexpr size_t FIELD_SIZE = 32; // 32 bytes per field element
    size_t expected_pub_len = num_inputs * FIELD_SIZE;
    if (num_inputs > 0) {
        if (public_inputs == nullptr) {
            set_last_error("null public inputs with non-zero count");
            return BB_ERR_INVALID_PUBLIC_INPUTS;
        }
        if (pub_len != expected_pub_len) {
            std::ostringstream oss;
            oss << "invalid public inputs length: expected " << expected_pub_len
                << " bytes (" << num_inputs << " inputs * 32), got " << pub_len;
            set_last_error(oss.str());
            return BB_ERR_INVALID_PUBLIC_INPUTS;
        }
    }

    try {
        auto* impl = reinterpret_cast<const bb_vkey_impl*>(vkey);

        // Validate public input count matches verification key expectation
        if (static_cast<size_t>(impl->num_public_inputs) != num_inputs) {
            std::ostringstream oss;
            oss << "public input count mismatch: verification key expects "
                << impl->num_public_inputs << ", got " << num_inputs;
            set_last_error(oss.str());
            return BB_ERR_INVALID_PUBLIC_INPUTS;
        }

        // Parse public inputs as field elements
        std::vector<bb::fr> public_inputs_vec;
        public_inputs_vec.reserve(num_inputs);

        for (size_t i = 0; i < num_inputs; ++i) {
            // Read 32-byte big-endian field element
            const uint8_t* elem_ptr = public_inputs + (i * FIELD_SIZE);

            // Convert from big-endian bytes to field element
            bb::fr elem;
            // barretenberg field elements are stored in Montgomery form
            // We need to read the big-endian representation and convert
            uint256_t value = 0;
            for (size_t j = 0; j < FIELD_SIZE; ++j) {
                value = (value << 8) | elem_ptr[j];
            }
            elem = bb::fr(value);
            public_inputs_vec.push_back(elem);
        }

        // Deserialize the proof
        std::vector<uint8_t> proof_bytes(proof, proof + proof_len);

        // Create the UltraHonk verifier
        bb::UltraVerifier verifier(impl->vk);

        // Parse the proof into the HonkProof structure
        bb::HonkProof honk_proof(proof_bytes);

        // Perform verification
        bool verified = verifier.verify_proof(honk_proof, public_inputs_vec);

        if (!verified) {
            set_last_error("proof verification failed");
            return BB_ERR_VERIFICATION_FAILED;
        }

        return BB_SUCCESS;

    } catch (const std::exception& e) {
        std::ostringstream oss;
        oss << "verification error: " << e.what();
        set_last_error(oss.str());
        return BB_ERR_INTERNAL;
    } catch (...) {
        set_last_error("unknown error during verification");
        return BB_ERR_INTERNAL;
    }
}

const char* bb_get_last_error(void) {
    if (g_last_error.empty()) {
        return nullptr;
    }
    return g_last_error.c_str();
}

void bb_clear_last_error(void) {
    clear_last_error();
}

const char* bb_version(void) {
    return BB_VERSION;
}

int bb_supports_ultrahonk(void) {
    return 1; // This wrapper only supports UltraHonk
}

} // extern "C"
