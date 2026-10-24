package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

func TestMsgUpdateParamsValidateBasic(t *testing.T) {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	invalidParams := DefaultParams()
	invalidParams.BypassMinFeeMsgTypes = []string{"invalid"}
	tests := []struct {
		name      string
		authority string
		params    Params
		err       string
	}{
		{
			name:      "invalid authority",
			authority: "invalid-address",
			params:    DefaultParams(),
			err:       "invalid authority address",
		},
		{
			name:   "empty authority",
			params: DefaultParams(),
			err:    "invalid authority address",
		},
		{
			name:      "invalid params",
			authority: authority,
			params:    invalidParams,
			err:       "invalid bypass msg type name",
		},
		{
			name:      "valid",
			authority: authority,
			params:    DefaultParams(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &MsgUpdateParams{Authority: tt.authority, Params: tt.params}
			if tt.err != "" {
				require.ErrorContains(t, msg.ValidateBasic(), tt.err)
				return
			}
			require.NoError(t, msg.ValidateBasic())
		})
	}
}
