package keeper_test

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	"github.com/burnt-labs/xion/x/jwk/keeper"
	"github.com/burnt-labs/xion/x/jwk/types"
	keepertest "jwk/testutil/keeper"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func TestAudienceMsgServerCreate(t *testing.T) {
	k, ctx := keepertest.JwkKeeper(t)
	srv := keeper.NewMsgServerImpl(*k)
	wctx := sdk.WrapSDKContext(ctx)
	admin := "A"
	for i := 0; i < 5; i++ {
		expected := &types.MsgCreateAudience{Admin: admin,
			Aud: strconv.Itoa(i),
		}
		_, err := srv.CreateAudience(wctx, expected)
		require.NoError(t, err)
		rst, found := k.GetAudience(ctx,
			expected.Aud,
		)
		require.True(t, found)
		require.Equal(t, expected.Admin, rst.Admin)
	}
}

func TestAudienceMsgServerUpdate(t *testing.T) {
	admin := "A"

	tests := []struct {
		desc    string
		request *types.MsgUpdateAudience
		err     error
	}{
		{
			desc: "Completed",
			request: &types.MsgUpdateAudience{Admin: admin,
				Aud: strconv.Itoa(0),
			},
		},
		{
			desc: "Unauthorized",
			request: &types.MsgUpdateAudience{Admin: "B",
				Aud: strconv.Itoa(0),
			},
			err: sdkerrors.ErrUnauthorized,
		},
		{
			desc: "KeyNotFound",
			request: &types.MsgUpdateAudience{Admin: admin,
				Aud: strconv.Itoa(100000),
			},
			err: sdkerrors.ErrKeyNotFound,
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			k, ctx := keepertest.JwkKeeper(t)
			srv := keeper.NewMsgServerImpl(*k)
			wctx := sdk.WrapSDKContext(ctx)
			expected := &types.MsgCreateAudience{Admin: admin,
				Aud: strconv.Itoa(0),
			}
			_, err := srv.CreateAudience(wctx, expected)
			require.NoError(t, err)

			_, err = srv.UpdateAudience(wctx, tc.request)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err)
				rst, found := k.GetAudience(ctx,
					expected.Aud,
				)
				require.True(t, found)
				require.Equal(t, expected.Admin, rst.Admin)
			}
		})
	}
}

func TestAudienceMsgServerDelete(t *testing.T) {
	admin := "A"

	tests := []struct {
		desc    string
		request *types.MsgDeleteAudience
		err     error
	}{
		{
			desc: "Completed",
			request: &types.MsgDeleteAudience{Admin: admin,
				Aud: strconv.Itoa(0),
			},
		},
		{
			desc: "Unauthorized",
			request: &types.MsgDeleteAudience{Admin: "B",
				Aud: strconv.Itoa(0),
			},
			err: sdkerrors.ErrUnauthorized,
		},
		{
			desc: "KeyNotFound",
			request: &types.MsgDeleteAudience{Admin: admin,
				Aud: strconv.Itoa(100000),
			},
			err: sdkerrors.ErrKeyNotFound,
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			k, ctx := keepertest.JwkKeeper(t)
			srv := keeper.NewMsgServerImpl(*k)
			wctx := sdk.WrapSDKContext(ctx)

			_, err := srv.CreateAudience(wctx, &types.MsgCreateAudience{Admin: admin,
				Aud: strconv.Itoa(0),
			})
			require.NoError(t, err)
			_, err = srv.DeleteAudience(wctx, tc.request)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err)
				_, found := k.GetAudience(ctx,
					tc.request.Aud,
				)
				require.False(t, found)
			}
		})
	}
}
