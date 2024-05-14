package types

// DONTCOVER

import (
	errorsmod "cosmossdk.io/errors"
)

// x/jwk module sentinel errors
var (
	ErrInvalidJWK = errorsmod.Register(ModuleName, 1100, "invalid jwk")
)
