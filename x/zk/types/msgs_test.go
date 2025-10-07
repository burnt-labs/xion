package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"

	"github.com/burnt-labs/xion/x/zk/types"
)

func TestMsgUpdateParams(t *testing.T) {
	addrs := simtestutil.CreateIncrementalAccounts(1)
	addr := addrs[0]
	validAddress := addr.String()

	t.Run("NewMsgUpdateParams", func(t *testing.T) {
		msg := types.NewMsgUpdateParams(addr)
		require.NotNil(t, msg)
		require.Equal(t, validAddress, msg.Authority)
	})

	t.Run("Route", func(t *testing.T) {
		msg := types.MsgUpdateParams{}
		require.Equal(t, types.ModuleName, msg.Route())
	})

	t.Run("Type", func(t *testing.T) {
		msg := types.MsgUpdateParams{}
		require.Equal(t, "update_params", msg.Type())
	})

	t.Run("GetSigners", func(t *testing.T) {
		msg := &types.MsgUpdateParams{Authority: validAddress}
		signers := msg.GetSigners()
		require.Len(t, signers, 1)
		require.Equal(t, addr, signers[0])
	})

	t.Run("ValidateBasic - valid", func(t *testing.T) {
		msg := &types.MsgUpdateParams{
			Authority: validAddress,
			Params:    types.DefaultParams(),
		}
		err := msg.ValidateBasic()
		require.NoError(t, err)
	})

	t.Run("ValidateBasic - invalid address", func(t *testing.T) {
		msg := &types.MsgUpdateParams{
			Authority: "invalid",
			Params:    types.DefaultParams(),
		}
		err := msg.ValidateBasic()
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid authority address")
	})
}

func TestParamsString(t *testing.T) {
	params := types.DefaultParams()
	str := params.String()
	require.NotEmpty(t, str)
	require.Contains(t, str, "vkey")
}

func TestParamsValidate(t *testing.T) {
	params := types.DefaultParams()
	err := params.Validate()
	require.NoError(t, err)
}
