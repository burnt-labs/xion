package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/jwk/types"
)

// SetAudience set a specific audience in the store from its index
func (k Keeper) SetAudience(ctx sdk.Context, audience types.Audience) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AudienceKeyPrefix))
	b := k.cdc.MustMarshal(&audience)
	store.Set(types.AudienceKey(
		audience.Aud,
	), b)
}

// GetAudience returns a audience from its index
func (k Keeper) GetAudience(
	ctx sdk.Context,
	aud string,
) (val types.Audience, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AudienceKeyPrefix))

	b := store.Get(types.AudienceKey(
		aud,
	))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveAudience removes a audience from the store
func (k Keeper) RemoveAudience(
	ctx sdk.Context,
	aud string,
) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AudienceKeyPrefix))
	store.Delete(types.AudienceKey(
		aud,
	))
}

// GetAllAudience returns all audience
func (k Keeper) GetAllAudience(ctx sdk.Context) (list []types.Audience) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AudienceKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Audience
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}
