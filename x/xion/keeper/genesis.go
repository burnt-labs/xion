package keeper

import (
	"github.com/burnt-labs/xion/x/xion/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// InitGenesis initializes the bank module's state from a given genesis state.
func (k Keeper) InitGenesis(ctx sdk.Context, genState *types.GenesisState) {
	k.OverwritePlatformPercentage(ctx, genState.PlatformPercentage)
}

// ExportGenesis returns the bank module's genesis state.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	rv := types.NewGenesisState(
		uint32(k.GetPlatformPercentage(ctx).Uint64()),
	)
	return rv
}
