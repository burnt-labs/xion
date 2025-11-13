package indexer

import (
	"context"

	"cosmossdk.io/collections"
	core "cosmossdk.io/collections/corecompat"
	"cosmossdk.io/collections/indexes"
	storetypes "cosmossdk.io/store/types"
	"cosmossdk.io/x/feegrant"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	FeeGrantsPrefix            = collections.NewPrefix(0)
	FeeGrantGranteeIndexPrefix = collections.NewPrefix(1)
)

type feegrantIndexes struct {
	Grantee *indexes.ReversePair[sdk.AccAddress, sdk.AccAddress, feegrant.Grant]
}

type FeeGrantHandler struct {
	kvStoreService core.KVStoreService
	cdc            codec.Codec
	Schema         collections.Schema
	// key: (granter, grantee)
	// value: authorization
	// secondary index: grantee
	FeeAllowances *collections.IndexedMap[collections.Pair[sdk.AccAddress, sdk.AccAddress], feegrant.Grant, feegrantIndexes]
}

func newFeeGrantIndexes(sb *collections.SchemaBuilder) feegrantIndexes {
	return feegrantIndexes{
		Grantee: indexes.NewReversePair[feegrant.Grant](
			sb,
			FeeGrantGranteeIndexPrefix,
			"feegrant_by_grantee",
			collections.PairKeyCodec(sdk.AccAddressKey, sdk.AccAddressKey),
		),
	}
}

func NewFeeGrantHandler(kvStoreService core.KVStoreService, cdc codec.Codec) (*FeeGrantHandler, error) {
	sb := collections.NewSchemaBuilder(kvStoreService)

	feegrantIndexes := newFeeGrantIndexes(sb)

	// Create the indexed map with the indexes
	authorizations := collections.NewIndexedMap(
		sb,
		FeeGrantsPrefix,
		"feegrant", // name of the collection
		collections.PairKeyCodec(sdk.AccAddressKey, sdk.AccAddressKey),
		codec.CollValue[feegrant.Grant](cdc),
		feegrantIndexes,
	)

	schema, err := sb.Build()
	if err != nil {
		return nil, err
	}

	return &FeeGrantHandler{
		Schema:         schema,
		kvStoreService: kvStoreService,
		FeeAllowances:  authorizations,
		cdc:            cdc,
	}, nil
}

func (ah *FeeGrantHandler) GetGrant(ctx context.Context, granter, grantee sdk.AccAddress) (feegrant.Grant, error) {
	key := collections.Join(granter, grantee)
	return ah.FeeAllowances.Get(ctx, key)
}

func (ah *FeeGrantHandler) SetGrant(ctx context.Context, granter, grantee sdk.AccAddress, grant feegrant.Grant) error {
	key := collections.Join(granter, grantee)
	return ah.FeeAllowances.Set(ctx, key, grant)
}

func (ah *FeeGrantHandler) HandleUpdate(ctx context.Context, pair *storetypes.StoreKVPair) error {
	granterAddrBz, granteeAddrBz := feegrant.ParseAddressesFromFeeAllowanceKey(pair.Key)
	granterAddr := sdk.AccAddress(granterAddrBz)
	granteeAddr := sdk.AccAddress(granteeAddrBz)
	if pair.Delete {
		if has, err := ah.FeeAllowances.Has(ctx, collections.Join(granterAddr, granteeAddr)); err != nil {
			return err
		} else if !has {
			// Graceful handling: deleting a non-existent allowance is a no-op, not an error
			// This ensures the indexer remains robust during edge cases
			return nil
		}
		return ah.FeeAllowances.Remove(ctx, collections.Join(granterAddr, granteeAddr))
	}

	feegrant := feegrant.Grant{}
	err := ah.cdc.Unmarshal(pair.Value, &feegrant)
	if err != nil {
		return err
	}
	return ah.SetGrant(ctx, granterAddr, granteeAddr, feegrant)
}
