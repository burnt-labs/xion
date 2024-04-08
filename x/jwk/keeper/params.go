package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/jwk/types"
)

// GetParams get all parameters as types.Params
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	return types.NewParams(k.GetTimeOffset(ctx), k.GetDeploymentGas(ctx))
}

// SetParams set the params
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramspace.SetParamSet(ctx, &params)
}

func (k Keeper) GetTimeOffset(ctx sdk.Context) uint64 {
	var to uint64
	k.paramspace.Get(ctx, types.ParamStoreKeyTimeOffset, &to)
	return to
}

func (k Keeper) GetDeploymentGas(ctx sdk.Context) uint64 {
	var dg uint64
	k.paramspace.Get(ctx, types.ParamStoreKeyDeploymentGas, &dg)
	return dg
}
