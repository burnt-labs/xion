package v3

import (
	"cosmossdk.io/collections"
	"cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/dkim/types"
)

// MigrateStore performs in-place migrations for the DKIM module from v2 to v3.
// This migration backfills MinRsaKeyBits for chains that upgraded through v28
// where the field was introduced but the v1→v2 migration did not set it
// (because the field did not exist at that time).
func MigrateStore(
	ctx sdk.Context,
	paramsCollection collections.Item[types.Params],
) error {
	ctx.Logger().Info("Running DKIM module migration from v2 to v3")

	params, err := paramsCollection.Get(ctx)
	if err != nil {
		if errors.IsOf(err, collections.ErrNotFound) {
			ctx.Logger().Info("No existing DKIM params found, skipping MinRsaKeyBits backfill")
			return nil
		}
		return err
	}

	if params.MinRsaKeyBits == 0 {
		params.MinRsaKeyBits = types.DefaultMinRSAKeyBits
		ctx.Logger().Info("Backfilling DKIM MinRsaKeyBits", "value", params.MinRsaKeyBits)
		if err := paramsCollection.Set(ctx, params); err != nil {
			return err
		}
	}

	ctx.Logger().Info("DKIM module migration from v2 to v3 completed successfully")
	return nil
}
