package indexer

import (
	"context"
	"log/slog"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	indexerauthz "github.com/burnt-labs/xion/indexer/authz"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	authz "github.com/cosmos/cosmos-sdk/x/authz"
	"github.com/gogo/status"
	"google.golang.org/grpc/codes"
)

type authzQuerier struct {
	authzHandler *AuthzHandler
	cdc          codec.Codec
	addrCodec    address.Codec
}

var _ indexerauthz.QueryServer = &authzQuerier{}

func NewAuthzQuerier(handler *AuthzHandler, cdc codec.Codec, addrCodec address.Codec) indexerauthz.QueryServer {
	return &authzQuerier{handler, cdc, addrCodec}
}

func (aq *authzQuerier) Grants(ctx context.Context, req *indexerauthz.QueryGrantsRequest) (*indexerauthz.QueryGrantsResponse, error) {
	return nil, nil
}

func (aq *authzQuerier) GranterGrants(ctx context.Context, req *indexerauthz.QueryGranterGrantsRequest) (*indexerauthz.QueryGranterGrantsResponse, error) {
	slog.Info("authz_querier", "granter", req.Granter)

	granter, err := aq.addrCodec.StringToBytes(req.Granter)
	if err != nil {
		return nil, err
	}
	granterAddr := sdk.AccAddress(granter)
	slog.Info("authz_querier", "step", "1")

	grants, pageRes, err := query.CollectionPaginate(
		ctx,
		aq.authzHandler.Authorizations,
		req.Pagination,
		func(primaryKey collections.Triple[sdk.AccAddress, sdk.AccAddress, string], grant authz.Grant) (*authz.GrantAuthorization, error) {
			auth1, err := grant.GetAuthorization()
			if err != nil {
				return nil, err
			}

			anyValue, err := codectypes.NewAnyWithValue(auth1)
			if err != nil {
				return nil, status.Errorf(codes.Internal, err.Error())
			}
			granter, err := aq.addrCodec.BytesToString(primaryKey.K1())
			if err != nil {
				return nil, err
			}

			grantee, err := aq.addrCodec.BytesToString(primaryKey.K2())
			if err != nil {
				return nil, err
			}

			return &authz.GrantAuthorization{
				Granter:       granter,
				Grantee:       grantee,
				Authorization: anyValue,
				Expiration:    grant.Expiration,
			}, nil
		},
		WithCollectionPaginationTriplePrefix[sdk.AccAddress, sdk.AccAddress, string](granterAddr),
	)
	if err != nil {
		return nil, err
	}
	return &indexerauthz.QueryGranterGrantsResponse{
		Grants:     grants,
		Pagination: pageRes,
	}, nil
}

func WithCollectionPaginationTriplePrefix[K1, K2, K3 any](prefix K1) func(o *query.CollectionsPaginateOptions[collections.Triple[K1, K2, K3]]) {
	return func(o *query.CollectionsPaginateOptions[collections.Triple[K1, K2, K3]]) {
		prefix := collections.TriplePrefix[K1, K2, K3](prefix)
		o.Prefix = &prefix
	}
}

func (aq *authzQuerier) GranteeGrants(ctx context.Context, req *indexerauthz.QueryGranteeGrantsRequest) (*indexerauthz.QueryGranteeGrantsResponse, error) {
	slog.Info("authz_querier", "grantee", req.Grantee)
	grantee, err := aq.addrCodec.StringToBytes(req.Grantee)
	if err != nil {
		return nil, err
	}
	granteeAddr := sdk.AccAddress(grantee)

	grants, pageRes, err := query.CollectionPaginate(
		ctx,
		aq.authzHandler.Authorizations.Indexes.Grantee,
		req.Pagination,
		// key is the index key (refKey, primaryKey)
		// primary key is a composite key of (granter, grantee, msgType)
		// value is empty because index only stores keys in a KeySet
		// final key in the callback is (refKey, [granter, grantee, msgType])
		func(key collections.Pair[sdk.AccAddress, collections.Triple[sdk.AccAddress, sdk.AccAddress, string]], value collections.NoValue) (*authz.GrantAuthorization, error) {
			slog.Info("authz_querier", "key", key.K2(), "grantee", granteeAddr.String(), "granter", key.K1().String())

			primaryKey := key.K2()
			// need to use .K2() because the index is reversed
			grant, err := aq.authzHandler.Authorizations.Get(ctx, primaryKey)
			if err != nil {
				return nil, err
			}

			auth1, err := grant.GetAuthorization()
			if err != nil {
				return nil, err
			}

			anyValue, err := codectypes.NewAnyWithValue(auth1)
			if err != nil {
				return nil, status.Errorf(codes.Internal, err.Error())
			}
			granter, err := aq.addrCodec.BytesToString(primaryKey.K1())
			if err != nil {
				return nil, err
			}

			return &authz.GrantAuthorization{
				Granter:       granter,
				Grantee:       req.Grantee,
				Authorization: anyValue,
				Expiration:    grant.Expiration,
			}, nil
		},
		// stablish the prefix of the index
		// {grantee}:{granter}
		query.WithCollectionPaginationPairPrefix[sdk.AccAddress, collections.Triple[sdk.AccAddress, sdk.AccAddress, string]](granteeAddr),
	)
	if err != nil {
		return nil, err
	}
	slog.Info("authz_querier", "grants", len(grants))
	return &indexerauthz.QueryGranteeGrantsResponse{
		Grants:     grants,
		Pagination: pageRes,
	}, nil
}
