package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	xionapp "github.com/burnt-labs/xion/app"
	"github.com/burnt-labs/xion/x/abstractaccount/keeper"
	"github.com/burnt-labs/xion/x/abstractaccount/types"
)

func TestQueryParams(t *testing.T) {
	app := xionapp.Setup(t)
	ctx := app.NewContext(false)

	queryServer := keeper.NewQueryServerImpl(app.AbstractAccountKeeper)

	res, err := queryServer.Params(sdk.WrapSDKContext(ctx), &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, mockParams, res.Params)
}
