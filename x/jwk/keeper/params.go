package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/jwk/types"
)

// GetParams get all parameters as types.Params
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ParamsKey)
	if bz == nil {
		return types.DefaultParams()
	}
	var params types.Params
	k.cdc.MustUnmarshal(bz, &params)
	return params
}

// SetParams validates and stores the params.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) error {
	if err := params.Validate(); err != nil {
		return err
	}
	ctx.KVStore(k.storeKey).Set(types.ParamsKey, k.cdc.MustMarshal(&params))
	return nil
}

func (k Keeper) GetTimeOffset(ctx sdk.Context) uint64 {
	return k.GetParams(ctx).TimeOffset
}

func (k Keeper) GetDeploymentGas(ctx sdk.Context) uint64 {
	return k.GetParams(ctx).DeploymentGas
}
