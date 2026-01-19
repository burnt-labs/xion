# Test Vectors for Barretenberg Bindings

This directory contains test vectors for the Barretenberg Go bindings.

## Required Files

- `valid_ultrahonk_vkey.bin` - UltraHonk verification key
- `valid_ultrahonk_proof.bin` - Valid UltraHonk proof
- `test_inputs.json` - Public inputs for the proof

## Generating Test Vectors

You can generate test vectors using Noir and the Barretenberg CLI.

### Prerequisites

1. Install Noir: https://noir-lang.org/docs/getting_started/installation
2. Install Barretenberg CLI: `npm install -g @aztec/bb`

### Steps

1. Create a simple test circuit:

```bash
# Create new Noir project
nargo new test_circuit
cd test_circuit
```

2. Edit `src/main.nr` with a simple circuit:

```noir
fn main(x: pub Field, y: pub Field) {
    assert(x * x == y);
}
```

3. Compile and generate artifacts:

```bash
# Compile the circuit
nargo compile

# Create witness (with x=3, y=9)
echo '{"x": "3", "y": "9"}' > Prover.toml
nargo execute witness

# Generate verification key
bb write_vk --scheme ultra_honk -b ./target/test_circuit.json -o ../valid_ultrahonk_vkey.bin

# Generate proof
bb prove --scheme ultra_honk -b ./target/test_circuit.json -w ./target/witness.gz -o ../valid_ultrahonk_proof.bin
```

4. Create `test_inputs.json`:

```json
{
  "public_inputs": ["3", "9"]
}
```

## File Formats

### Verification Key (`.bin`)
Binary format as output by `bb write_vk --scheme ultra_honk`.

### Proof (`.bin`)
Binary format as output by `bb prove --scheme ultra_honk`.

### Public Inputs (`.json`)
JSON object with a `public_inputs` array of field elements as strings:
- Decimal: `"42"`
- Hex with prefix: `"0x2a"`
