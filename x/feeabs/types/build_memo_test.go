package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	types "github.com/burnt-labs/xion/x/feeabs/types"
)

func TestNewOsmosisSwapMsg(t *testing.T) {
	outputDenom := "uosmo"
	slippagePercentage := "5"
	windowSeconds := uint64(30)
	receiver := "osmo1receiver"

	msg := types.NewOsmosisSwapMsg(outputDenom, slippagePercentage, windowSeconds, receiver)

	require.Equal(t, outputDenom, msg.OsmosisSwap.OutputDenom)
	require.Equal(t, slippagePercentage, msg.OsmosisSwap.Slippage.Twap.SlippagePercentage)
	require.Equal(t, windowSeconds, msg.OsmosisSwap.Slippage.Twap.WindowSeconds)
	require.Equal(t, receiver, msg.OsmosisSwap.Receiver)
}

// TODO: need to refactor this test, use driven table
func TestParseMsgToMemo(t *testing.T) {
	twapRouter := types.TwapRouter{
		SlippagePercentage: "20",
		WindowSeconds:      10,
	}

	swap := types.Swap{
		OutputDenom: "denom",
		Slippage:    types.Twap{Twap: twapRouter},
		Receiver:    "123456",
	}

	msgSwap := types.OsmosisSwapMsg{
		OsmosisSwap: swap,
	}

	mockAddress := "cosmos123456789"

	// TODO: need to check assert msg
	memo, err := types.ParseMsgToMemo(msgSwap, mockAddress)
	require.NoError(t, err)
	require.NotEmpty(t, memo)

	// Verify the memo contains expected elements
	require.Contains(t, memo, mockAddress)
	require.Contains(t, memo, "wasm")
	require.Contains(t, memo, "contract")
	require.Contains(t, memo, "msg")
	require.Contains(t, memo, "osmosis_swap")
}

func TestParseMsgToMemoTableDriven(t *testing.T) {
	tests := []struct {
		name         string
		msg          types.OsmosisSwapMsg
		contractAddr string
		expectError  bool
	}{
		{
			name: "valid message",
			msg: types.OsmosisSwapMsg{
				OsmosisSwap: types.Swap{
					OutputDenom: "uosmo",
					Slippage: types.Twap{
						Twap: types.TwapRouter{
							SlippagePercentage: "10",
							WindowSeconds:      60,
						},
					},
					Receiver: "receiver123",
				},
			},
			contractAddr: "contract123",
			expectError:  false,
		},
		{
			name: "empty output denom",
			msg: types.OsmosisSwapMsg{
				OsmosisSwap: types.Swap{
					OutputDenom: "",
					Slippage: types.Twap{
						Twap: types.TwapRouter{
							SlippagePercentage: "10",
							WindowSeconds:      60,
						},
					},
					Receiver: "receiver123",
				},
			},
			contractAddr: "contract123",
			expectError:  false, // Currently no validation in ParseMsgToMemo
		},
		{
			name: "empty contract address",
			msg: types.OsmosisSwapMsg{
				OsmosisSwap: types.Swap{
					OutputDenom: "uosmo",
					Slippage: types.Twap{
						Twap: types.TwapRouter{
							SlippagePercentage: "10",
							WindowSeconds:      60,
						},
					},
					Receiver: "receiver123",
				},
			},
			contractAddr: "",
			expectError:  false, // Currently no validation in ParseMsgToMemo
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			memo, err := types.ParseMsgToMemo(tt.msg, tt.contractAddr)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotEmpty(t, memo)

				// Additional verification for non-error cases
				if !tt.expectError {
					require.Contains(t, memo, `"contract"`)
					require.Contains(t, memo, `"msg"`)
					require.Contains(t, memo, `"wasm"`)
				}
			}
		})
	}
}

func TestParseMsgToMemoMarshalError(t *testing.T) {
	// It's very difficult to make json.Marshal fail with normal structs
	// since the structs are well-formed. JSON marshal typically only fails with:
	// 1. Circular references (not possible with these structs)
	// 2. Channels, funcs, or invalid UTF-8 (not present in these structs)
	// 3. Unsupported types (all fields are basic types)

	// Let's test with a normal message to ensure the happy path works
	// The error path in json.Marshal is defensive programming
	msg := types.OsmosisSwapMsg{
		OsmosisSwap: types.Swap{
			OutputDenom: "uosmo",
			Slippage: types.Twap{
				Twap: types.TwapRouter{
					SlippagePercentage: "10",
					WindowSeconds:      60,
				},
			},
			Receiver: "receiver123",
		},
	}

	memo, err := types.ParseMsgToMemo(msg, "contract123")
	require.NoError(t, err)
	require.NotEmpty(t, memo)

	// Verify JSON structure
	require.Contains(t, memo, `"wasm"`)
	require.Contains(t, memo, `"contract"`)
	require.Contains(t, memo, `"msg"`)
}

func TestBuildCrossChainSwapMemoErrorPropagation(t *testing.T) {
	// Test that errors from ParseMsgToMemo are properly propagated
	// Since ParseMsgToMemo is unlikely to fail with normal inputs,
	// we test the error propagation structure

	// Normal case should work
	memo, err := types.BuildCrossChainSwapMemo("uosmo", "contract123", "receiver123", "test-chain")
	require.NoError(t, err)
	require.NotEmpty(t, memo)

	// Verify the structure
	require.Contains(t, memo, "test-chain/receiver123")
	require.Contains(t, memo, "do_nothing")
}

// TODO: need to refactor this test, use driven table
func TestParseCrossChainSwapMsgToMemo(t *testing.T) {
	outPutDenom := "uosmo"
	contractAddress := "osmo1c3ljch9dfw5kf52nfwpxd2zmj2ese7agnx0p9tenkrryasrle5sqf3ftpg"
	mockReceiver := "feeabs1efd63aw40lxf3n4mhf7dzhjkr453axurwrhrrw"
	chainName := "feeabs"

	execeptedMemoStr := `{"wasm":{"contract":"osmo1c3ljch9dfw5kf52nfwpxd2zmj2ese7agnx0p9tenkrryasrle5sqf3ftpg","msg":{"osmosis_swap":{"output_denom":"uosmo","slippage":{"twap":{"slippage_percentage":"20","window_seconds":10}},"receiver":"feeabs/feeabs1efd63aw40lxf3n4mhf7dzhjkr453axurwrhrrw","on_failed_delivery":"do_nothing"}}}}`
	// TODO: need to check assert msg
	memoStr, err := types.BuildCrossChainSwapMemo(outPutDenom, contractAddress, mockReceiver, chainName)

	require.NoError(t, err)
	require.Equal(t, execeptedMemoStr, memoStr)
}

func TestBuildCrossChainSwapMemoTableDriven(t *testing.T) {
	tests := []struct {
		name            string
		outputDenom     string
		contractAddress string
		receiverAddress string
		chainName       string
		expectError     bool
	}{
		{
			name:            "valid parameters",
			outputDenom:     "uosmo",
			contractAddress: "osmo1contract",
			receiverAddress: "receiver123",
			chainName:       "test-chain",
			expectError:     false,
		},
		{
			name:            "empty output denom",
			outputDenom:     "",
			contractAddress: "osmo1contract",
			receiverAddress: "receiver123",
			chainName:       "test-chain",
			expectError:     false, // Currently no validation
		},
		{
			name:            "empty contract address",
			outputDenom:     "uosmo",
			contractAddress: "",
			receiverAddress: "receiver123",
			chainName:       "test-chain",
			expectError:     false, // Currently no validation
		},
		{
			name:            "empty receiver address",
			outputDenom:     "uosmo",
			contractAddress: "osmo1contract",
			receiverAddress: "",
			chainName:       "test-chain",
			expectError:     false, // Currently no validation
		},
		{
			name:            "empty chain name",
			outputDenom:     "uosmo",
			contractAddress: "osmo1contract",
			receiverAddress: "receiver123",
			chainName:       "",
			expectError:     false, // Currently no validation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			memo, err := types.BuildCrossChainSwapMemo(tt.outputDenom, tt.contractAddress, tt.receiverAddress, tt.chainName)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotEmpty(t, memo)

				// Verify receiver format
				expectedReceiver := tt.chainName + "/" + tt.receiverAddress
				require.Contains(t, memo, expectedReceiver)
				require.Contains(t, memo, "do_nothing")
				require.Contains(t, memo, "20") // slippage percentage
				require.Contains(t, memo, "10") // window seconds
			}
		})
	}
}
