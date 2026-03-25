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
		return errorsmod.Wrapf(ErrInvalidParams, "max_ultrahonk_proof_size_bytes must be positive")
	}

	if p.MaxUltraHonkPublicInputSizeBytes == 0 {
		return errorsmod.Wrapf(ErrInvalidParams, "max_ultrahonk_public_input_size_bytes must be positive")
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
	return chunks * p.UploadChunkGas, nil
}
