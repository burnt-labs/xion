package keeper

import (
	abci "github.com/cometbft/cometbft/abci/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/abstractaccount/types"
)

func (k Keeper) InitGenesis(ctx sdk.Context, gs *types.GenesisState) []abci.ValidatorUpdate {
	if err := k.SetParams(ctx, gs.Params); err != nil {
		panic(err)
	}

	err := k.SetNextAccountID(ctx, gs.NextAccountId)
	if err != nil {
		panic(err)
	}

	return []abci.ValidatorUpdate{}
}

func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	params, err := k.GetParams(ctx)
	if err != nil {
		panic(err)
	}

	nextAccountID, err := k.GetNextAccountID(ctx)
	if err != nil {
		panic(err)
	}
	return &types.GenesisState{
		Params:        params,
		NextAccountId: nextAccountID,
	}
}
