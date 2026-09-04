package v2

import (
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/abstractaccount/types"
)

// MigrateStore performs in-place params migrations of
// BypassMinFeeMsgTypes and MaxTotalBypassMinFeeMsgGasUsage
// from app.toml to globalfee params.
func MigrateStore(ctx sdk.Context, key storetypes.StoreKey, cdc codec.BinaryCodec) error {
	store := ctx.KVStore(key)

	params, err := getParams(ctx, store, cdc)
	if err != nil {
		return err
	}

	return setParams(ctx, store, cdc, params)
}

func getParams(_ sdk.Context, store storetypes.KVStore, cdc codec.BinaryCodec) (*types.Params, error) {
	bz := store.Get(types.KeyParams)
	if bz == nil {
		params := types.DefaultParams()
		return params, nil
	}

	var params types.Params
	if err := cdc.Unmarshal(bz, &params); err != nil {
		return nil, types.ErrParsingParams.Wrap(err.Error())
	}

	return &params, nil
}

func setParams(_ sdk.Context, store storetypes.KVStore, cdc codec.BinaryCodec, params *types.Params) error {
	bz, err := cdc.Marshal(params)
	if err != nil {
		return types.ErrParsingParams.Wrap(err.Error())
	}
	store.Set(types.KeyParams, bz)

	return nil
}
