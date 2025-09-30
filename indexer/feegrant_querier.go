package indexer

import (
	"context"
	"errors"
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

// ParseAllowanceRequestParams validates and parses the Allowance request parameters
// This is pure business logic that can be fully unit tested
func ParseAllowanceRequestParams(req *indexerfeegrant.QueryAllowanceRequest, addrCodec address.Codec) (
	granterAddr sdk.AccAddress,
	granteeAddr sdk.AccAddress,
	err error,
) {
	// Parse granter
	granterBytes, err := addrCodec.StringToBytes(req.Granter)
	if err != nil {
		return nil, nil, err
	}
	granterAddr = sdk.AccAddress(granterBytes)

	// Parse grantee
	granteeBytes, err := addrCodec.StringToBytes(req.Grantee)
	if err != nil {
		return nil, nil, err
	}
	granteeAddr = sdk.AccAddress(granteeBytes)

	return granterAddr, granteeAddr, nil
}

// ParseAllowancesRequestParams validates and parses the Allowances request parameters
// This is pure business logic that can be fully unit tested
func ParseAllowancesRequestParams(req *indexerfeegrant.QueryAllowancesRequest, addrCodec address.Codec) (
	granteeAddr sdk.AccAddress,
	err error,
) {
	// Parse grantee
	granteeBytes, err := addrCodec.StringToBytes(req.Grantee)
	if err != nil {
		return nil, err
	}
	granteeAddr = sdk.AccAddress(granteeBytes)

	return granteeAddr, nil
}

// ParseAllowancesByGranterRequestParams validates and parses the AllowancesByGranter request parameters
// This is pure business logic that can be fully unit tested
func ParseAllowancesByGranterRequestParams(req *indexerfeegrant.QueryAllowancesByGranterRequest, addrCodec address.Codec) (
	granterAddr sdk.AccAddress,
	err error,
) {
	// Parse granter
	granterBytes, err := addrCodec.StringToBytes(req.Granter)
	if err != nil {
		return nil, err
	}
	granterAddr = sdk.AccAddress(granterBytes)

	return granterAddr, nil
}

func (fq *feegrantQuerier) Allowance(ctx context.Context, req *indexerfeegrant.QueryAllowanceRequest) (*indexerfeegrant.QueryAllowanceResponse, error) {
	granter, err := fq.addrCodec.StringToBytes(req.Granter)
	if err != nil {
		return nil, err
	}
	grantee, err := fq.addrCodec.StringToBytes(req.Grantee)
	if err != nil {
		return nil, err
	}
	granterAddr := sdk.AccAddress(granter)
	granteeAddr := sdk.AccAddress(grantee)

	grant, err := fq.feegrantHandler.FeeAllowances.Get(ctx, collections.Join(granterAddr, granteeAddr))
	if err != nil {
		// If not found, return nil grant but no error
		if errors.Is(err, collections.ErrNotFound) {
			return &indexerfeegrant.QueryAllowanceResponse{
				Allowance: nil,
			}, nil
		}
		return nil, err
	}
	return &indexerfeegrant.QueryAllowanceResponse{
		Allowance: &grant,
	}, nil
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
