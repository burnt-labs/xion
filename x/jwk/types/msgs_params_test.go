package types_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/burnt-labs/xion/x/jwk/types"
)

func TestMsgUpdateParamsValidateBasic(t *testing.T) {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	tests := []struct {
		name      string
		authority string
		params    types.Params
		err       string
	}{
		{
			name:      "invalid authority",
			authority: "invalid-address",
			params:    types.DefaultParams(),
			err:       "invalid authority address",
		},
		{
			name:   "empty authority",
			params: types.DefaultParams(),
			err:    "invalid authority address",
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
			name:      "valid",
			authority: authority,
			params:    types.DefaultParams(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &types.MsgUpdateParams{Authority: tt.authority, Params: tt.params}
			if tt.err != "" {
				require.ErrorContains(t, msg.ValidateBasic(), tt.err)
				return
			}
			require.NoError(t, msg.ValidateBasic())
		})
	}
}
