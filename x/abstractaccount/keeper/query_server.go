package keeper

import (
	"context"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/abstractaccount/types"
)

type queryServer struct {
	k Keeper
}

func (qs queryServer) AccountAddress(goCtx context.Context, req *types.QueryAccountAddressRequest) (*types.QueryAccountAddressResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	sender, err := sdk.AccAddressFromBech32(req.Sender)
	if err != nil {
		return nil, err
	}
	if err := wasmtypes.ValidateSalt(req.Salt); err != nil {
		return nil, err
	}

	if address, found := qs.k.GetAccountAddress(ctx, sender, req.Salt); found {
		return &types.QueryAccountAddressResponse{Address: address.String(), Registered: true}, nil
	}

	address, err := qs.k.PredictAccountAddress(ctx, sender, req.Salt)
	if err != nil {
		return nil, err
	}

	return &types.QueryAccountAddressResponse{
		Address:    address.String(),
		Registered: qs.k.IsAbstractAccount(ctx, address),
	}, nil
}

func NewQueryServerImpl(k Keeper) types.QueryServer {
	return &queryServer{k}
}

func (qs queryServer) Params(goCtx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	params, err := qs.k.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	return &types.QueryParamsResponse{Params: params}, nil
}
