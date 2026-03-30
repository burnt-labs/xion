package v3

import (
	"errors"

	"cosmossdk.io/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/dkim/types"
)

// MigrateStore performs in-place migrations for the DKIM module from v2 to v3.
// This migration sets MinRsaKeyBits on existing params to DefaultMinRSAKeyBits
// (1024) for chains that were upgraded before the field existed (proto default
// would otherwise leave it as 0, which fails Params.Validate()).
func MigrateStore(
	ctx sdk.Context,
	paramsCollection collections.Item[types.Params],
) error {
	ctx.Logger().Info("Running DKIM module migration from v2 to v3")

	existingParams, err := paramsCollection.Get(ctx)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			ctx.Logger().Info("No existing params found, setting defaults")
			return paramsCollection.Set(ctx, types.DefaultParams())
		}
		return err
	}

	if existingParams.MinRsaKeyBits == 0 {
		existingParams.MinRsaKeyBits = types.DefaultMinRSAKeyBits
		ctx.Logger().Info("Setting MinRsaKeyBits to default", "value", types.DefaultMinRSAKeyBits)
	}

	ctx.Logger().Info("DKIM module migration from v2 to v3 completed successfully")
	return paramsCollection.Set(ctx, existingParams)
}
