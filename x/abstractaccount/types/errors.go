package types

import "cosmossdk.io/errors"

var (
	ErrMalformedAllowList            = errors.Register(ModuleName, 2, "code ID allow list must contain non-zero, unique, and sorted code IDs")
	ErrNonEmptyAllowList             = errors.Register(ModuleName, 3, "code ID allow list must be empty when AllowAllCodeIDs is true")
	ErrNotAllowedCodeID              = errors.Register(ModuleName, 4, "not an allowed wasm code ID")
	ErrNotBaseAccount                = errors.Register(ModuleName, 5, "account is not an authtypes.BaseAccount")
	ErrNotSingleSignature            = errors.Register(ModuleName, 6, "signature is not a txsigning.SingleSignatureData")
	ErrParsingParams                 = errors.Register(ModuleName, 7, "failed to marshal or unmarshal module params")
	ErrZeroMaxGas                    = errors.Register(ModuleName, 8, "max gas cannot be zero")
	ErrNoBlockTime                   = errors.Register(ModuleName, 9, "block time can not be zero")
	ErrRegistrationDisabled          = errors.Register(ModuleName, 10, "abstract account registration is disabled")
	ErrImmutableAddressHash          = errors.Register(ModuleName, 12, "address derivation hash is immutable once configured")
	ErrCodeIDNotFound                = errors.Register(ModuleName, 13, "wasm code ID does not exist")
	ErrAccountAlreadyRegistered      = errors.Register(ModuleName, 14, "account namespace is already registered")
	ErrRegistrationNotConfigured     = errors.Register(ModuleName, 16, "abstract account address derivation hash is not configured")
	ErrInvalidAddressDerivationHash  = errors.Register(ModuleName, 17, "address derivation hash must be exactly 32 bytes")
	ErrInvalidAccountAddressRegistry = errors.Register(ModuleName, 18, "invalid account address registry entry")
)
