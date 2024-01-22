package types

// DONTCOVER

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/jwk module sentinel errors
var (
	ErrInvalidJWK = sdkerrors.Register(ModuleName, 1100, "invalid jwk")
)
