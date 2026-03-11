package types

import (
	"encoding/json"

	errorsmod "cosmossdk.io/errors"
)

const (
	DefaultMaxPubKeySizeBytes uint64 = 512

	// Default public input indices for the Authenticate query
	DefaultMinPublicInputsLength  uint64 = 88
	DefaultEmailHashIndex         uint64 = 68
	DefaultDkimDomainRangeStart   uint64 = 0
	DefaultDkimDomainRangeEnd     uint64 = 9
	DefaultDkimHashIndex          uint64 = 9
	DefaultTxBytesRangeStart      uint64 = 12
	DefaultTxBytesRangeEnd        uint64 = 68
	DefaultEmailHostRangeStart    uint64 = 70
	DefaultEmailHostRangeEnd      uint64 = 79
	DefaultEmailSubjectRangeStart uint64 = 79
	DefaultEmailSubjectRangeEnd   uint64 = 88
)

// DefaultIndexRange returns an IndexRange with the given start and end.
func DefaultIndexRange(start, end uint64) IndexRange {
	return IndexRange{
		Start: start,
		End:   end,
	}
}

// DefaultPublicInputIndices returns the default public input indices configuration.
func DefaultPublicInputIndices() PublicInputIndices {
	return PublicInputIndices{
		MinLength:         DefaultMinPublicInputsLength,
		EmailHashIndex:    DefaultEmailHashIndex,
		DkimDomainRange:   DefaultIndexRange(DefaultDkimDomainRangeStart, DefaultDkimDomainRangeEnd),
		DkimHashIndex:     DefaultDkimHashIndex,
		TxBytesRange:      DefaultIndexRange(DefaultTxBytesRangeStart, DefaultTxBytesRangeEnd),
		EmailHostRange:    DefaultIndexRange(DefaultEmailHostRangeStart, DefaultEmailHostRangeEnd),
		EmailSubjectRange: DefaultIndexRange(DefaultEmailSubjectRangeStart, DefaultEmailSubjectRangeEnd),
	}
}

// DefaultParams returns default module parameters.
func DefaultParams() Params {
	vkeyIdentifier := uint64(1)

	return Params{
		VkeyIdentifier:     vkeyIdentifier,
		MaxPubkeySizeBytes: DefaultMaxPubKeySizeBytes,
		PublicInputIndices: DefaultPublicInputIndices(),
	}
}

// Stringer method for Params.
func (p Params) String() string {
	bz, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}

	return string(bz)
}

// Length returns the length of the range (end - start).
func (r IndexRange) Length() uint64 {
	return r.End - r.Start
}

// Validate does the sanity check on the IndexRange.
func (r IndexRange) Validate(name string) error {
	if r.End <= r.Start {
		return errorsmod.Wrapf(ErrInvalidParams, "%s: end (%d) must be greater than start (%d)", name, r.End, r.Start)
	}
	return nil
}

// Validate does the sanity check on the PublicInputIndices.
func (p PublicInputIndices) Validate() error {
	if p.MinLength == 0 {
		return errorsmod.Wrap(ErrInvalidParams, "min_length must be positive")
	}

	if err := p.DkimDomainRange.Validate("dkim_domain_range"); err != nil {
		return err
	}

	if err := p.TxBytesRange.Validate("tx_bytes_range"); err != nil {
		return err
	}

	if err := p.EmailHostRange.Validate("email_host_range"); err != nil {
		return err
	}

	if err := p.EmailSubjectRange.Validate("email_subject_range"); err != nil {
		return err
	}

	// Validate that all indices are within min_length bounds
	if p.EmailHashIndex >= p.MinLength {
		return errorsmod.Wrapf(ErrInvalidParams, "email_hash_index (%d) must be less than min_length (%d)", p.EmailHashIndex, p.MinLength)
	}

	if p.DkimHashIndex >= p.MinLength {
		return errorsmod.Wrapf(ErrInvalidParams, "dkim_hash_index (%d) must be less than min_length (%d)", p.DkimHashIndex, p.MinLength)
	}

	if p.DkimDomainRange.End > p.MinLength {
		return errorsmod.Wrapf(ErrInvalidParams, "dkim_domain_range.end (%d) must be <= min_length (%d)", p.DkimDomainRange.End, p.MinLength)
	}

	if p.TxBytesRange.End > p.MinLength {
		return errorsmod.Wrapf(ErrInvalidParams, "tx_bytes_range.end (%d) must be <= min_length (%d)", p.TxBytesRange.End, p.MinLength)
	}

	if p.EmailHostRange.End > p.MinLength {
		return errorsmod.Wrapf(ErrInvalidParams, "email_host_range.end (%d) must be <= min_length (%d)", p.EmailHostRange.End, p.MinLength)
	}

	if p.EmailSubjectRange.End > p.MinLength {
		return errorsmod.Wrapf(ErrInvalidParams, "email_subject_range.end (%d) must be <= min_length (%d)", p.EmailSubjectRange.End, p.MinLength)
	}

	return nil
}

// Validate does the sanity check on the params.
func (p Params) Validate() error {
	if p.MaxPubkeySizeBytes <= 0 {
		return errorsmod.Wrap(ErrInvalidParams, "max_pubkey_size_bytes must be positive")
	}

	if err := p.PublicInputIndices.Validate(); err != nil {
		return err
	}

	return nil
}
