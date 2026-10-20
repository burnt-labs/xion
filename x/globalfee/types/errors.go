package types

import errorsmod "cosmossdk.io/errors"

var ErrInvalidAuthority = errorsmod.Register(ModuleName, 2, "invalid authority")
