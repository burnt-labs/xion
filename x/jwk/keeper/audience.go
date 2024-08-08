package keeper

import (
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/jwk/types"
)

func (k Keeper) SetAudienceClaim(ctx sdk.Context, hash []byte, signer sdk.AccAddress) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AudienceClaimKeyPrefix))
	audClaim := types.AudienceClaim{Signer: signer.String()}
	b := k.cdc.MustMarshal(&audClaim)
	store.Set(types.AudienceClaimKey(hash), b)
}

func (k Keeper) GetAudienceClaim(ctx sdk.Context, hash []byte) (val types.AudienceClaim, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AudienceClaimKeyPrefix))

	b := store.Get(types.AudienceClaimKey(
		hash,
	))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveAudienceClaim removes an audience claim from the store
func (k Keeper) RemoveAudienceClaim(
	ctx sdk.Context,
	audHash []byte,
) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.AudienceClaimKeyPrefix))
	store.Delete(types.AudienceClaimKey(
		audHash,
	))
}

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

// RemoveAudience removes an audience from the store
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
	iterator := storetypes.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Audience
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}
