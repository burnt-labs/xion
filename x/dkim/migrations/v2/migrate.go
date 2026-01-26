package v2

import (
	"cosmossdk.io/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/dkim/types"
)

// MigrateStore performs in-place migrations for the DKIM module from v1 to v2.
// This migration:
// 1. Adds the new public_input_indices field to existing params with default values
// 2. Clears PoseidonHash from all stored DKIM records (now computed dynamically)
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

	// Step 2: Clear PoseidonHash from all DKIM records
	ctx.Logger().Info("Clearing PoseidonHash from DKIM records (now computed dynamically)")

	iter, err := dkimPubKeys.Iterate(ctx, nil)
	if err != nil {
		return err
	}
	defer iter.Close()

	kvs, err := iter.KeyValues()
	if err != nil {
		return err
	}

	for _, kv := range kvs {
		// Only update if PoseidonHash is set
		if len(kv.Value.PoseidonHash) > 0 {
			// Clear PoseidonHash and re-save the record
			updatedKey := types.DkimPubKey{
				Domain:       kv.Value.Domain,
				PubKey:       kv.Value.PubKey,
				Selector:     kv.Value.Selector,
				Version:      kv.Value.Version,
				KeyType:      kv.Value.KeyType,
				PoseidonHash: nil, // Clear the hash
			}
			//nolint:govet // copylocks: unavoidable when storing protobuf messages in collections.Map
			if err := dkimPubKeys.Set(ctx, kv.Key, updatedKey); err != nil {
				return err
			}
		}
	}

	ctx.Logger().Info("DKIM module migration from v1 to v2 completed successfully")
	return nil
}
