package indexer

import (
	"context"
	"log/slog"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	feegrant "cosmossdk.io/x/feegrant"
	indexerfeegrant "github.com/burnt-labs/xion/indexer/feegrant"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
)

type feegrantQuerier struct {
	feegrantHandler *FeeGrantHandler
	cdc             codec.Codec
	addrCodec       address.Codec
}

var _ indexerfeegrant.QueryServer = &feegrantQuerier{}

func NewFeegrantQuerier(handler *FeeGrantHandler, cdc codec.Codec, addrCodec address.Codec) indexerfeegrant.QueryServer {
	return &feegrantQuerier{handler, cdc, addrCodec}
}

func (fq *feegrantQuerier) Allowance(ctx context.Context, req *indexerfeegrant.QueryAllowanceRequest) (*indexerfeegrant.QueryAllowanceResponse, error) {
	return nil, nil
}

func (fq *feegrantQuerier) Allowances(ctx context.Context, req *indexerfeegrant.QueryAllowancesRequest) (*indexerfeegrant.QueryAllowancesResponse, error) {
	grantee, err := fq.addrCodec.StringToBytes(req.Grantee)
	if err != nil {
		return nil, err
	}
	granteeAddr := sdk.AccAddress(grantee)

	allowances, pageRes, err := query.CollectionPaginate(
		ctx,
		fq.feegrantHandler.FeeAllowances.Indexes.Grantee,
		req.Pagination,
		// key is the reverse key pair reference
		// value is empty because index only stores keys in a KeySet
		// final key in the callback is (grantee, granter)
		func(key collections.Pair[sdk.AccAddress, sdk.AccAddress], value collections.NoValue) (*feegrant.Grant, error) {
			grant, err := fq.feegrantHandler.FeeAllowances.Get(ctx, collections.Join(key.K2(), key.K1()))
			if err != nil {
				return nil, err
			}
			return &grant, nil
		},
		query.WithCollectionPaginationPairPrefix[sdk.AccAddress, sdk.AccAddress](granteeAddr),
	)
	if err != nil {
		return nil, err
	}
	slog.Info("feegrant_querier", "allowances", len(allowances))
	return &indexerfeegrant.QueryAllowancesResponse{
		Allowances: allowances,
		Pagination: pageRes,
	}, nil
}

func (fq *feegrantQuerier) AllowancesByGranter(ctx context.Context, req *indexerfeegrant.QueryAllowancesByGranterRequest) (*indexerfeegrant.QueryAllowancesByGranterResponse, error) {
	granter, err := fq.addrCodec.StringToBytes(req.Granter)
	if err != nil {
		return nil, err
	}
	granterAddr := sdk.AccAddress(granter)

	allowances, pageRes, err := query.CollectionPaginate(
		ctx,
		fq.feegrantHandler.FeeAllowances,
		req.Pagination,
		// iterate over the original key space (granter, grantee)
		// value is the grant
		// final key in the callback is (granter, grantee)
		func(key collections.Pair[sdk.AccAddress, sdk.AccAddress], grant feegrant.Grant) (*feegrant.Grant, error) {
			return &grant, nil
		},
		query.WithCollectionPaginationPairPrefix[sdk.AccAddress, sdk.AccAddress](granterAddr),
	)
	if err != nil {
		return nil, err
	}
	slog.Info("feegrant_querier", "allowances", len(allowances))
	return &indexerfeegrant.QueryAllowancesByGranterResponse{
		Allowances: allowances,
		Pagination: pageRes,
	}, nil
}
