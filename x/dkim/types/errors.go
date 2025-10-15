package types

// DONTCOVER

import (
	errorsmod "cosmossdk.io/errors"
)

// x/jwk module sentinel errors
var (
	ErrParsingPrivKey      = errorsmod.Register(ModuleName, 1100, "error parsing privkey")
	ErrParsingPubKey       = errorsmod.Register(ModuleName, 1101, "error parsing pubkey")
	ErrEncodingElement     = errorsmod.Register(ModuleName, 1102, "error encoding element")
	ErrCalculatingPoseidon = errorsmod.Register(ModuleName, 1103, "error hashing poseidon hash")
	ErrInvalidPublicInput  = errorsmod.Register(ModuleName, 1104, "invalid public input")
)
