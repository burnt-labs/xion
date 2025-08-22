package types_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"

	"github.com/burnt-labs/xion/x/globalfee/types"
)

func TestGlobalFeeGenesis(t *testing.T) {
	// Test DefaultGenesisState
	defaultGenesis := types.DefaultGenesisState()
	require.NotNil(t, defaultGenesis)
	require.Equal(t, types.DefaultParams(), defaultGenesis.Params)

	// Test NewGenesisState
	params := types.DefaultParams()
	genesis := types.NewGenesisState(params)
	require.NotNil(t, genesis)
	require.Equal(t, params, genesis.Params)

	// Test GetGenesisStateFromAppState
	appState := map[string]interface{}{
		types.ModuleName: map[string]interface{}{
			"params": map[string]interface{}{
				"minimum_gas_prices":                     []interface{}{},
				"bypass_min_fee_msg_types":               []interface{}{},
				"max_total_bypass_min_fee_msg_gas_usage": uint64(1000000),
			},
		},
	}

	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	appStateBytes := make(map[string]json.RawMessage)
	for k, v := range appState {
		if b, err := json.Marshal(v); err == nil {
			appStateBytes[k] = b
		}
	}
	genesisState := types.GetGenesisStateFromAppState(cdc, appStateBytes)
	require.NotNil(t, genesisState)

	// Test ValidateGenesis
	err := types.ValidateGenesis(*defaultGenesis)
	require.NoError(t, err)
}

func TestGlobalFeeParams(t *testing.T) {
	// Test DefaultParams
	params := types.DefaultParams()
	require.NotNil(t, params)
	require.NotNil(t, params.MinimumGasPrices)
	require.NotNil(t, params.BypassMinFeeMsgTypes)
	require.Equal(t, uint64(1_000_000), params.MaxTotalBypassMinFeeMsgGasUsage)

	// Test ParamKeyTable
	keyTable := types.ParamKeyTable()
	require.NotNil(t, keyTable)

	// Test ValidateBasic
	err := params.ValidateBasic()
	require.NoError(t, err)

	// Test ParamSetPairs
	pairs := params.ParamSetPairs()
	require.NotNil(t, pairs)
	require.Len(t, pairs, 3) // MinimumGasPrices, BypassMinFeeMsgTypes, MaxTotalBypassMinFeeMsgGasUsage
}

func TestGlobalFeeCodec(t *testing.T) {
	// Test codec instantiation
	amino := codec.NewLegacyAmino()
	require.NotNil(t, amino)

	// Test interface registry
	registry := codectypes.NewInterfaceRegistry()
	require.NotNil(t, registry)
}
