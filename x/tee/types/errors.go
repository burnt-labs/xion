package types

// DONTCOVER

import (
	errorsmod "cosmossdk.io/errors"
)

var (
	ErrInvalidRequest    = errorsmod.Register(ModuleName, 1100, "invalid request")
	ErrQuoteTooLarge     = errorsmod.Register(ModuleName, 1101, "quote exceeds maximum size")
	ErrQuoteParseFailed  = errorsmod.Register(ModuleName, 1102, "failed to parse TDX quote")
	ErrQuoteVerifyFailed = errorsmod.Register(ModuleName, 1103, "TDX quote verification failed")
)
