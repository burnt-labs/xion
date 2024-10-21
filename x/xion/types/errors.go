package types

import errorsmod "cosmossdk.io/errors"

// Codes for general xion errors
const (
	DefaultCodespace = ModuleName
)

var (
	ErrNoAllowedContracts = errorsmod.Register(DefaultCodespace, 2, "no contract addresses specified")
	ErrNoValidAllowances  = errorsmod.Register(DefaultCodespace, 3, "none of the allowances accepted the msg")
	ErrInconsistentExpiry = errorsmod.Register(DefaultCodespace, 4, "multi allowances must all expire together")
	ErrMinimumNotMet      = errorsmod.Register(DefaultCodespace, 5, "minimum send amount not met")
)
