package types

import (
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	abci "github.com/cometbft/cometbft/abci/types"
)

const (
	// IBCPortID is the default port id that profiles module binds to.
	IBCPortID = "feeabs"
)

var ModuleCdc = codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

// IBCPortKey defines the key to store the port ID in store.
var (
	IBCPortKey        = []byte{0x01}
	FeePoolAddressKey = []byte{0x02}
)

// NewQueryArithmeticTwapToNowRequest create new packet for ibc.
func NewQueryArithmeticTwapToNowRequest(
	poolID uint64,
	baseDenom string,
	quoteDenom string,
	startTime time.Time,
) QueryArithmeticTwapToNowRequest {
	return QueryArithmeticTwapToNowRequest{
		PoolId:     poolID,
		BaseAsset:  baseDenom,
		QuoteAsset: quoteDenom,
		StartTime:  startTime,
	}
}

func (p QueryArithmeticTwapToNowRequest) GetBytes() []byte {
	return ModuleCdc.MustMarshal(&p)
}

func SerializeCosmosQuery(reqs []abci.RequestQuery) (bz []byte, err error) {
	q := &CosmosQuery{
		Requests: reqs,
	}
	return ModuleCdc.Marshal(q)
}

func DeserializeCosmosQuery(bz []byte) (reqs []abci.RequestQuery, err error) {
	var q CosmosQuery
	err = ModuleCdc.Unmarshal(bz, &q)
	return q.Requests, err
}

func SerializeCosmosResponse(resps []abci.ResponseQuery) (bz []byte, err error) {
	r := &CosmosResponse{
		Responses: resps,
	}
	return ModuleCdc.Marshal(r)
}

func DeserializeCosmosResponse(bz []byte) (resps []abci.ResponseQuery, err error) {
	var r CosmosResponse
	err = ModuleCdc.Unmarshal(bz, &r)
	return r.Responses, err
}

func NewInterchainQueryRequest(path string, data []byte) InterchainQueryRequest {
	return InterchainQueryRequest{
		Data: data,
		Path: path,
	}
}

func NewInterchainQueryPacketData(data []byte, memo string) InterchainQueryPacketData {
	return InterchainQueryPacketData{
		Data: data,
		Memo: memo,
	}
}

// GetBytes returns the JSON marshalled interchain query packet data.
func (p InterchainQueryPacketData) GetBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(&p))
}
