package keeper_test

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/burnt-labs/xion/x/jwk/types"
	keepertest "jwk/testutil/keeper"
	"jwk/testutil/nullify"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func TestAudienceQuerySingle(t *testing.T) {
	keeper, ctx := keepertest.JwkKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNAudience(keeper, ctx, 2)
	tests := []struct {
		desc     string
		request  *types.QueryGetAudienceRequest
		response *types.QueryGetAudienceResponse
		err      error
	}{
		{
			desc: "First",
			request: &types.QueryGetAudienceRequest{
				Aud: msgs[0].Aud,
			},
			response: &types.QueryGetAudienceResponse{Audience: msgs[0]},
		},
		{
			desc: "Second",
			request: &types.QueryGetAudienceRequest{
				Aud: msgs[1].Aud,
			},
			response: &types.QueryGetAudienceResponse{Audience: msgs[1]},
		},
		{
			desc: "KeyNotFound",
			request: &types.QueryGetAudienceRequest{
				Aud: strconv.Itoa(100000),
			},
			err: status.Error(codes.NotFound, "not found"),
		},
		{
			desc: "InvalidRequest",
			err:  status.Error(codes.InvalidArgument, "invalid request"),
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			response, err := keeper.Audience(wctx, tc.request)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err)
				require.Equal(t,
					nullify.Fill(tc.response),
					nullify.Fill(response),
				)
			}
		})
	}
}

func TestAudienceQueryPaginated(t *testing.T) {
	keeper, ctx := keepertest.JwkKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	msgs := createNAudience(keeper, ctx, 5)

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllAudienceRequest {
		return &types.QueryAllAudienceRequest{
			Pagination: &query.PageRequest{
				Key:        next,
				Offset:     offset,
				Limit:      limit,
				CountTotal: total,
			},
		}
	}
	t.Run("ByOffset", func(t *testing.T) {
		step := 2
		for i := 0; i < len(msgs); i += step {
			resp, err := keeper.AudienceAll(wctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Audience), step)
			require.Subset(t,
				nullify.Fill(msgs),
				nullify.Fill(resp.Audience),
			)
		}
	})
	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(msgs); i += step {
			resp, err := keeper.AudienceAll(wctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Audience), step)
			require.Subset(t,
				nullify.Fill(msgs),
				nullify.Fill(resp.Audience),
			)
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := keeper.AudienceAll(wctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(msgs), int(resp.Pagination.Total))
		require.ElementsMatch(t,
			nullify.Fill(msgs),
			nullify.Fill(resp.Audience),
		)
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := keeper.AudienceAll(wctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}
