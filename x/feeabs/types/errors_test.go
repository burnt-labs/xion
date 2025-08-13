package types

import (
	"testing"

	sdkerrors "cosmossdk.io/errors"
	"github.com/stretchr/testify/require"
)

func TestErrors(t *testing.T) {
	// Test all error variables to ensure they are properly defined
	errorTests := []struct {
		name      string
		err       error
		code      uint32
		codespace string
		message   string
	}{
		{
			name:      "ErrInvalidExchangeRate",
			err:       ErrInvalidExchangeRate,
			code:      1,
			codespace: ModuleName,
			message:   "invalid exchange rate",
		},
		{
			name:      "ErrInvalidIBCFees",
			err:       ErrInvalidIBCFees,
			code:      2,
			codespace: ModuleName,
			message:   "invalid ibc fees",
		},
		{
			name:      "ErrHostZoneConfigNotFound",
			err:       ErrHostZoneConfigNotFound,
			code:      3,
			codespace: ModuleName,
			message:   "host zone config not found",
		},
		{
			name:      "ErrDuplicateHostZoneConfig",
			err:       ErrDuplicateHostZoneConfig,
			code:      4,
			codespace: ModuleName,
			message:   "duplicate host zone config",
		},
		{
			name:      "ErrNotEnoughFundInModuleAddress",
			err:       ErrNotEnoughFundInModuleAddress,
			code:      5,
			codespace: ModuleName,
			message:   "not have funding yet",
		},
		{
			name:      "ErrUnsupportedDenom",
			err:       ErrUnsupportedDenom,
			code:      6,
			codespace: ModuleName,
			message:   "unsupported denom",
		},
		{
			name:      "ErrHostZoneFrozen",
			err:       ErrHostZoneFrozen,
			code:      7,
			codespace: ModuleName,
			message:   "host zone is frozen",
		},
		{
			name:      "ErrHostZoneOutdated",
			err:       ErrHostZoneOutdated,
			code:      8,
			codespace: ModuleName,
			message:   "host zone is outdated",
		},
		{
			name:      "ErrInvalidSigner",
			err:       ErrInvalidSigner,
			code:      9,
			codespace: ModuleName,
			message:   "invalid signer",
		},
	}

	for _, tt := range errorTests {
		t.Run(tt.name, func(t *testing.T) {
			require.NotNil(t, tt.err)

			// Test that it's a properly registered error
			sdkErr, ok := tt.err.(*sdkerrors.Error)
			require.True(t, ok, "error should be of type *sdkerrors.Error")

			require.Equal(t, tt.code, sdkErr.ABCICode())
			require.Equal(t, tt.codespace, sdkErr.Codespace())
			require.Contains(t, sdkErr.Error(), tt.message)
		})
	}
}
