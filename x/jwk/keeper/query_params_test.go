package keeper_test

import (
	"testing"

	"github.com/burnt-labs/xion/x/jwk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	testkeeper "jwk/testutil/keeper"
)

func TestParamsQuery(t *testing.T) {
	keeper, ctx := testkeeper.JwkKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	params := types.DefaultParams()
	keeper.SetParams(ctx, params)

	response, err := keeper.Params(wctx, &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, &types.QueryParamsResponse{Params: params}, response)
}
