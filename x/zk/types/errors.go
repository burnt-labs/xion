package types

// DONTCOVER

import (
	errorsmod "cosmossdk.io/errors"
)

var (
	ErrEncodingElement     = errorsmod.Register(ModuleName, 1100, "error encoding element")
	ErrCalculatingPoseidon = errorsmod.Register(ModuleName, 1101, "error hashing poseidon hash")
	ErrInvalidAuthority    = errorsmod.Register(ModuleName, 1102, "invalid authority")
	ErrVKeyExists          = errorsmod.Register(ModuleName, 1103, "verification key already exists")
	ErrVKeyNotFound        = errorsmod.Register(ModuleName, 1104, "verification key not found")
	ErrInvalidVKey         = errorsmod.Register(ModuleName, 1105, "invalid verification key")
	ErrIncreaseSequenceID  = errorsmod.Register(ModuleName, 1106, "error increasing sequence ID")
	ErrInvalidRequest      = errorsmod.Register(ModuleName, 1107, "invalid request")
)
