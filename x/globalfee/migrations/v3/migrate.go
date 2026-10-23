package v3

import (
	"context"

	corestore "cosmossdk.io/core/store"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/globalfee/types"
)

type ParamsSetter interface {
	SetParams(context.Context, types.Params) error
}

func MigrateStore(ctx sdk.Context, setter ParamsSetter, legacyParams corestore.KVStoreService, amino *codec.LegacyAmino) error {
	store := legacyParams.OpenKVStore(ctx)
	params := types.DefaultParams()
	for _, field := range []struct {
		key   string
		value interface{}
	}{
		{"globalfee/MinimumGasPricesParam", &params.MinimumGasPrices},
		{"globalfee/BypassMinFeeMsgTypes", &params.BypassMinFeeMsgTypes},
		{"globalfee/MaxTotalBypassMinFeeMsgGasUsage", &params.MaxTotalBypassMinFeeMsgGasUsage},
	} {
		bz, err := store.Get([]byte(field.key))
		if err != nil {
			return err
		}
		if bz != nil {
			if err := amino.UnmarshalJSON(bz, field.value); err != nil {
				return err
			}
		}
	}
	return setter.SetParams(ctx, params)
}
