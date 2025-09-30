package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"

	"github.com/burnt-labs/xion/x/dkim/types"
)

func TestRegisterLegacyAminoCodec(t *testing.T) {
	cdc := codec.NewLegacyAmino()
	types.RegisterLegacyAminoCodec(cdc)

	addrs := simtestutil.CreateIncrementalAccounts(1)
	addr := addrs[0]

	// Verify that the codec can marshal/unmarshal the messages
	msg := &types.MsgUpdateParams{
		Authority: addr.String(),
		Params:    types.DefaultParams(),
	}

	bz, err := cdc.MarshalJSON(msg)
	require.NoError(t, err)
	require.NotEmpty(t, bz)

	var decoded types.MsgUpdateParams
	err = cdc.UnmarshalJSON(bz, &decoded)
	require.NoError(t, err)
	require.Equal(t, msg.Authority, decoded.Authority)
}

func TestRegisterInterfaces(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)

	addrs := simtestutil.CreateIncrementalAccounts(1)
	addr := addrs[0]

	// Verify that MsgUpdateParams is registered
	msg := &types.MsgUpdateParams{
		Authority: addr.String(),
		Params:    types.DefaultParams(),
	}

	any, err := codectypes.NewAnyWithValue(msg)
	require.NoError(t, err)
	require.NotNil(t, any)
}
