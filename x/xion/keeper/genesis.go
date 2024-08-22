package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/xion/types"
)

// InitGenesis initializes the bank module's state from a given genesis state.
func (k Keeper) InitGenesis(ctx sdk.Context, genState *types.GenesisState) {
	k.OverwritePlatformPercentage(ctx, genState.PlatformPercentage)
}

// ExportGenesis returns the bank module's genesis state.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	rv := types.NewGenesisState(
		//integer overflow conversion uint64 -> uint32 (gosec)
		//nolint:gosec
		uint32(k.GetPlatformPercentage(ctx).Uint64()),
	)
	return rv
}
