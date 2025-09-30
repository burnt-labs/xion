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

// ParseGrantsRequestParams validates and parses the Grants request parameters
// This is pure business logic that can be fully unit tested without pagination
func ParseGrantsRequestParams(req *indexerauthz.QueryGrantsRequest, addrCodec address.Codec) (
	granterAddr sdk.AccAddress,
	granteeAddr sdk.AccAddress,
	prefixOpt func(*query.CollectionsPaginateOptions[collections.Triple[sdk.AccAddress, sdk.AccAddress, string]]),
	err error,
) {
	// Parse granter if provided
	if req.Granter != "" {
		granterBytes, err := addrCodec.StringToBytes(req.Granter)
		if err != nil {
			return nil, nil, nil, err
		}
		granterAddr = sdk.AccAddress(granterBytes)
	}

	// Parse grantee if provided
	if req.Grantee != "" {
		granteeBytes, err := addrCodec.StringToBytes(req.Grantee)
		if err != nil {
			return nil, nil, nil, err
		}
		granteeAddr = sdk.AccAddress(granteeBytes)
	}

	// Determine prefix option based on provided parameters
	if req.Granter != "" && req.Grantee != "" {
		// Both granter and grantee specified
		prefixOpt = WithCollectionPaginationTriplePairPrefix[sdk.AccAddress, sdk.AccAddress, string](granterAddr, granteeAddr)
	} else if req.Granter != "" {
		// Only granter specified
		prefixOpt = WithCollectionPaginationTriplePrefix[sdk.AccAddress, sdk.AccAddress, string](granterAddr)
	}
	// Note: when only grantee is specified, we should use the index (not covered here)

	return granterAddr, granteeAddr, prefixOpt, nil
}

// ParseGranterRequestParams validates and parses the GranterGrants request parameters
// This is pure business logic that can be fully unit tested
func ParseGranterRequestParams(req *indexerauthz.QueryGranterGrantsRequest, addrCodec address.Codec) (
	granterAddr sdk.AccAddress,
	err error,
) {
	granterBytes, err := addrCodec.StringToBytes(req.Granter)
	if err != nil {
		return nil, err
	}
	return sdk.AccAddress(granterBytes), nil
}

// ParseGranteeRequestParams validates and parses the GranteeGrants request parameters
// This is pure business logic that can be fully unit tested
func ParseGranteeRequestParams(req *indexerauthz.QueryGranteeGrantsRequest, addrCodec address.Codec) (
	granteeAddr sdk.AccAddress,
	err error,
) {
	granteeBytes, err := addrCodec.StringToBytes(req.Grantee)
	if err != nil {
		return nil, err
	}
	return sdk.AccAddress(granteeBytes), nil
}

// TransformGrantToAuthorization converts a Grant with its primary key to GrantAuthorization
// This is testable business logic for result transformation
func TransformGrantToAuthorization(
	primaryKey collections.Triple[sdk.AccAddress, sdk.AccAddress, string],
	grant authz.Grant,
	cdc codec.Codec,
	addrCodec address.Codec,
) (*authz.GrantAuthorization, error) {
	// Get authorization
	auth, err := grant.GetAuthorization()
	if err != nil {
		return nil, err
	}

	// Pack to Any
	anyValue, err := codectypes.NewAnyWithValue(auth)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}

	// Convert addresses
	granter, err := addrCodec.BytesToString(primaryKey.K1())
	if err != nil {
		return nil, err
	}

	grantee, err := addrCodec.BytesToString(primaryKey.K2())
	if err != nil {
		return nil, err
	}

	return &authz.GrantAuthorization{
		Granter:       granter,
		Grantee:       grantee,
		Authorization: anyValue,
		Expiration:    grant.Expiration,
	}, nil
}

func (aq *authzQuerier) Grants(ctx context.Context, req *indexerauthz.QueryGrantsRequest) (*indexerauthz.QueryGrantsResponse, error) {
	// Convert granter and grantee to sdk.AccAddress if provided
	var (
		granterAddr sdk.AccAddress
		granteeAddr sdk.AccAddress
		err         error
	)
	if req.Granter != "" {
		granterBytes, err := aq.addrCodec.StringToBytes(req.Granter)
		if err != nil {
			return nil, err
		}
		granterAddr = sdk.AccAddress(granterBytes)
	}
	if req.Grantee != "" {
		granteeBytes, err := aq.addrCodec.StringToBytes(req.Grantee)
		if err != nil {
			return nil, err
		}
		granteeAddr = sdk.AccAddress(granteeBytes)
	}

	// Determine prefix for pagination
	var prefixOpt func(o *query.CollectionsPaginateOptions[collections.Triple[sdk.AccAddress, sdk.AccAddress, string]])
	if req.Granter != "" && req.Grantee != "" {
		// Both granter and grantee specified
		prefixOpt = WithCollectionPaginationTriplePairPrefix[sdk.AccAddress, sdk.AccAddress, string](granterAddr, granteeAddr)
	} else if req.Granter != "" {
		// Only granter specified
		prefixOpt = WithCollectionPaginationTriplePrefix[sdk.AccAddress, sdk.AccAddress, string](granterAddr)
	} else if req.Grantee != "" {
		// Only grantee specified - use index instead
		// This should use the Grantee index for better performance
		prefixOpt = nil // Cannot filter by grantee only with triple prefix on main collection
	} else {
		// No prefix, query all
		prefixOpt = nil
	}

	// Paginate over Authorizations
	grants, pageRes, err := query.CollectionPaginate(
		ctx,
		aq.authzHandler.Authorizations,
		req.Pagination,
		func(primaryKey collections.Triple[sdk.AccAddress, sdk.AccAddress, string], grant authz.Grant) (*authz.Grant, error) {
			return &grant, nil
		},
		prefixOpt,
	)
	if err != nil {
		return nil, err
	}
	return &indexerauthz.QueryGrantsResponse{
		Grants:     grants,
		Pagination: pageRes,
	}, nil
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
				return nil, status.Errorf(codes.Internal, "%v", err)
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

func WithCollectionPaginationTriplePairPrefix[K1, K2, K3 any](k1 K1, k2 K2) func(o *query.CollectionsPaginateOptions[collections.Triple[K1, K2, K3]]) {
	return func(o *query.CollectionsPaginateOptions[collections.Triple[K1, K2, K3]]) {
		prefix := collections.TripleSuperPrefix[K1, K2, K3](k1, k2)
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
				return nil, status.Errorf(codes.Internal, "%v", err)
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
