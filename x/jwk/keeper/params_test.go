package keeper_test

import (
	"testing"

	"github.com/burnt-labs/xion/x/jwk/types"
	"github.com/stretchr/testify/require"
	testkeeper "jwk/testutil/keeper"
)

func TestGetParams(t *testing.T) {
	k, ctx := testkeeper.JwkKeeper(t)
	params := types.DefaultParams()

	k.SetParams(ctx, params)

	require.EqualValues(t, params, k.GetParams(ctx))
}
