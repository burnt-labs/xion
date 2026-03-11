package v2

import (
	"cosmossdk.io/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/zk/types"
)

// MigrateStore performs in-place migrations for the zk module from v1 to v2.
// This migration updates the vkey with id 1 to the new vkey to avoid gov prop.
func MigrateStore(
	ctx sdk.Context,
	vkeys collections.Map[uint64, types.VKey],
) error {
	ctx.Logger().Info("Running zk module migration from v1 to v2")
	// update the vkey to the new vkey value
	defaultVkeys := types.DefaultGenesisState().Vkeys
	var defaultVkey types.VKey
	for _, vk := range defaultVkeys {
		if vk.Id == 1 {
			defaultVkey = vk.Vkey
			break
		}
	}
	if len(defaultVkey.KeyBytes) != 0 {
		ctx.Logger().Info("Setting updated vkey with default vkey for id 1")
		if err := vkeys.Set(ctx, 1, defaultVkey); err != nil {
			return err
		}
	} else {
		ctx.Logger().Info("WARNING: default vkey with id 1 not found in DefaultGenesisState; skipping vkey update")
	}

	ctx.Logger().Info("ZK module migration from v1 to v2 completed successfully")
	return nil
}
