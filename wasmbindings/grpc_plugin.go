package wasmbinding

import (
	"fmt"

	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"

	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/cosmos/gogoproto/proto"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func GrpcQuerier(queryRouter baseapp.GRPCQueryRouter) func(ctx sdk.Context, request *wasmvmtypes.GrpcQuery) (proto.Message, error) {
	return func(ctx sdk.Context, request *wasmvmtypes.GrpcQuery) (proto.Message, error) {
		protoResponse, err := GetWhitelistedQuery(request.Path)
		if err != nil {
			return nil, err
		}

		route := queryRouter.Route(request.Path)
		if route == nil {
			return nil, wasmvmtypes.UnsupportedRequest{Kind: fmt.Sprintf("No route to query '%s'", request.Path)}
		}

		res, err := route(ctx, &abci.RequestQuery{
			Data: request.Data,
			Path: request.Path,
		})
		if err != nil {
			return nil, err
		}

		if res.Value == nil {
			return nil, fmt.Errorf("res returned from abci query route is nil")
		}
		err = proto.Unmarshal(res.Value, protoResponse)
		if err != nil {
			return nil, err
		}

		return protoResponse, nil
	}
}
