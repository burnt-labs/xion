package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestUpdateParamsCodec(t *testing.T) {
	params := DefaultParams()
	params.MaxTotalBypassMinFeeMsgGasUsage = 123456
	var err error
	params.MinimumGasPrices, err = sdk.ParseDecCoins("0.001uxion")
	require.NoError(t, err)
	msg := &MsgUpdateParams{Authority: "authority", Params: params}

	registry := codectypes.NewInterfaceRegistry()
	RegisterInterfaces(registry)
	resolved, err := registry.Resolve("/xion.globalfee.v1.MsgUpdateParams")
	require.NoError(t, err)
	require.IsType(t, &MsgUpdateParams{}, resolved)
	require.Implements(t, (*sdk.Msg)(nil), resolved)

	packed, err := codectypes.NewAnyWithValue(msg)
	require.NoError(t, err)
	require.Equal(t, "/xion.globalfee.v1.MsgUpdateParams", packed.TypeUrl)
	cdc := codec.NewProtoCodec(registry)
	bz, err := cdc.Marshal(packed)
	require.NoError(t, err)
	var decodedAny codectypes.Any
	require.NoError(t, cdc.Unmarshal(bz, &decodedAny))
	var decoded sdk.Msg
	require.NoError(t, cdc.UnpackAny(&decodedAny, &decoded))
	require.Equal(t, msg, decoded)

	amino := codec.NewLegacyAmino()
	RegisterLegacyAminoCodec(amino)
	bz, err = amino.MarshalJSON(msg)
	require.NoError(t, err)
	require.Contains(t, string(bz), `"type":"globalfee/MsgUpdateParams"`)
	var aminoDecoded MsgUpdateParams
	require.NoError(t, amino.UnmarshalJSON(bz, &aminoDecoded))
	require.Equal(t, msg, &aminoDecoded)
}
