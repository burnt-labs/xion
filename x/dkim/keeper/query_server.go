package keeper

import (
	"context"

	"cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkError "github.com/cosmos/cosmos-sdk/types/errors"

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

// DkimPubKeys implements types.QueryServer.
func (k Querier) DkimPubKey(ctx context.Context, msg *types.QueryDkimPubKeyRequest) (*types.QueryDkimPubKeyResponse, error) {
	dkimPubKey, err := k.OrmDB.DkimPubKeyTable().Get(ctx, msg.Domain, msg.Selector)
	if err != nil {
		return nil, err
	}
	return &types.QueryDkimPubKeyResponse{DkimPubkey: &types.DkimPubKey{
		Domain:   dkimPubKey.Domain,
		PubKey:   dkimPubKey.PubKey,
		Selector: dkimPubKey.Selector,
	}, PoseidonHash: []byte(dkimPubKey.PubKey)}, nil
}

// PoseidonHash implements types.QueryServer.
func (k Querier) PoseidonHash(_ context.Context, msg *types.PoseidonHashRequest) (*types.PoseidonHashResponse, error) {
	hash, err := types.ComputePoseidonHash(msg.PublicKey)
	if err != nil {
		return nil, errors.Wrap(sdkError.ErrInvalidRequest, err.Error())
	}
	return &types.PoseidonHashResponse{
		PoseidonHash: []byte(hash.String()),
	}, nil
}
