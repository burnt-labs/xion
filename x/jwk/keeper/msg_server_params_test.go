package keeper_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/burnt-labs/xion/x/jwk/keeper"
	"github.com/burnt-labs/xion/x/jwk/types"
)

func TestUpdateParams(t *testing.T) {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	tests := []struct {
		name      string
		authority string
		params    types.Params
		err       string
	}{
		{
			name:      "wrong authority",
			authority: authtypes.NewModuleAddress("other").String(),
			params:    types.NewParams(45_000_000_000, 20_000),
			err:       "expected " + authority + ", got ",
		},
		{
			name:      "invalid deployment gas",
			authority: authority,
			params:    types.NewParams(45_000_000_000, 0),
			err:       "deployment gas must be positive",
		},
		{
			name:      "invalid time offset",
			authority: authority,
			params:    types.NewParams(0, 20_000),
			err:       "time offset must be positive",
		},
		{
			name:      "time offset overflows int64",
			authority: authority,
			params:    types.NewParams(math.MaxInt64+1, 20_000),
			err:       "time offset exceeds max int64",
		},
		{
			name:      "success",
			authority: authority,
			params:    types.NewParams(45_000_000_000, 20_000),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k, ctx := setupKeeper(t)
			before := types.DefaultParams()
			require.NoError(t, k.SetParams(ctx, before))
			server := keeper.NewMsgServerImpl(k)
			response, err := server.UpdateParams(ctx, &types.MsgUpdateParams{
				Authority: tt.authority,
				Params:    tt.params,
			})
			if tt.err != "" {
				require.ErrorContains(t, err, tt.err)
				if tt.authority != authority {
					require.ErrorIs(t, err, types.ErrInvalidAuthority)
				}
				require.Nil(t, response)
				require.Equal(t, before, k.GetParams(ctx))
				return
			}
			require.NoError(t, err)
			require.Equal(t, &types.MsgUpdateParamsResponse{}, response)
			require.Equal(t, tt.params, k.GetParams(ctx))
		})
	}
}
