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

	ErrInvalidKeyType = errorsmod.Register(ModuleName, 1105, "invalid key type: only RSA keys are supported")
	ErrInvalidVersion = errorsmod.Register(ModuleName, 1106, "invalid version: only DKIM1 is supported")
	ErrInvalidPubKey  = errorsmod.Register(ModuleName, 1107, "invalid public key")
	ErrNotRSAKey      = errorsmod.Register(ModuleName, 1108, "public key is not an RSA key")
	ErrInvalidParams  = errorsmod.Register(ModuleName, 1109, "invalid params")
	ErrPubKeyTooLarge = errorsmod.Register(ModuleName, 1110, "dkim public key exceeds maximum size")
	ErrInvalidatedKey = errorsmod.Register(ModuleName, 1111, "dkim public key has been revoked")
	ErrInvalidEmailSubject = errorsmod.Register(ModuleName, 1111, "invalid email subject")
)
