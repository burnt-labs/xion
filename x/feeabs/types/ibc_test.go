package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	abci "github.com/cometbft/cometbft/abci/types"
)

func TestIBCConstants(t *testing.T) {
	require.Equal(t, "feeabs", IBCPortID)
	require.Equal(t, []byte{0x01}, IBCPortKey)
	require.Equal(t, []byte{0x02}, FeePoolAddressKey)
}

func TestModuleCdc(t *testing.T) {
	require.NotNil(t, ModuleCdc)
}

func TestNewQueryArithmeticTwapToNowRequest(t *testing.T) {
	poolID := uint64(123)
	baseDenom := "uatom"
	quoteDenom := "uosmo"
	startTime := time.Now()

	req := NewQueryArithmeticTwapToNowRequest(poolID, baseDenom, quoteDenom, startTime)

	require.Equal(t, poolID, req.PoolId)
	require.Equal(t, baseDenom, req.BaseAsset)
	require.Equal(t, quoteDenom, req.QuoteAsset)
	require.Equal(t, startTime, req.StartTime)
}

func TestQueryArithmeticTwapToNowRequest_GetBytes(t *testing.T) {
	req := NewQueryArithmeticTwapToNowRequest(1, "uatom", "uosmo", time.Now())

	bytes := req.GetBytes()
	require.NotNil(t, bytes)
	require.Greater(t, len(bytes), 0)

	// Should be able to unmarshal back
	var decoded QueryArithmeticTwapToNowRequest
	err := ModuleCdc.Unmarshal(bytes, &decoded)
	require.NoError(t, err)
	require.Equal(t, req.PoolId, decoded.PoolId)
	require.Equal(t, req.BaseAsset, decoded.BaseAsset)
	require.Equal(t, req.QuoteAsset, decoded.QuoteAsset)
}

func TestSerializeDeserializeCosmosQuery(t *testing.T) {
	// Create test requests
	reqs := []abci.RequestQuery{
		{
			Path:   "/test/path1",
			Data:   []byte("test-data-1"),
			Height: 100,
			Prove:  false,
		},
		{
			Path:   "/test/path2",
			Data:   []byte("test-data-2"),
			Height: 200,
			Prove:  true,
		},
	}

	// Serialize
	bz, err := SerializeCosmosQuery(reqs)
	require.NoError(t, err)
	require.Greater(t, len(bz), 0)

	// Deserialize
	decodedReqs, err := DeserializeCosmosQuery(bz)
	require.NoError(t, err)
	require.Len(t, decodedReqs, 2)

	// Verify contents
	require.Equal(t, reqs[0].Path, decodedReqs[0].Path)
	require.Equal(t, reqs[0].Data, decodedReqs[0].Data)
	require.Equal(t, reqs[0].Height, decodedReqs[0].Height)
	require.Equal(t, reqs[0].Prove, decodedReqs[0].Prove)

	require.Equal(t, reqs[1].Path, decodedReqs[1].Path)
	require.Equal(t, reqs[1].Data, decodedReqs[1].Data)
	require.Equal(t, reqs[1].Height, decodedReqs[1].Height)
	require.Equal(t, reqs[1].Prove, decodedReqs[1].Prove)
}

func TestSerializeDeserializeCosmosResponse(t *testing.T) {
	// Create test responses
	resps := []abci.ResponseQuery{
		{
			Code:   0,
			Key:    []byte("key1"),
			Value:  []byte("value1"),
			Height: 100,
		},
		{
			Code:   1,
			Key:    []byte("key2"),
			Value:  []byte("value2"),
			Height: 200,
		},
	}

	// Serialize
	bz, err := SerializeCosmosResponse(resps)
	require.NoError(t, err)
	require.Greater(t, len(bz), 0)

	// Deserialize
	decodedResps, err := DeserializeCosmosResponse(bz)
	require.NoError(t, err)
	require.Len(t, decodedResps, 2)

	// Verify contents
	require.Equal(t, resps[0].Code, decodedResps[0].Code)
	require.Equal(t, resps[0].Key, decodedResps[0].Key)
	require.Equal(t, resps[0].Value, decodedResps[0].Value)
	require.Equal(t, resps[0].Height, decodedResps[0].Height)

	require.Equal(t, resps[1].Code, decodedResps[1].Code)
	require.Equal(t, resps[1].Key, decodedResps[1].Key)
	require.Equal(t, resps[1].Value, decodedResps[1].Value)
	require.Equal(t, resps[1].Height, decodedResps[1].Height)
}

func TestNewInterchainQueryRequest(t *testing.T) {
	path := "/test/query/path"
	data := []byte("test-query-data")

	req := NewInterchainQueryRequest(path, data)

	require.Equal(t, path, req.Path)
	require.Equal(t, data, req.Data)
}

func TestNewInterchainQueryPacketData(t *testing.T) {
	data := []byte("test-packet-data")
	memo := "test-memo"

	packet := NewInterchainQueryPacketData(data, memo)

	require.Equal(t, data, packet.Data)
	require.Equal(t, memo, packet.Memo)
}

func TestInterchainQueryPacketData_GetBytes(t *testing.T) {
	packet := NewInterchainQueryPacketData([]byte("test-data"), "test-memo")

	bytes := packet.GetBytes()
	require.NotNil(t, bytes)
	require.Greater(t, len(bytes), 0)

	// Should be valid JSON
	var decoded InterchainQueryPacketData
	err := ModuleCdc.UnmarshalJSON(bytes, &decoded)
	require.NoError(t, err)
	require.Equal(t, packet.Data, decoded.Data)
	require.Equal(t, packet.Memo, decoded.Memo)
}

func TestSerializeCosmosQueryError(t *testing.T) {
	// Test with nil requests (should not error, but create empty query)
	bz, err := SerializeCosmosQuery(nil)
	require.NoError(t, err)
	require.NotNil(t, bz)

	// Deserialize back
	reqs, err := DeserializeCosmosQuery(bz)
	require.NoError(t, err)
	require.Len(t, reqs, 0)
}

func TestDeserializeCosmosQueryError(t *testing.T) {
	// Test with invalid data
	_, err := DeserializeCosmosQuery([]byte("invalid-data"))
	require.Error(t, err)
}

func TestSerializeCosmosResponseError(t *testing.T) {
	// Test with nil responses (should not error, but create empty response)
	bz, err := SerializeCosmosResponse(nil)
	require.NoError(t, err)
	require.NotNil(t, bz)

	// Deserialize back
	resps, err := DeserializeCosmosResponse(bz)
	require.NoError(t, err)
	require.Len(t, resps, 0)
}

func TestDeserializeCosmosResponseError(t *testing.T) {
	// Test with invalid data
	_, err := DeserializeCosmosResponse([]byte("invalid-data"))
	require.Error(t, err)
}
