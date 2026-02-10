#!/usr/bin/env bash
#
# generate.sh - Generate test vectors for Barretenberg bindings
#
# Prerequisites:
#   - Noir (nargo): https://noir-lang.org/docs/getting_started/installation
#   - Barretenberg CLI (bb): npm install -g @aztec/bb
#
# Usage:
#   ./generate.sh
#

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEMP_DIR=$(mktemp -d)

cleanup() {
    rm -rf "${TEMP_DIR}"
}
trap cleanup EXIT

# Clean up stale artifacts and recreate output directory
rm -rf "${SCRIPT_DIR}/statics"
mkdir -p "${SCRIPT_DIR}/statics"

echo "Checking prerequisites..."

if ! command -v nargo &> /dev/null; then
    echo "Error: nargo not found. Install Noir: https://noir-lang.org/docs/getting_started/installation"
    exit 1
fi

if ! command -v bb &> /dev/null; then
    echo "Error: bb not found. Install: npm install -g @aztec/bb"
    exit 1
fi

echo "Creating test circuit..."
cd "${TEMP_DIR}"

# Create Noir project structure manually
mkdir -p test_circuit/src
cat > test_circuit/Nargo.toml << 'EOF'
[package]
name = "test_circuit"
type = "bin"
authors = [""]
compiler_version = ">=0.30.0"

[dependencies]
EOF

# Create a simple circuit: verify x * x == y
cat > test_circuit/src/main.nr << 'EOF'
fn main(x: pub Field, y: pub Field) {
    assert(x * x == y);
}
EOF

cd test_circuit

echo "Compiling circuit..."
nargo compile

echo "Creating witness..."
cat > Prover.toml << 'EOF'
x = "3"
y = "9"
EOF

nargo execute witness

echo "Generating verification key..."
BB_VK_DIR="${TEMP_DIR}/bb_vk_out"
mkdir -p "${BB_VK_DIR}"
bb write_vk --scheme ultra_honk \
    -b ./target/test_circuit.json \
    -o "${BB_VK_DIR}"

# bb v3 writes "vk" file inside the output directory
if [ ! -f "${BB_VK_DIR}/vk" ]; then
    echo "Error: bb write_vk did not produce ${BB_VK_DIR}/vk"
    ls -la "${BB_VK_DIR}"
    exit 1
fi
cp "${BB_VK_DIR}/vk" "${SCRIPT_DIR}/statics/vk"

echo "Generating proof..."
BB_PROOF_DIR="${TEMP_DIR}/bb_proof_out"
mkdir -p "${BB_PROOF_DIR}"
bb prove --scheme ultra_honk \
    -b ./target/test_circuit.json \
    -w ./target/witness.gz \
    -k "${BB_VK_DIR}/vk" \
    -o "${BB_PROOF_DIR}"

if [ ! -f "${BB_PROOF_DIR}/proof" ]; then
    echo "Error: bb prove did not produce ${BB_PROOF_DIR}/proof"
    ls -la "${BB_PROOF_DIR}"
    exit 1
fi
cp "${BB_PROOF_DIR}/proof" "${SCRIPT_DIR}/statics/proof"

echo "Self-verifying proof..."
bb verify --scheme ultra_honk \
    -k "${BB_VK_DIR}/vk" \
    -p "${BB_PROOF_DIR}/proof" \
    -i "${BB_PROOF_DIR}/public_inputs"

echo "Copying public inputs..."
if [ ! -f "${BB_PROOF_DIR}/public_inputs" ]; then
    echo "Error: bb prove did not produce ${BB_PROOF_DIR}/public_inputs"
    ls -la "${BB_PROOF_DIR}"
    exit 1
fi
cp "${BB_PROOF_DIR}/public_inputs" "${SCRIPT_DIR}/statics/public_inputs"

echo ""
echo "Generated test vectors:"
ls -la "${SCRIPT_DIR}/statics/"

echo ""
echo "Done! Test vectors are ready in ${SCRIPT_DIR}/statics/"
