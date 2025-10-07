package types

// DONTCOVER

import (
	errorsmod "cosmossdk.io/errors"
)

var (
	ErrEncodingElement     = errorsmod.Register(ModuleName, 1100, "error encoding element")
	ErrCalculatingPoseidon = errorsmod.Register(ModuleName, 1101, "error hashing poseidon hash")
)
