// Package barretenberg provides Go bindings for Barretenberg's UltraHonk
// proof verification system.
//
// This package enables verification of zero-knowledge proofs generated using
// Noir (Aztec's domain-specific language) compiled to the UltraHonk proof system.
//
// # Overview
//
// The package provides a high-level API for proof verification:
//
//   - [VerificationKey]: Parse and manage UltraHonk verification keys
//   - [Proof]: Parse UltraHonk proofs
//   - [Verifier]: Verify proofs against verification keys
//   - [PublicInputs]: Handle public inputs for proof verification
//
// # Quick Start
//
// Basic proof verification:
//
//	// Parse verification key and proof
//	vkey, err := barretenberg.ParseVerificationKey(vkeyBytes)
//	if err != nil {
//	    return err
//	}
//	defer vkey.Close()
//
//	proof, err := barretenberg.ParseProof(proofBytes)
//	if err != nil {
//	    return err
//	}
//
//	// Create verifier and verify
//	verifier, err := barretenberg.NewVerifier(vkey)
//	if err != nil {
//	    return err
//	}
//	defer verifier.Close()
//
//	valid, err := verifier.Verify(proof, []string{"42", "0x1234..."})
//	if err != nil {
//	    return err
//	}
//	if !valid {
//	    return errors.New("proof verification failed")
//	}
//
// # One-Off Verification
//
// For single verifications, use the convenience function:
//
//	valid, err := barretenberg.VerifyProofBytes(vkeyBytes, proofBytes, publicInputs)
//
// # Thread Safety
//
// All types in this package are safe for concurrent use by multiple goroutines.
// The [Verifier] uses internal locking to ensure thread-safe verification.
//
// # Resource Management
//
// [VerificationKey] and [Verifier] hold native (C++) resources that must be
// released. Call Close() when done, or rely on Go's garbage collector for
// cleanup (though explicit Close() is preferred for timely resource release).
//
// # Error Handling
//
// The package uses sentinel errors that can be checked with errors.Is():
//
//   - [ErrInvalidVKey]: Verification key is malformed
//   - [ErrInvalidProof]: Proof data is malformed
//   - [ErrInvalidPublicInputs]: Public inputs are invalid
//   - [ErrVerificationFailed]: Proof verification failed (proof is invalid)
//   - [ErrClosed]: Operation on closed resource
//
// Note that [ErrVerificationFailed] indicates the proof is invalid, not that
// an error occurred. The [Verifier.Verify] method returns (false, nil) for
// invalid proofs to distinguish from actual errors.
//
// # Building
//
// This package requires the Barretenberg static library to be built for your
// target platform. The library files should be placed in:
//
//   - lib/linux_amd64/libbarretenberg.a
//   - lib/linux_arm64/libbarretenberg.a
//   - lib/darwin_arm64/libbarretenberg.a
//
// Use the build script in wrapper/build.sh to build the library:
//
//	cd wrapper && ./build.sh
//
// # Supported Proof Systems
//
// Currently, only UltraHonk proofs are supported. This is Noir's current
// default backend.
package barretenberg
