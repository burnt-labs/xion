package types

import (
	"cosmossdk.io/errors"
)

// MaxQuoteSize is the maximum allowed size for a TDX quote in bytes (20KB).
const MaxQuoteSize = 20 * 1024

// ValidateQuoteSize checks that the quote does not exceed the maximum allowed size.
func ValidateQuoteSize(quote []byte) error {
	if len(quote) > MaxQuoteSize {
		return errors.Wrapf(ErrQuoteTooLarge, "quote size %d exceeds maximum %d", len(quote), MaxQuoteSize)
	}
	return nil
}
