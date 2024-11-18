package types

// DONTCOVER

import (
	errorsmod "cosmossdk.io/errors"
)

// x/jwk module sentinel errors
var (
	ErrParsingPrivKey = errorsmod.Register(ModuleName, 1100, "error parsing privkey")
)
