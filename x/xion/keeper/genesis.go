package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/xion/types"
)

// InitGenesis initializes the bank module's state from a given genesis state.
func (k Keeper) InitGenesis(ctx sdk.Context, genState *types.GenesisState) {
	k.OverwritePlatformPercentage(ctx, genState.PlatformPercentage)
	err := k.OverwritePlatformMinimum(ctx, genState.PlatformMinimums)
	if err != nil {
		panic(err)
	}
}

// ExportGenesis returns the bank module's genesis state.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	platformPercentage := uint32(k.GetPlatformPercentage(ctx).Uint64())

	platformMinimums, err := k.GetPlatformMinimums(ctx)
	if err != nil {
		panic(err)
	}
	rv := types.NewGenesisState(platformPercentage, platformMinimums)
	return rv
}
