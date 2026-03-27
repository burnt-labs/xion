package types

import (
	"encoding/json"

	errorsmod "cosmossdk.io/errors"
)

const (
	// DefaultMaxVKeySizeBytes caps the default allowed verification key payload size.
	DefaultMaxVKeySizeBytes uint64 = 256 * 1024 // 256 KiB
	// DefaultUploadChunkSize defines the default tier size (bytes) used to scale gas.
	DefaultUploadChunkSize uint64 = 20
	// DefaultUploadChunkGas defines the default gas cost per upload chunk.
	DefaultUploadChunkGas uint64 = 10_000

	// DefaultMaxGroth16ProofSizeBytes caps the maximum allowed Groth16 proof JSON payload size.
	// Used by `Query.ProofVerify`.
	DefaultMaxGroth16ProofSizeBytes uint64 = 4 * 1024 // 4 KiB
	// DefaultMaxGroth16PublicInputSizeBytes caps the maximum allowed Groth16 public inputs size.
	// The size is computed as the total UTF-8 byte length of all provided public input strings.
	DefaultMaxGroth16PublicInputSizeBytes uint64 = 30 * 1024 // 30 KiB

	// DefaultMaxUltraHonkProofSizeBytes caps the maximum allowed UltraHonk proof bytes size.
	// Used by `Query.ProofVerifyUltraHonk`.
	DefaultMaxUltraHonkProofSizeBytes uint64 = 20 * 1024 // 20 KiB

	// DefaultMaxUltraHonkPublicInputSizeBytes caps the maximum allowed UltraHonk public inputs bytes size.
	// For UltraHonk, public inputs are provided as raw bytes.
	DefaultMaxUltraHonkPublicInputSizeBytes uint64 = 10 * 1024 // 10 KiB

	// Gas constants for proof verification queries.
	// These are charged proportional to input sizes to prevent free DoS via
	// Stargate-whitelisted or CosmWasm-callable query endpoints.

	// ProofVerifyBaseGas is the flat overhead charged on every Groth16/BN254 proof
	// verification call regardless of payload size.
	ProofVerifyBaseGas uint64 = 100_000
	// ProofVerifyPerByteGas is charged per byte of proof + public inputs for Groth16.
	ProofVerifyPerByteGas uint64 = 10

	// ProofVerifyUltraHonkBaseGas is the flat overhead for every UltraHonk verification call.
	ProofVerifyUltraHonkBaseGas uint64 = 150_000
	// ProofVerifyUltraHonkPerByteGas is charged per byte of proof + public inputs for UltraHonk.
	ProofVerifyUltraHonkPerByteGas uint64 = 15
)

// NewParams creates a new Params instance.
func NewParams(maxSize, chunkSize, chunkGas uint64) Params {
	return Params{
		MaxVkeySizeBytes:                 maxSize,
		UploadChunkSize:                  chunkSize,
		UploadChunkGas:                   chunkGas,
		MaxGroth16ProofSizeBytes:         DefaultMaxGroth16ProofSizeBytes,
		MaxGroth16PublicInputSizeBytes:   DefaultMaxGroth16PublicInputSizeBytes,
		MaxUltraHonkProofSizeBytes:       DefaultMaxUltraHonkProofSizeBytes,
		MaxUltraHonkPublicInputSizeBytes: DefaultMaxUltraHonkPublicInputSizeBytes,
	}
}

// DefaultParams returns the default module parameters.
func DefaultParams() Params {
	return NewParams(DefaultMaxVKeySizeBytes, DefaultUploadChunkSize, DefaultUploadChunkGas)
}

// WithMaxLimitDefaults backfills proof/public-input max size parameters when they are unset (zero).
//
// This keeps old chains/genesis compatible after the params schema evolves.
func (p Params) WithMaxLimitDefaults() Params {
	if p.MaxGroth16ProofSizeBytes == 0 {
		p.MaxGroth16ProofSizeBytes = DefaultMaxGroth16ProofSizeBytes
	}
	if p.MaxGroth16PublicInputSizeBytes == 0 {
		p.MaxGroth16PublicInputSizeBytes = DefaultMaxGroth16PublicInputSizeBytes
	}

	if p.MaxUltraHonkProofSizeBytes == 0 {
		p.MaxUltraHonkProofSizeBytes = DefaultMaxUltraHonkProofSizeBytes
	}
	if p.MaxUltraHonkPublicInputSizeBytes == 0 {
		p.MaxUltraHonkPublicInputSizeBytes = DefaultMaxUltraHonkPublicInputSizeBytes
	}
	return p
}

// String implements the fmt.Stringer interface.
func (p Params) String() string {
	bz, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}

	return string(bz)
}

// Validate performs basic parameter validation.
func (p Params) Validate() error {
	if p.MaxVkeySizeBytes == 0 {
		return errorsmod.Wrapf(ErrInvalidParams, "max_vkey_size_bytes must be positive")
	}

	if p.UploadChunkSize == 0 {
		return errorsmod.Wrapf(ErrInvalidParams, "upload_chunk_size must be positive")
	}

	if p.UploadChunkGas == 0 {
		return errorsmod.Wrapf(ErrInvalidParams, "upload_chunk_gas must be positive")
	}

	if p.MaxGroth16ProofSizeBytes == 0 {
		return errorsmod.Wrapf(ErrInvalidParams, "max_groth16_proof_size_bytes must be positive")
	}

	if p.MaxGroth16PublicInputSizeBytes == 0 {
		return errorsmod.Wrapf(ErrInvalidParams, "max_groth16_public_input_size_bytes must be positive")
	}

	if p.MaxUltraHonkProofSizeBytes == 0 {
		return errorsmod.Wrapf(ErrInvalidParams, "max_ultra_honk_proof_size_bytes must be positive")
	}

	if p.MaxUltraHonkPublicInputSizeBytes == 0 {
		return errorsmod.Wrapf(ErrInvalidParams, "max_ultra_honk_public_input_size_bytes must be positive")
	}

	return nil
}

// GasCostForSize returns the gas cost for storing a payload of the given size.
func (p Params) GasCostForSize(size uint64) (uint64, error) {
	if err := p.Validate(); err != nil {
		return 0, err
	}

	if size == 0 {
		return 0, errorsmod.Wrap(ErrInvalidVKey, "vkey_bytes cannot be empty")
	}

	if size > p.MaxVkeySizeBytes {
		return 0, errorsmod.Wrapf(ErrVKeyTooLarge, "size %d > max %d", size, p.MaxVkeySizeBytes)
	}

	chunks := (size + p.UploadChunkSize - 1) / p.UploadChunkSize
	cost := chunks * p.UploadChunkGas
	// Check for overflow
	if chunks != 0 && cost/chunks != p.UploadChunkGas {
		return 0, errorsmod.Wrapf(ErrInvalidParams, "gas cost overflow: chunks=%d, chunkGas=%d", chunks, p.UploadChunkGas)
	}
	return cost, nil
}
