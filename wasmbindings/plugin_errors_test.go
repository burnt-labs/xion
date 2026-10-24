package wasmbinding_test

import (
	"errors"
	"testing"

	wasmvmtypes "github.com/CosmWasm/wasmvm/v3/types"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/gogoproto/proto"

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	xionapp "github.com/burnt-labs/xion/app"
	wasmbinding "github.com/burnt-labs/xion/wasmbindings"
	zktypes "github.com/burnt-labs/xion/x/zk/types"
)

const balancePath = "/cosmos.bank.v1beta1.Query/Balance"

func balanceRequest(t *testing.T, address string) []byte {
	t.Helper()
	bz, err := proto.Marshal(&banktypes.QueryBalanceRequest{Address: address, Denom: sdk.DefaultBondDenom})
	require.NoError(t, err)
	return bz
}

// A whitelisted path that the router does not serve is refused as unsupported
// rather than reaching a nil handler.
func TestQueriersRejectWhitelistedPathWithoutRoute(t *testing.T) {
	router := baseapp.NewGRPCQueryRouter()
	cdc := xionapp.MakeEncodingConfig(t).Codec

	_, err := wasmbinding.GrpcQuerier(*router)(sdk.Context{}, &wasmvmtypes.GrpcQuery{Path: balancePath})
	var unsupported wasmvmtypes.UnsupportedRequest
	require.ErrorAs(t, err, &unsupported)
	require.Contains(t, unsupported.Kind, "No route to query")

	_, err = wasmbinding.StargateQuerier(*router, cdc)(sdk.Context{}, &wasmvmtypes.StargateQuery{Path: balancePath})
	require.ErrorAs(t, err, &unsupported)
	require.Contains(t, unsupported.Kind, "No route to query")
}

// A module handler failure is returned to the contract as is.
func TestQueriersPropagateRouteErrors(t *testing.T) {
	app := xionapp.Setup(t)
	ctx := app.NewContext(true)
	data := balanceRequest(t, "not-a-bech32-address")

	_, err := wasmbinding.GrpcQuerier(*app.GRPCQueryRouter())(ctx, &wasmvmtypes.GrpcQuery{Path: balancePath, Data: data})
	require.ErrorContains(t, err, "decoding bech32 failed")

	_, err = wasmbinding.StargateQuerier(*app.GRPCQueryRouter(), app.AppCodec())(ctx, &wasmvmtypes.StargateQuery{Path: balancePath, Data: data})
	require.ErrorContains(t, err, "decoding bech32 failed")
}

// A response that does not decode into the whitelisted type is an error: the
// balance response carries a Coin message in field 1 while the substituted
// type expects a bool there.
func TestQueriersRejectResponseOfUnexpectedType(t *testing.T) {
	app := xionapp.Setup(t)
	ctx := app.NewContext(true)
	restore := wasmbinding.OverrideWhitelistedQuery(balancePath, func() proto.Message { return &zktypes.ProofVerifyResponse{} })
	t.Cleanup(restore)
	data := balanceRequest(t, admin)

	_, err := wasmbinding.GrpcQuerier(*app.GRPCQueryRouter())(ctx, &wasmvmtypes.GrpcQuery{Path: balancePath, Data: data})
	require.ErrorContains(t, err, "wrong wireType")

	_, err = wasmbinding.StargateQuerier(*app.GRPCQueryRouter(), app.AppCodec())(ctx, &wasmvmtypes.StargateQuery{Path: balancePath, Data: data})
	require.ErrorIs(t, err, wasmvmtypes.Unknown{})
}

type jsonFailingCodec struct {
	codec.Codec
}

func (jsonFailingCodec) MarshalJSON(proto.Message) ([]byte, error) {
	return nil, errors.New("json encoding refused")
}

func TestConvertProtoToJSONMarshalHidesJSONEncodingFailure(t *testing.T) {
	cdc := xionapp.MakeEncodingConfig(t).Codec
	bz, err := cdc.Marshal(&banktypes.QueryBalanceResponse{Balance: &sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdkmath.OneInt()}})
	require.NoError(t, err)

	_, err = wasmbinding.ConvertProtoToJSONMarshal(&banktypes.QueryBalanceResponse{}, bz, jsonFailingCodec{Codec: cdc})
	require.ErrorIs(t, err, wasmvmtypes.Unknown{})
}
