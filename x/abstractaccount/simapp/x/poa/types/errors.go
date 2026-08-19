package types

import "cosmossdk.io/errors"

var ErrEmptyValidatorSet = errors.Register(ModuleName, 2, "validator set is empty lmao")
