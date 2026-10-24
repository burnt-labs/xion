package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	corestore "cosmossdk.io/core/store"

	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/burnt-labs/xion/x/globalfee/types"
)

type Keeper struct {
	cdc       codec.BinaryCodec
	Schema    collections.Schema
	Params    collections.Item[types.Params]
	authority string
}

func NewKeeper(cdc codec.BinaryCodec, storeService corestore.KVStoreService, authority string) Keeper {
	sb := collections.NewSchemaBuilder(storeService)
	k := Keeper{
		cdc:       cdc,
		Params:    collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		authority: authority,
	}
	var err error
	k.Schema, err = sb.Build()
	if err != nil {
		panic(err)
	}
	return k
}

func (k Keeper) GetAuthority() string {
	return k.authority
}

func (k Keeper) GetParams(ctx context.Context) (types.Params, error) {
	params, err := k.Params.Get(ctx)
	if errors.Is(err, collections.ErrNotFound) {
		return types.DefaultParams(), nil
	}
	return params, err
}

func (k Keeper) SetParams(ctx context.Context, params types.Params) error {
	if err := params.ValidateBasic(); err != nil {
		return err
	}
	return k.Params.Set(ctx, params)
}
