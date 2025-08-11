package indexer

import (
	"context"
	"reflect"
	"unsafe"

	"cosmossdk.io/collections"
	core "cosmossdk.io/collections/corecompat"
	"cosmossdk.io/collections/indexes"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/kv"
	authz "github.com/cosmos/cosmos-sdk/x/authz"
)

var (
	AuthzPrefix             = collections.NewPrefix(0)
	AuthzGranteeIndexPrefix = collections.NewPrefix(1)
)

type authzIndexes struct {
	Grantee *indexes.Multi[sdk.AccAddress, collections.Triple[sdk.AccAddress, sdk.AccAddress, string], authz.Grant]
}

type AuthzHandler struct {
	kvStoreService core.KVStoreService
	cdc            codec.Codec
	Schema         collections.Schema
	// key: (granter, grantee)
	// value: authorization
	// secondary index: grantee
	Authorizations *collections.IndexedMap[collections.Triple[sdk.AccAddress, sdk.AccAddress, string], authz.Grant, authzIndexes]
}

func newAuthzIndexes(sb *collections.SchemaBuilder) authzIndexes {
	indexByGrantee := indexes.NewMulti[
		sdk.AccAddress,
		collections.Triple[sdk.AccAddress, sdk.AccAddress, string],
		authz.Grant,
	](
		sb,
		AuthzGranteeIndexPrefix,
		"authz_by_grantee",
		sdk.AccAddressKey, // reference key code (grantee sdk.AccAddresss)
		collections.TripleKeyCodec(sdk.AccAddressKey, sdk.AccAddressKey, collections.StringKey),
		func(pk collections.Triple[sdk.AccAddress, sdk.AccAddress, string], value authz.Grant) (sdk.AccAddress, error) {
			return pk.K2(), nil
		},
	)

	return authzIndexes{
		Grantee: indexByGrantee,
	}
}
func NewAuthzHandler(kvStoreService core.KVStoreService, cdc codec.Codec) (*AuthzHandler, error) {
	sb := collections.NewSchemaBuilder(kvStoreService)

	authzIndexes := newAuthzIndexes(sb)

	// Create the indexed map with the indexes
	authorizations := collections.NewIndexedMap(
		sb,
		AuthzPrefix,
		"authorizations", // name of the collection
		collections.TripleKeyCodec(sdk.AccAddressKey, sdk.AccAddressKey, collections.StringKey),
		codec.CollValue[authz.Grant](cdc),
		authzIndexes,
	)

	schema, err := sb.Build()
	if err != nil {
		return nil, err
	}

	return &AuthzHandler{
		Schema:         schema,
		cdc:            cdc,
		kvStoreService: kvStoreService,
		Authorizations: authorizations,
	}, nil
}

func (ah *AuthzHandler) GetGrant(ctx context.Context, granter, grantee sdk.AccAddress, msgType string) (authz.Grant, error) {
	key := collections.Join3(granter, grantee, msgType)
	return ah.Authorizations.Get(ctx, key)
}

func (ah *AuthzHandler) SetGrant(ctx context.Context, granter, grantee sdk.AccAddress, msgType string, grant authz.Grant) error {
	key := collections.Join3(granter, grantee, msgType)
	return ah.Authorizations.Set(ctx, key, grant)
}

func (ah *AuthzHandler) HandleUpdate(ctx context.Context, pair *storetypes.StoreKVPair) error {
	granterAddr, granteeAddr, msgType := parseGrantStoreKey(pair.Key)
	if pair.Delete {
		if has, err := ah.Authorizations.Has(ctx, collections.Join3(granterAddr, granteeAddr, msgType)); err != nil {
			return err
		} else if !has {
			return ErrGrantNotFound
		}
		return ah.Authorizations.Remove(ctx, collections.Join3(granterAddr, granteeAddr, msgType))
	}

	grant := authz.Grant{}
	err := ah.cdc.Unmarshal(pair.Value, &grant)
	if err != nil {
		return err
	}
	return ah.SetGrant(ctx, granterAddr, granteeAddr, msgType, grant)
}

// copied from x/authz/keeper
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

// UnsafeStrToBytes uses unsafe to convert string into byte array. Returned bytes
// must not be altered after this function is called as it will cause a segmentation fault.
func UnsafeStrToBytes(s string) []byte {
	var buf []byte
	sHdr := (*reflect.StringHeader)(unsafe.Pointer(&s))
	bufHdr := (*reflect.SliceHeader)(unsafe.Pointer(&buf))
	bufHdr.Data = sHdr.Data
	bufHdr.Cap = sHdr.Len
	bufHdr.Len = sHdr.Len
	return buf
}

// UnsafeBytesToStr is meant to make a zero allocation conversion
// from []byte -> string to speed up operations, it is not meant
// to be used generally, but for a specific pattern to delete keys
// from a map.
func UnsafeBytesToStr(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
