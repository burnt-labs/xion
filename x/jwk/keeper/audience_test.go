package keeper_test

import (
	"strconv"
	"testing"

	"github.com/burnt-labs/xion/x/jwk/keeper"
	"github.com/burnt-labs/xion/x/jwk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	keepertest "jwk/testutil/keeper"
	"jwk/testutil/nullify"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func createNAudience(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.Audience {
	items := make([]types.Audience, n)
	for i := range items {
		items[i].Aud = strconv.Itoa(i)

		keeper.SetAudience(ctx, items[i])
	}
	return items
}

func TestAudienceGet(t *testing.T) {
	keeper, ctx := keepertest.JwkKeeper(t)
	items := createNAudience(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetAudience(ctx,
			item.Aud,
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&rst),
		)
	}
}
func TestAudienceRemove(t *testing.T) {
	keeper, ctx := keepertest.JwkKeeper(t)
	items := createNAudience(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveAudience(ctx,
			item.Aud,
		)
		_, found := keeper.GetAudience(ctx,
			item.Aud,
		)
		require.False(t, found)
	}
}

func TestAudienceGetAll(t *testing.T) {
	keeper, ctx := keepertest.JwkKeeper(t)
	items := createNAudience(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllAudience(ctx)),
	)
}
