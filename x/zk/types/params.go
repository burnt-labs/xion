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
)

// NewParams creates a new Params instance.
func NewParams(maxSize, chunkSize, chunkGas uint64) Params {
	return Params{
		MaxVkeySizeBytes: maxSize,
		UploadChunkSize:  chunkSize,
		UploadChunkGas:   chunkGas,
	}
}

// DefaultParams returns the default module parameters.
func DefaultParams() Params {
	return NewParams(DefaultMaxVKeySizeBytes, DefaultUploadChunkSize, DefaultUploadChunkGas)
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
