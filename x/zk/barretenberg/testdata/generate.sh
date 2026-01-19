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
bb write_vk --scheme ultra_honk \
    -b ./target/test_circuit.json \
    -o "${SCRIPT_DIR}/valid_ultrahonk_vkey.bin"

echo "Generating proof..."
bb prove --scheme ultra_honk \
    -b ./target/test_circuit.json \
    -w ./target/witness.gz \
    -o "${SCRIPT_DIR}/valid_ultrahonk_proof.bin"

echo "Creating test inputs JSON..."
cat > "${SCRIPT_DIR}/test_inputs.json" << 'EOF'
{
  "public_inputs": ["3", "9"],
  "description": "Simple squaring circuit: verifies x^2 == y with x=3, y=9"
}
EOF

echo ""
echo "Generated test vectors:"
ls -la "${SCRIPT_DIR}"/*.bin "${SCRIPT_DIR}"/*.json 2>/dev/null || true

echo ""
echo "Done! Test vectors are ready in ${SCRIPT_DIR}"
