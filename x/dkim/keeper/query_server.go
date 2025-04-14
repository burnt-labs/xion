package keeper

import (
	"context"

	queryv1beta1 "cosmossdk.io/api/cosmos/base/query/v1beta1"
	"cosmossdk.io/orm/model/ormlist"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	dkimv1 "github.com/burnt-labs/xion/api/xion/dkim/v1"
	"github.com/burnt-labs/xion/x/dkim/types"
)

var _ types.QueryServer = Querier{}

type Querier struct {
	Keeper
}

func NewQuerier(keeper Keeper) Querier {
	return Querier{Keeper: keeper}
}

func (k Querier) Params(c context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	p, err := k.Keeper.Params.Get(ctx)
	if err != nil {
		return nil, err
	}

	return &types.QueryParamsResponse{Params: &p}, nil
}

// DkimPubKey implements types.QueryServer.
func (k Querier) DkimPubKey(ctx context.Context, msg *types.QueryDkimPubKeyRequest) (*types.QueryDkimPubKeyResponse, error) {
	dkimPubKey, err := k.OrmDB.DkimPubKeyTable().Get(ctx, msg.Domain, msg.Selector)
	if err != nil {
		return nil, err
	}
	return &types.QueryDkimPubKeyResponse{DkimPubKey: &types.DkimPubKey{
		Domain:   dkimPubKey.Domain,
		PubKey:   dkimPubKey.PubKey,
		Selector: dkimPubKey.Selector,
	}}, nil
}

// DkimPubKeys implements types.QueryServer
func (k Querier) DkimPubKeys(ctx context.Context, msg *types.QueryDkimPubKeysRequest) (*types.QueryDkimPubKeysResponse, error) {
	switch {
	case msg.Domain != "" && msg.Selector != "":
		// direct request for a pubKey
		dkimPubKey, err := k.OrmDB.DkimPubKeyTable().Get(ctx, msg.Domain, msg.Selector)
		if err != nil {
			return nil, err
		}
		return &types.QueryDkimPubKeysResponse{
			DkimPubKeys: []*types.DkimPubKey{{
				Domain:       dkimPubKey.Domain,
				PubKey:       dkimPubKey.PubKey,
				Selector:     dkimPubKey.Selector,
				PoseidonHash: dkimPubKey.PoseidonHash,
				Version:      types.Version(dkimPubKey.Version),
				KeyType:      types.KeyType(dkimPubKey.KeyType),
			}},
			Pagination: nil,
		}, nil
	case msg.Domain != "" && msg.PoseidonHash != nil:
		// secondary index for domain+hash exists, and should be used
		pageRequest := convertPageRequest(msg.Pagination)
		results, err := k.OrmDB.DkimPubKeyTable().List(ctx,
			dkimv1.DkimPubKeyDomainPoseidonHashIndexKey{}.WithDomainPoseidonHash(msg.Domain, msg.PoseidonHash),
			ormlist.Paginate(pageRequest))
		if err != nil {
			return nil, err
		}

		pubKeys, err := consumeIteratorResults(results)
		if err != nil {
			return nil, err
		}

		return &types.QueryDkimPubKeysResponse{
			DkimPubKeys: pubKeys,
			Pagination:  convertPageResponse(results.PageResponse()),
		}, nil
	case msg.Domain != "":
		// all pubKeys for a domain
		pageRequest := convertPageRequest(msg.Pagination)
		results, err := k.OrmDB.DkimPubKeyTable().List(ctx,
			dkimv1.DkimPubKeyPrimaryKey{}.WithDomain(msg.Domain),
			ormlist.Paginate(pageRequest))
		if err != nil {
			return nil, err
		}

		pubKeys, err := consumeIteratorResults(results)
		if err != nil {
			return nil, err
		}

		return &types.QueryDkimPubKeysResponse{
			DkimPubKeys: pubKeys,
			Pagination:  convertPageResponse(results.PageResponse()),
		}, nil
	default:
		// all pubKeys
		pageRequest := convertPageRequest(msg.Pagination)
		results, err := k.OrmDB.DkimPubKeyTable().List(ctx,
			dkimv1.DkimPubKeyPrimaryKey{},
			ormlist.Paginate(pageRequest))
		if err != nil {
			return nil, err
		}

		pubKeys, err := consumeIteratorResults(results)
		if err != nil {
			return nil, err
		}

		return &types.QueryDkimPubKeysResponse{
			DkimPubKeys: pubKeys,
			Pagination:  convertPageResponse(results.PageResponse()),
		}, nil
	}
}

func convertPageRequest(request *query.PageRequest) *queryv1beta1.PageRequest {
	if request != nil {
		pageRequest := queryv1beta1.PageRequest{}
		pageRequest.CountTotal = request.CountTotal
		pageRequest.Key = request.Key
		pageRequest.Offset = request.Offset
		pageRequest.Limit = request.Limit
		pageRequest.Reverse = request.Reverse
		return &pageRequest
	}

	return nil
}

func convertPageResponse(response *queryv1beta1.PageResponse) *query.PageResponse {
	if response != nil {
		pageResponse := query.PageResponse{}
		pageResponse.NextKey = response.NextKey
		pageResponse.Total = response.Total
		return &pageResponse
	}

	return nil
}

func consumeIteratorResults(iterator dkimv1.DkimPubKeyIterator) (output []*types.DkimPubKey, err error) {
	defer iterator.Close()

	for iterator.Next() {
		dkimPubKey, err := iterator.Value()
		if err != nil {
			return nil, err
		}
		output = append(output, &types.DkimPubKey{
			Domain:   dkimPubKey.Domain,
			PubKey:   dkimPubKey.PubKey,
			Selector: dkimPubKey.Selector,
		})
	}

	return output, nil
}
