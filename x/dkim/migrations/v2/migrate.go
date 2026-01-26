package v2

import (
	"cosmossdk.io/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/dkim/types"
)

// MigrateStore performs in-place params migrations to add PublicInputIndices.
// This migration adds the new public_input_indices field to existing params
// with default values.
func MigrateStore(ctx sdk.Context, paramsCollection collections.Item[types.Params]) error {
	ctx.Logger().Info("Running DKIM module migration from v1 to v2")

	// Get existing params
	existingParams, err := paramsCollection.Get(ctx)
	if err != nil {
		// If params don't exist, set defaults
		ctx.Logger().Info("No existing params found, setting defaults")
		return paramsCollection.Set(ctx, types.DefaultParams())
	}

	// Add default PublicInputIndices to existing params
	existingParams.PublicInputIndices = types.DefaultPublicInputIndices()

	ctx.Logger().Info("Setting updated params with PublicInputIndices")
	return paramsCollection.Set(ctx, existingParams)
}
