# Barretenberg Go Bindings

Go CGo bindings for [Barretenberg](https://github.com/AztecProtocol/barretenberg)'s UltraHonk proof verification system.

## Overview

This package provides Go bindings for verifying zero-knowledge proofs generated using [Noir](https://noir-lang.org/) compiled to the UltraHonk proof system. It wraps Barretenberg's C++ verification logic with a safe, idiomatic Go API.

## Building

### Development (Stub Library)

For development and testing without full Barretenberg dependencies:

```bash
make barretenberg-build
# or explicitly:
make barretenberg-build-stub
```

This builds a stub library that allows the Go code to compile and run basic tests.

### Production (Full Library)

Building the full Barretenberg library from source requires:
- CMake 3.16+
- C++20 compiler (clang/gcc)
- Docker (for cross-platform builds)

```bash
# Build from source (requires all dependencies)
make barretenberg-build-full

# Or build using Docker (recommended for cross-compilation)
make barretenberg-build-docker
```

### Pre-built Binaries

For production use, you can also place pre-built static libraries in:
- `lib/linux_amd64/libbarretenberg.a`
- `lib/linux_arm64/libbarretenberg.a`
- `lib/darwin_arm64/libbarretenberg.a`

## Usage

```go
package main

import (
    "log"

    "github.com/burnt-labs/xion/x/zk/barretenberg"
)

func main() {
    // Parse verification key
    vkey, err := barretenberg.ParseVerificationKey(vkeyBytes)
    if err != nil {
        log.Fatal(err)
    }
    defer vkey.Close()

    // Parse proof
    proof, err := barretenberg.ParseProof(proofBytes)
    if err != nil {
        log.Fatal(err)
    }

    // Create verifier
    verifier, err := barretenberg.NewVerifier(vkey)
    if err != nil {
        log.Fatal(err)
    }
    defer verifier.Close()

    // Verify proof with public inputs
    valid, err := verifier.Verify(proof, []string{"42", "0x1234..."})
    if err != nil {
        log.Fatal(err)
    }

    if valid {
        log.Println("Proof verified successfully!")
    } else {
        log.Println("Proof verification failed")
    }
}
```

## Testing

```bash
# Run tests
make barretenberg-test

# Run benchmarks
make barretenberg-bench
```

### Test Vectors

To generate test vectors (requires Noir and bb CLI):

```bash
make barretenberg-generate-testdata
```

Or manually:

```bash
cd testdata && ./generate.sh
```

## API Reference

### Types

- `VerificationKey` - Parsed UltraHonk verification key
- `Proof` - Parsed UltraHonk proof
- `Verifier` - Thread-safe proof verifier
- `PublicInputs` - Collection of public input field elements

### Functions

- `ParseVerificationKey(data []byte)` - Parse vkey from binary
- `ParseProof(data []byte)` - Parse proof from binary
- `NewVerifier(vkey *VerificationKey)` - Create verifier
- `VerifyProofBytes(vkey, proof []byte, inputs []string)` - One-shot verification

### Error Handling

The package uses sentinel errors that can be checked with `errors.Is()`:

- `ErrInvalidVKey` - Verification key is malformed
- `ErrInvalidProof` - Proof data is malformed
- `ErrInvalidPublicInputs` - Public inputs are invalid
- `ErrVerificationFailed` - Proof is invalid (not an error)
- `ErrClosed` - Operation on closed resource

## Package Structure

```
x/zk/barretenberg/
├── lib/                          # Pre-built static libraries
│   ├── linux_amd64/
│   ├── linux_arm64/
│   └── darwin_arm64/
├── include/
│   └── barretenberg_wrapper.h    # C API declarations
├── wrapper/
│   ├── barretenberg_wrapper.cpp  # C++ implementation
│   ├── barretenberg_stub.cpp     # Stub for development
│   ├── CMakeLists.txt
│   └── build.sh
├── bindings.go                   # CGo interface
├── verifier.go                   # High-level Verifier API
├── vkey.go                       # VKey parsing
├── proof.go                      # Proof parsing
├── errors.go                     # Error types
├── doc.go                        # Package documentation
├── *_test.go                     # Tests
└── testdata/                     # Test vectors
```

## Thread Safety

All types in this package are safe for concurrent use. The `Verifier` uses internal locking to ensure thread-safe verification.

## Resource Management

`VerificationKey` and `Verifier` hold native (C++) resources. Call `Close()` when done, or rely on Go's garbage collector (though explicit `Close()` is preferred for timely release).

## Supported Proof Systems

Currently only **UltraHonk** proofs are supported. This is Noir's current default backend.
