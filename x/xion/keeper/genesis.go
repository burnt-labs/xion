package keeper

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/xion/types"
)

// InitGenesis initializes the bank module's state from a given genesis state.
func (k Keeper) InitGenesis(ctx sdk.Context, genState *types.GenesisState) {
	k.OverwritePlatformPercentage(ctx, genState.PlatformPercentage)
}

// ExportGenesis returns the bank module's genesis state.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	platformPercentage := k.GetPlatformPercentage(ctx).Uint64()
	rv := types.NewGenesisState(
		uint32(platformPercentage),
	)
	return rv
}
