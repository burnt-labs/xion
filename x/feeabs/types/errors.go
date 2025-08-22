package types

import (
	sdkerrors "cosmossdk.io/errors"
)

var (
	ErrInvalidExchangeRate          = sdkerrors.Register(ModuleName, 1, "invalid exchange rate")
	ErrInvalidIBCFees               = sdkerrors.Register(ModuleName, 2, "invalid ibc fees")
	ErrHostZoneConfigNotFound       = sdkerrors.Register(ModuleName, 3, "host zone config not found")
	ErrDuplicateHostZoneConfig      = sdkerrors.Register(ModuleName, 4, "duplicate host zone config")
	ErrNotEnoughFundInModuleAddress = sdkerrors.Register(ModuleName, 5, "not have funding yet")
	ErrUnsupportedDenom             = sdkerrors.Register(ModuleName, 6, "unsupported denom")
	ErrHostZoneFrozen               = sdkerrors.Register(ModuleName, 7, "host zone is frozen")
	ErrHostZoneOutdated             = sdkerrors.Register(ModuleName, 8, "host zone is outdated")

	ErrInvalidSigner = sdkerrors.Register(ModuleName, 9, "invalid signer")
)
