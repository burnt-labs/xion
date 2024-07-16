package types

import errorsmod "cosmossdk.io/errors"

// Codes for general xion errors
const (
	DefaultCodespace = ModuleName
)

var ErrNoAllowedContracts = errorsmod.Register(DefaultCodespace, 2, "no contract addresses specified")
