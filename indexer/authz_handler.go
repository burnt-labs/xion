package indexer

import (
	"context"
	"unsafe"

	"cosmossdk.io/collections"
	core "cosmossdk.io/collections/corecompat"
	"cosmossdk.io/collections/indexes"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authz "github.com/burnt-labs/xion/x/authz"
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
			// Graceful handling: deleting a non-existent grant is a no-op, not an error
			// This ensures the indexer remains robust during edge cases
			return nil
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

// parseGrantStoreKey - split granter, grantee address and msg type from the authorization key
// This decodes the collections.TripleKeyCodec format used by the IndexedMap
func parseGrantStoreKey(key []byte) (granterAddr, granteeAddr sdk.AccAddress, msgType string) {
	// key is of format:
	// 0x01<triple_key_encoded>
	// where triple is encoded by collections.TripleKeyCodec(sdk.AccAddressKey, sdk.AccAddressKey, collections.StringKey)

	// Skip the prefix byte
	if len(key) < 1 {
		return nil, nil, ""
	}
	keyWithoutPrefix := key[1:]

	// Decode using the triple key codec
	codec := collections.TripleKeyCodec(sdk.AccAddressKey, sdk.AccAddressKey, collections.StringKey)
	_, triple, err := codec.Decode(keyWithoutPrefix)
	if err != nil {
		// Return empty values if decode fails
		return nil, nil, ""
	}

	return triple.K1(), triple.K2(), triple.K3()
}

// UnsafeStrToBytes uses unsafe to convert string into byte array. Returned bytes
// must not be altered after this function is called as it will cause a segmentation fault.
func UnsafeStrToBytes(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

// UnsafeBytesToStr is meant to make a zero allocation conversion
// from []byte -> string to speed up operations, it is not meant
// to be used generally, but for a specific pattern to delete keys
// from a map.
func UnsafeBytesToStr(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
