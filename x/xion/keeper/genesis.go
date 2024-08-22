package keeper

import (
	"encoding/binary"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/xion/types"
)

// InitGenesis initializes the bank module's state from a given genesis state.
func (k Keeper) InitGenesis(ctx sdk.Context, genState *types.GenesisState) {
	k.OverwritePlatformPercentage(ctx, genState.PlatformPercentage)
}

// ExportGenesis returns the bank module's genesis state.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	bz := ctx.KVStore(k.storeKey).Get(types.PlatformPercentageKey)
	platformPercentage := binary.BigEndian.Uint32(bz)
	rv := types.NewGenesisState(platformPercentage)
	return rv
}
