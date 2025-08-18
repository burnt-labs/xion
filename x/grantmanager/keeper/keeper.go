package keeper

import (
	"context"
	"unsafe"

	"cosmossdk.io/core/store"
	feegrantkeeper "cosmossdk.io/x/feegrant/keeper"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	"github.com/cosmos/cosmos-sdk/types/kv"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
)

const gasCostPerIteration = uint(20)

type Keeper struct {
	authzStoreService    store.KVStoreService
	feegrantStoreService store.KVStoreService
	authzKeeper          authzkeeper.Keeper
	feegrantKeeper       feegrantkeeper.Keeper
}

func NewKeeper(authzStoreService, feegrantStoreService store.KVStoreService, authzkeeper authzkeeper.Keeper, feegrantkeeper feegrantkeeper.Keeper) Keeper {
	return Keeper{
		authzStoreService:    authzStoreService,
		feegrantStoreService: feegrantStoreService,
		authzKeeper:          authzkeeper,
		feegrantKeeper:       feegrantkeeper,
	}
}

func (k Keeper) RevokeAuthzGrants(ctx context.Context, granter sdk.AccAddress, limit int) error {

	store := runtime.KVStoreAdapter(k.authzStoreService.OpenKVStore(ctx))
	// iterate by granter

	iter := storetypes.KVStorePrefixIterator(store, grantStoreKey(nil, granter, ""))
	defer iter.Close()

	count := 0
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	for ; iter.Valid() && count < limit; iter.Next() {
		sdkCtx.GasMeter().ConsumeGas(storetypes.Gas(gasCostPerIteration), "delete grant")
		count++
		granter, grantee, msgType := parseGrantStoreKey(iter.Key())
		err := k.authzKeeper.DeleteGrant(ctx, grantee, granter, msgType)
		if err != nil {
			return err
		}

	}
	return nil

}

// RevokeFeegrantAllowances
func (k Keeper) RevokeFeegrantAllowances(ctx context.Context, granter sdk.AccAddress, limit int) error {
	return nil

}

// grantStoreKey - return authorization store key
// Items are stored with the following key: values
//
// - 0x01<granterAddressLen (1 Byte)><granterAddress_Bytes><granteeAddressLen (1 Byte)><granteeAddress_Bytes><msgType_Bytes>: Grant
func grantStoreKey(grantee, granter sdk.AccAddress, msgType string) []byte {
	m := UnsafeStrToBytes(msgType)
	granter = address.MustLengthPrefix(granter)
	grantee = address.MustLengthPrefix(grantee)
	key := sdk.AppendLengthPrefixedBytes(authzkeeper.GrantKey, granter, grantee, m)

	return key
}

func UnsafeStrToBytes(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

// UnsafeBytesToStr is meant to make a zero allocation conversion
// from []byte -> string to speed up operations, it is not meant
// to be used generally, but for a specific pattern to delete keys
// from a map.
func UnsafeBytesToStr(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}

// parseGrantStoreKey - split granter, grantee address and msg type from the authorization key
func parseGrantStoreKey(key []byte) (granterAddr, granteeAddr sdk.AccAddress, msgType string) {
	// key is of format:
	// 0x01<granterAddressLen (1 Byte)><granterAddress_Bytes><granteeAddressLen (1 Byte)><granteeAddress_Bytes><msgType_Bytes>

	granterAddrLen, granterAddrLenEndIndex := sdk.ParseLengthPrefixedBytes(key, 1, 1) // ignore key[0] since it is a prefix key
	granterAddr, granterAddrEndIndex := sdk.ParseLengthPrefixedBytes(key, granterAddrLenEndIndex+1, int(granterAddrLen[0]))

	granteeAddrLen, granteeAddrLenEndIndex := sdk.ParseLengthPrefixedBytes(key, granterAddrEndIndex+1, 1)
	granteeAddr, granteeAddrEndIndex := sdk.ParseLengthPrefixedBytes(key, granteeAddrLenEndIndex+1, int(granteeAddrLen[0]))

	kv.AssertKeyAtLeastLength(key, granteeAddrEndIndex+1)
	return granterAddr, granteeAddr, UnsafeBytesToStr(key[(granteeAddrEndIndex + 1):])
}
