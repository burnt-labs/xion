package wasmbinding_test

import (
	"testing"

	wasmvmtypes "github.com/CosmWasm/wasmvm/v3/types"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/gogoproto/proto"

	"github.com/cosmos/cosmos-sdk/codec"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	xionapp "github.com/burnt-labs/xion/app"
	wasmbinding "github.com/burnt-labs/xion/wasmbindings"
)

func TestGrpcQuerier_ErrorCases(t *testing.T) {
	app := xionapp.Setup(t)
	ctx := app.NewContext(true)

	tests := []struct {
		name          string
		path          string
		data          []byte
		expectError   bool
		errorContains string
	}{
		{
			name:          "unsupported query path",
			path:          "/unsupported.query.Path",
			data:          []byte{},
			expectError:   true,
			errorContains: "path is not allowed from the contract",
		},
		{
			name: "invalid request data for valid path",
			path: "/cosmos.bank.v1beta1.Query/Balance",
			data: []byte("invalid protobuf data"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			grpcQuerier := wasmbinding.GrpcQuerier(*app.GRPCQueryRouter())
			grpcRequest := &wasmvmtypes.GrpcQuery{
				Path: tt.path,
				Data: tt.data,
			}

			response, err := grpcQuerier(ctx, grpcRequest)

			if tt.expectError {
				require.Error(t, err)
				require.Nil(t, response)
				if tt.errorContains != "" {
					require.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, response)
			}
		})
	}
}

func TestStargateQuerier_ErrorCases(t *testing.T) {
	app := xionapp.Setup(t)
	ctx := app.NewContext(true)

	tests := []struct {
		name          string
		path          string
		data          []byte
		expectError   bool
		errorContains string
	}{
		{
			name:          "unsupported query path",
			path:          "/unsupported.query.Path",
			data:          []byte{},
			expectError:   true,
			errorContains: "path is not allowed from the contract",
		},
		{
			name: "invalid request data for valid path",
			path: "/cosmos.bank.v1beta1.Query/Balance",
			data: []byte("invalid protobuf data"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stargateQuerier := wasmbinding.StargateQuerier(*app.GRPCQueryRouter(), app.AppCodec())
			stargateRequest := &wasmvmtypes.StargateQuery{
				Path: tt.path,
				Data: tt.data,
			}

			response, err := stargateQuerier(ctx, stargateRequest)

			if tt.expectError {
				require.Error(t, err)
				require.Nil(t, response)
				if tt.errorContains != "" {
					require.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, response)
			}
		})
	}
}

func TestConvertProtoToJSONMarshal_ErrorCases(t *testing.T) {
	app := xionapp.Setup(t)
	cdc := app.AppCodec()

	tests := []struct {
		name             string
		protoMessage     proto.Message
		data             []byte
		expectError      bool
		expectUnknownErr bool
	}{
		{
			name:             "invalid protobuf data - unmarshal error",
			protoMessage:     &banktypes.QueryBalanceResponse{},
			data:             []byte("invalid protobuf data"),
			expectError:      true,
			expectUnknownErr: true,
		},
		{
			name:         "valid data",
			protoMessage: &banktypes.QueryBalanceResponse{},
			data: func() []byte {
				resp := &banktypes.QueryBalanceResponse{}
				data, _ := cdc.Marshal(resp)
				return data
			}(),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := wasmbinding.ConvertProtoToJSONMarshal(tt.protoMessage, tt.data, cdc)

			if tt.expectError {
				require.Error(t, err)
				require.Nil(t, result)
				if tt.expectUnknownErr {
					_, ok := err.(wasmvmtypes.Unknown)
					require.True(t, ok, "Expected wasmvmtypes.Unknown error")
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
			}
		})
	}
}

// Test to specifically cover the JSON marshal error path in ConvertProtoToJSONMarshal
func TestConvertProtoToJSONMarshal_JSONMarshalError(t *testing.T) {
	// Create a custom codec that will fail on JSON marshal
	registry := xionapp.Setup(t).InterfaceRegistry()
	cdc := &failingJSONCodec{codec.NewProtoCodec(registry)}

	protoMessage := &banktypes.QueryBalanceResponse{}

	// Create valid protobuf data
	validData, err := cdc.Marshal(protoMessage)
	require.NoError(t, err)

	// This should fail on JSON marshal
	result, err := wasmbinding.ConvertProtoToJSONMarshal(protoMessage, validData, cdc)

	require.Error(t, err)
	require.Nil(t, result)
	_, ok := err.(wasmvmtypes.Unknown)
	require.True(t, ok, "Expected wasmvmtypes.Unknown error")
}

// failingJSONCodec is a codec that fails on JSON marshal operations
type failingJSONCodec struct {
	codec.Codec
}

func (c *failingJSONCodec) MarshalJSON(o proto.Message) ([]byte, error) {
	return nil, &mockError{msg: "mock JSON marshal error"}
}

type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}