package indexer

import (
	"context"
	"log/slog"

	"github.com/gogo/status"
	"google.golang.org/grpc/codes"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	authz "github.com/cosmos/cosmos-sdk/x/authz"

	indexerauthz "github.com/burnt-labs/xion/indexer/authz"
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
	switch {
	case req.Granter != "" && req.Grantee != "":
		// Both granter and grantee specified
		prefixOpt = WithCollectionPaginationTriplePairPrefix[sdk.AccAddress, sdk.AccAddress, string](granterAddr, granteeAddr)
	case req.Granter != "":
		// Only granter specified
		prefixOpt = WithCollectionPaginationTriplePrefix[sdk.AccAddress, sdk.AccAddress, string](granterAddr)
	default:
		// Only grantee specified or no filters - cannot use prefix optimization
		// For grantee-only queries, the Grantee index should be used instead
		prefixOpt = nil
	}

	// Paginate over Authorizations
	var opts []func(o *query.CollectionsPaginateOptions[collections.Triple[sdk.AccAddress, sdk.AccAddress, string]])
	if prefixOpt != nil {
		opts = append(opts, prefixOpt)
	}

	grants, pageRes, err := query.CollectionPaginate(
		ctx,
		aq.authzHandler.Authorizations,
		req.Pagination,
		func(primaryKey collections.Triple[sdk.AccAddress, sdk.AccAddress, string], grant authz.Grant) (*authz.Grant, error) {
			return &grant, nil
		},
		opts...,
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
	slog.Debug("authz_querier", "granter", req.Granter)

	granter, err := aq.addrCodec.StringToBytes(req.Granter)
	if err != nil {
		return nil, err
	}
	granterAddr := sdk.AccAddress(granter)
	slog.Debug("authz_querier", "step", "1")

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
	slog.Debug("authz_querier", "grantee", req.Grantee)
	grantee, err := aq.addrCodec.StringToBytes(req.Grantee)
	if err != nil {
		return nil, err
	}
	granteeAddr := sdk.AccAddress(grantee)

	// Use MultiIterateRaw for raw byte-range pagination when pagination key is provided
	// This demonstrates the use case for MultiIterateRaw: custom pagination with byte boundaries
	useRawIteration := req.Pagination != nil && len(req.Pagination.Key) > 0

	if useRawIteration {
		slog.Debug("authz_querier", "using_raw_iteration", true, "pagination_key_len", len(req.Pagination.Key))
		// Use MultiIterateRaw for raw byte-range iteration
		// This is the primary use case: pagination from a specific byte position
		return aq.granteeGrantsWithRawIteration(ctx, req, granteeAddr)
	}

	slog.Debug("authz_querier", "using_standard_iteration", true)
	// Use the grantee index to efficiently query grants for this grantee
	// The index stores: Pair[grantee, Triple[granter, grantee, msgType]]
	// We iterate over the index with a grantee prefix, then fetch from main collection
	ranger := collections.NewPrefixedPairRange[sdk.AccAddress, collections.Triple[sdk.AccAddress, sdk.AccAddress, string]](granteeAddr)

	iter, err := aq.authzHandler.Authorizations.Indexes.Grantee.Iterate(ctx, ranger)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	// Manual pagination handling
	var (
		grants     []*authz.GrantAuthorization
		count      uint64
		nextKey    []byte
		limit      uint64 = query.DefaultLimit
		offset     uint64
		countTotal bool
		total      uint64
	)

	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		offset = req.Pagination.Offset
		countTotal = req.Pagination.CountTotal

		// If pagination key is provided, skip to that position
		if len(req.Pagination.Key) > 0 {
			// Fast-forward the iterator to the key position
			// We need to skip items until we reach the pagination key
			keyCodec := aq.authzHandler.Authorizations.Indexes.Grantee.KeyCodec()
			for ; iter.Valid(); iter.Next() {
				fullKey, err := iter.FullKey()
				if err != nil {
					return nil, err
				}
				// Encode the full key to compare with pagination key
				buf := make([]byte, 128) // Pre-allocate with length, not just capacity
				n, err := keyCodec.EncodeNonTerminal(buf, fullKey)
				if err != nil {
					return nil, err
				}
				keyBytes := buf[:n]
				if string(keyBytes) >= string(req.Pagination.Key) {
					break
				}
			}
		} else if offset > 0 {
			// Skip offset items
			for i := uint64(0); i < offset && iter.Valid(); i++ {
				iter.Next()
			}
		}
	}

	// Collect results up to limit
	for ; iter.Valid() && count < limit; iter.Next() {
		primaryKey, err := iter.PrimaryKey()
		if err != nil {
			return nil, err
		}

		// Fetch the grant from the main collection using the primary key
		grant, err := aq.authzHandler.Authorizations.Get(ctx, primaryKey)
		if err != nil {
			return nil, err
		}

		auth, err := grant.GetAuthorization()
		if err != nil {
			return nil, err
		}

		anyValue, err := codectypes.NewAnyWithValue(auth)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "%v", err)
		}

		granter, err := aq.addrCodec.BytesToString(primaryKey.K1())
		if err != nil {
			return nil, err
		}

		grants = append(grants, &authz.GrantAuthorization{
			Granter:       granter,
			Grantee:       req.Grantee,
			Authorization: anyValue,
			Expiration:    grant.Expiration,
		})

		count++
		total++
	}

	// If there are more results, set the next key
	if iter.Valid() {
		fullKey, err := iter.FullKey()
		if err != nil {
			return nil, err
		}
		keyCodec := aq.authzHandler.Authorizations.Indexes.Grantee.KeyCodec()
		buf := make([]byte, 128) // Pre-allocate with length, not just capacity
		n, err := keyCodec.EncodeNonTerminal(buf, fullKey)
		if err != nil {
			return nil, err
		}
		nextKey = buf[:n]
	}

	// Count remaining if countTotal is requested
	if countTotal {
		for ; iter.Valid(); iter.Next() {
			total++
		}
	}

	pageRes := &query.PageResponse{
		NextKey: nextKey,
	}
	if countTotal {
		pageRes.Total = total
	}

	slog.Debug("authz_querier", "grants", len(grants))
	return &indexerauthz.QueryGranteeGrantsResponse{
		Grants:     grants,
		Pagination: pageRes,
	}, nil
}

// granteeGrantsWithRawIteration uses MultiIterateRaw for raw byte-range pagination
// This demonstrates the primary use case for MultiIterateRaw: efficient pagination
// from a specific byte position using raw iteration.
func (aq *authzQuerier) granteeGrantsWithRawIteration(
	ctx context.Context,
	req *indexerauthz.QueryGranteeGrantsRequest,
	granteeAddr sdk.AccAddress,
) (*indexerauthz.QueryGranteeGrantsResponse, error) {
	// Use MultiIterateRaw to start iteration from the pagination key's byte position
	iter, err := MultiIterateRaw(
		ctx,
		aq.authzHandler.Authorizations.Indexes.Grantee,
		req.Pagination.Key, // Start from this byte position
		nil,                // No end boundary
		collections.OrderAscending,
	)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	var (
		grants     []*authz.GrantAuthorization
		count      uint64
		nextKey    []byte
		limit      uint64 = query.DefaultLimit
		countTotal bool
		total      uint64
	)

	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		countTotal = req.Pagination.CountTotal
	}

	// Collect results up to limit
	// Note: With MultiIterateRaw, we get a Pair[grantee, Triple[granter, grantee, msgType]]
	for ; iter.Valid() && count < limit; iter.Next() {
		key, err := iter.Key()
		if err != nil {
			return nil, err
		}

		// Extract the primary key from the index key
		// key.K1() is the grantee (reference key)
		// key.K2() is the Triple[granter, grantee, msgType] (primary key)
		primaryKey := key.K2()

		// Filter by grantee if not matching (since raw iteration doesn't apply prefix)
		if !key.K1().Equals(granteeAddr) {
			continue
		}

		// Fetch the grant from the main collection
		grant, err := aq.authzHandler.Authorizations.Get(ctx, primaryKey)
		if err != nil {
			return nil, err
		}

		auth, err := grant.GetAuthorization()
		if err != nil {
			return nil, err
		}

		anyValue, err := codectypes.NewAnyWithValue(auth)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "%v", err)
		}

		granter, err := aq.addrCodec.BytesToString(primaryKey.K1())
		if err != nil {
			return nil, err
		}

		grants = append(grants, &authz.GrantAuthorization{
			Granter:       granter,
			Grantee:       req.Grantee,
			Authorization: anyValue,
			Expiration:    grant.Expiration,
		})

		count++
		total++
	}

	// If there are more results, set the next key
	if iter.Valid() {
		key, err := iter.Key()
		if err != nil {
			return nil, err
		}
		keyCodec := aq.authzHandler.Authorizations.Indexes.Grantee.KeyCodec()
		buf := make([]byte, 128)
		n, err := keyCodec.EncodeNonTerminal(buf, key)
		if err != nil {
			return nil, err
		}
		nextKey = buf[:n]
	}

	// Count remaining if countTotal is requested
	if countTotal {
		for ; iter.Valid(); iter.Next() {
			key, err := iter.Key()
			if err != nil {
				return nil, err
			}
			// Only count entries for this grantee
			if key.K1().Equals(granteeAddr) {
				total++
			}
		}
	}

	pageRes := &query.PageResponse{
		NextKey: nextKey,
	}
	if countTotal {
		pageRes.Total = total
	}

	slog.Debug("authz_querier", "grants_via_raw_iteration", len(grants))
	return &indexerauthz.QueryGranteeGrantsResponse{
		Grants:     grants,
		Pagination: pageRes,
	}, nil
}
