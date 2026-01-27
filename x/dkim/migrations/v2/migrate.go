package v2

import (
	"cosmossdk.io/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/dkim/types"
)

// MigrateStore performs in-place migrations for the DKIM module from v1 to v2.
// This migration adds the new public_input_indices field to existing params with default values.
func MigrateStore(
	ctx sdk.Context,
	paramsCollection collections.Item[types.Params],
	dkimPubKeys collections.Map[collections.Pair[string, string], types.DkimPubKey],
) error {
	ctx.Logger().Info("Running DKIM module migration from v1 to v2")

	// Step 1: Migrate params - add PublicInputIndices
	existingParams, err := paramsCollection.Get(ctx)
	if err != nil {
		// If params don't exist, set defaults
		ctx.Logger().Info("No existing params found, setting defaults")
		if err := paramsCollection.Set(ctx, types.DefaultParams()); err != nil {
			return err
		}
	} else {
		// Add default PublicInputIndices to existing params
		existingParams.PublicInputIndices = types.DefaultPublicInputIndices()
		ctx.Logger().Info("Setting updated params with PublicInputIndices")
		if err := paramsCollection.Set(ctx, existingParams); err != nil {
			return err
		}
	}

	ctx.Logger().Info("DKIM module migration from v1 to v2 completed successfully")
	return nil
}
