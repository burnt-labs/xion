/*
 * barretenberg_wrapper.cpp
 *
 * C++ implementation of the Barretenberg wrapper for UltraHonk verification.
 */

#include "../include/barretenberg_wrapper.h"

#include <memory>
#include <sstream>
#include <string>
#include <vector>

// Barretenberg headers
#include "barretenberg/common/serialize.hpp"          // from_buffer<T>, many_from_buffer
#include "barretenberg/flavor/ultra_zk_flavor.hpp"    // UltraZKFlavor
#include "barretenberg/ultra_honk/ultra_verifier.hpp" // UltraVerifier_
#include "barretenberg/srs/global_crs.hpp"            // init_net_crs_factory, bb_crs_path

// Version information
#ifndef BB_VERSION
#define BB_VERSION "4.0.4"
#endif

namespace
{

    // Thread-local error message storage
    thread_local std::string g_last_error;

    void set_last_error(const std::string &msg)
    {
        g_last_error = msg;
    }

    void clear_last_error()
    {
        g_last_error.clear();
    }

    // Use UltraZKFlavor to match the default behavior of the BB CLI (ZK enabled by default)
    using Flavor = bb::UltraZKFlavor;
    using VerificationKey = Flavor::VerificationKey;

    // Wrapper struct for verification key — stores raw bytes for fresh deserialization
    struct bb_vkey_impl
    {
        std::vector<uint8_t> vk_bytes;
        std::shared_ptr<VerificationKey> vk;
        uint32_t num_public_inputs;
        uint64_t circuit_size;
    };

} // anonymous namespace

extern "C"
{

    bb_vkey_t *bb_vkey_from_bytes(const uint8_t *data, size_t len)
    {
        clear_last_error();

        if (data == nullptr || len == 0)
        {
            set_last_error("null or empty verification key data");
            return nullptr;
        }

        // Minimum reasonable size for an UltraHonk verification key
        constexpr size_t MIN_VKEY_SIZE = 1024;
        if (len < MIN_VKEY_SIZE)
        {
            set_last_error("verification key data too small");
            return nullptr;
        }

        try
        {
            // Create a vector from the input data
            std::vector<uint8_t> vk_bytes(data, data + len);

            // Deserialize the verification key using from_buffer (binary serialization)
            auto vk = std::make_shared<VerificationKey>(
                from_buffer<VerificationKey>(vk_bytes));

            // Allocate the wrapper struct
            auto *impl = new (std::nothrow) bb_vkey_impl();
            if (impl == nullptr)
            {
                set_last_error("failed to allocate verification key wrapper");
                return nullptr;
            }

            impl->vk_bytes = std::move(vk_bytes);
            impl->vk = vk;
            impl->num_public_inputs = vk->num_public_inputs;
            impl->circuit_size = 1ULL << vk->log_circuit_size;

            return reinterpret_cast<bb_vkey_t *>(impl);
        }
        catch (const std::exception &e)
        {
            std::ostringstream oss;
            oss << "failed to parse verification key: " << e.what();
            set_last_error(oss.str());
            return nullptr;
        }
        catch (...)
        {
            set_last_error("unknown error parsing verification key");
            return nullptr;
        }
    }

    void bb_vkey_free(bb_vkey_t *vkey)
    {
        if (vkey != nullptr)
        {
            auto *impl = reinterpret_cast<bb_vkey_impl *>(vkey);
            delete impl;
        }
    }

    int32_t bb_vkey_num_public_inputs(const bb_vkey_t *vkey)
    {
        clear_last_error();

        if (vkey == nullptr)
        {
            set_last_error("null verification key");
            return -1;
        }

        auto *impl = reinterpret_cast<const bb_vkey_impl *>(vkey);
        return static_cast<int32_t>(impl->num_public_inputs);
    }

    uint64_t bb_vkey_circuit_size(const bb_vkey_t *vkey)
    {
        clear_last_error();

        if (vkey == nullptr)
        {
            set_last_error("null verification key");
            return 0;
        }

        auto *impl = reinterpret_cast<const bb_vkey_impl *>(vkey);
        return impl->circuit_size;
    }

    bb_error_t bb_verify_proof(
        const bb_vkey_t *vkey,
        const uint8_t *proof, size_t proof_len,
        const uint8_t *public_inputs, size_t pub_len,
        size_t num_inputs)
    {
        clear_last_error();

        // Validate inputs
        if (vkey == nullptr)
        {
            set_last_error("null verification key");
            return BB_ERR_NULL_POINTER;
        }

        if (proof == nullptr || proof_len == 0)
        {
            set_last_error("null or empty proof data");
            return BB_ERR_INVALID_PROOF;
        }

        // Validate public inputs size
        constexpr size_t FIELD_SIZE = 32; // 32 bytes per field element
        size_t expected_pub_len = num_inputs * FIELD_SIZE;
        if (num_inputs > 0)
        {
            if (public_inputs == nullptr)
            {
                set_last_error("null public inputs with non-zero count");
                return BB_ERR_INVALID_PUBLIC_INPUTS;
            }
            if (pub_len != expected_pub_len)
            {
                std::ostringstream oss;
                oss << "invalid public inputs length: expected " << expected_pub_len
                    << " bytes (" << num_inputs << " inputs * 32), got " << pub_len;
                set_last_error(oss.str());
                return BB_ERR_INVALID_PUBLIC_INPUTS;
            }
        }

        try
        {
            auto *impl = reinterpret_cast<const bb_vkey_impl *>(vkey);

            // Initialise the CRS factory (reads from BB_CRS_PATH env var or ~/.bb-crs).
            // Validators: set BB_CRS_PATH to a pre-populated directory.
            // If the path is absent or CRS files are missing, verification will throw below.
            try
            {
                const char *crs_path_env = std::getenv("BB_CRS_PATH");
                std::string crs_path = crs_path_env ? std::string(crs_path_env) : bb::srs::bb_crs_path().string();
                bb::srs::init_net_crs_factory(crs_path);
            }
            catch (const std::exception &e)
            {
                std::ostringstream oss;
                oss << "CRS initialisation failed (set BB_CRS_PATH or populate ~/.bb-crs): " << e.what();
                set_last_error(oss.str());
                return BB_ERR_INTERNAL;
            }

            // Deserialize VK fresh from bytes — mirrors BB CLI _verify() exactly
            auto vk = std::make_shared<VerificationKey>(
                from_buffer<VerificationKey>(impl->vk_bytes));
            auto vk_and_hash = std::make_shared<Flavor::VKAndHash>(vk);

            // Use the same deserialization as the BB CLI (many_from_buffer<uint256_t>)
            std::vector<uint8_t> pub_bytes(public_inputs, public_inputs + pub_len);
            std::vector<uint8_t> proof_bytes(proof, proof + proof_len);

            auto pub_u256 = many_from_buffer<uint256_t>(pub_bytes);
            auto proof_u256 = many_from_buffer<uint256_t>(proof_bytes);

            // Concatenate public inputs + proof into a single vector
            using DataType = typename Flavor::Transcript::DataType;
            std::vector<DataType> complete_proof;
            complete_proof.reserve(pub_u256.size() + proof_u256.size());
            complete_proof.insert(complete_proof.end(), pub_u256.begin(), pub_u256.end());
            complete_proof.insert(complete_proof.end(), proof_u256.begin(), proof_u256.end());

            // Verify using UltraZKVerifier (UltraZKFlavor + DefaultIO); v4 API takes VKAndHash
            bb::UltraZKVerifier verifier{vk_and_hash};
            auto output = verifier.verify_proof(complete_proof);

            if (!output.result)
            {
                set_last_error("proof verification failed");
                return BB_ERR_VERIFICATION_FAILED;
            }

            return BB_SUCCESS;
        }
        catch (const std::exception &e)
        {
            std::ostringstream oss;
            oss << "verification error: " << e.what();
            set_last_error(oss.str());
            return BB_ERR_INTERNAL;
        }
        catch (...)
        {
            set_last_error("unknown error during verification");
            return BB_ERR_INTERNAL;
        }
    }

    const char *bb_get_last_error(void)
    {
        if (g_last_error.empty())
        {
            return nullptr;
        }
        return g_last_error.c_str();
    }

    void bb_clear_last_error(void)
    {
        clear_last_error();
    }

    const char *bb_version(void)
    {
        return BB_VERSION;
    }

    int bb_supports_ultrahonk(void)
    {
        return 1; // This wrapper only supports UltraHonk
    }

} // extern "C"