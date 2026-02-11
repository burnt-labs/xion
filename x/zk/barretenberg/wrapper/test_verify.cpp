// Minimal test: verify proof using the same code path as the BB CLI
#include <cstdio>
#include <fstream>
#include <iostream>
#include <vector>

#include "barretenberg/common/serialize.hpp"
#include "barretenberg/flavor/ultra_zk_flavor.hpp"
#include "barretenberg/ultra_honk/ultra_verifier.hpp"
#include "barretenberg/special_public_inputs/special_public_inputs.hpp"
#include "barretenberg/srs/global_crs.hpp"

std::vector<uint8_t> read_file(const std::string& path) {
    std::ifstream file(path, std::ios::binary);
    if (!file) {
        fprintf(stderr, "Failed to open: %s\n", path.c_str());
        exit(1);
    }
    return std::vector<uint8_t>((std::istreambuf_iterator<char>(file)),
                                 std::istreambuf_iterator<char>());
}

int main(int argc, char* argv[]) {
    if (argc != 4) {
        fprintf(stderr, "Usage: %s <vk_file> <proof_file> <public_inputs_file>\n", argv[0]);
        return 1;
    }

    try {
        // Initialize CRS
        bb::srs::init_net_crs_factory(bb::srs::bb_crs_path());

        // Read files
        auto vk_bytes = read_file(argv[1]);
        auto proof_file = read_file(argv[2]);
        auto pub_file = read_file(argv[3]);

        fprintf(stderr, "vk: %zu bytes, proof: %zu bytes, pub: %zu bytes\n",
                vk_bytes.size(), proof_file.size(), pub_file.size());

        // Parse exactly like BB CLI
        auto public_inputs = many_from_buffer<uint256_t>(pub_file);
        auto proof = many_from_buffer<uint256_t>(proof_file);

        fprintf(stderr, "public_inputs: %zu, proof elements: %zu\n",
                public_inputs.size(), proof.size());

        // Reconstruct VK
        using Flavor = bb::UltraZKFlavor;
        using VerificationKey = Flavor::VerificationKey;
        auto vk = std::make_shared<VerificationKey>(from_buffer<VerificationKey>(vk_bytes));

        fprintf(stderr, "vk->num_public_inputs: %u\n", vk->num_public_inputs);

        // Concatenate (exactly like _verify in bbapi_ultra_honk.cpp)
        using DataType = typename Flavor::Transcript::DataType;
        std::vector<DataType> complete_proof;
        complete_proof.reserve(public_inputs.size() + proof.size());
        complete_proof.insert(complete_proof.end(), public_inputs.begin(), public_inputs.end());
        complete_proof.insert(complete_proof.end(), proof.begin(), proof.end());

        fprintf(stderr, "complete_proof size: %zu\n", complete_proof.size());

        // Verify using UltraZKFlavor (matches BB CLI default: ZK enabled)
        bb::UltraVerifier_<Flavor> verifier{ vk };
        auto output = verifier.verify_proof<bb::DefaultIO>(complete_proof);

        if (output.result) {
            fprintf(stderr, "Proof verified successfully!\n");
            return 0;
        } else {
            fprintf(stderr, "Proof verification FAILED\n");
            return 1;
        }
    } catch (const std::exception& e) {
        fprintf(stderr, "Exception: %s\n", e.what());
        return 1;
    }
}
