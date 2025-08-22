package types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestNewGenesisState(t *testing.T) {
	params := DefaultParams()
	genesisState := NewGenesisState(params)
	require.NotNil(t, genesisState)
	require.Equal(t, params, genesisState.Params)
}

func TestDefaultGenesisState(t *testing.T) {
	genesisState := DefaultGenesisState()
	require.NotNil(t, genesisState)
	require.Equal(t, DefaultParams(), genesisState.Params)
}

func TestGetGenesisStateFromAppState(t *testing.T) {
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

	// Test with valid genesis state
	genesisState := DefaultGenesisState()
	appState := map[string]json.RawMessage{
		ModuleName: cdc.MustMarshalJSON(genesisState),
	}

	result := GetGenesisStateFromAppState(cdc, appState)
	require.NotNil(t, result)
	require.Equal(t, genesisState.Params, result.Params)

	// Test with missing module state (should return default)
	emptyAppState := map[string]json.RawMessage{}
	result = GetGenesisStateFromAppState(cdc, emptyAppState)
	require.NotNil(t, result)
	require.Equal(t, DefaultGenesisState().Params, result.Params)
}

func TestValidateGenesis(t *testing.T) {
	tests := map[string]struct {
		genesisState GenesisState
		expectErr    bool
	}{
		"default genesis state, pass": {
			*DefaultGenesisState(),
			false,
		},
		"valid custom genesis state, pass": {
			GenesisState{
				Params: Params{
					MinimumGasPrices: sdk.DecCoins{
						sdk.NewDecCoin("atom", math.NewInt(1000)),
						sdk.NewDecCoin("stake", math.NewInt(2000)),
					},
					BypassMinFeeMsgTypes:            []string{"/cosmos.bank.v1beta1.MsgSend"},
					MaxTotalBypassMinFeeMsgGasUsage: 500_000,
				},
			},
			false,
		},
		"invalid params, fail": {
			GenesisState{
				Params: Params{
					MinimumGasPrices: sdk.DecCoins{
						sdk.NewDecCoin("photon", math.OneInt()),
						sdk.NewDecCoin("atom", math.OneInt()), // Not sorted
					},
					BypassMinFeeMsgTypes:            DefaultBypassMinFeeMsgTypes,
					MaxTotalBypassMinFeeMsgGasUsage: DefaultmaxTotalBypassMinFeeMsgGasUsage,
				},
			},
			true,
		},
		"invalid bypass msg types, fail": {
			GenesisState{
				Params: Params{
					MinimumGasPrices:                sdk.DecCoins{},
					BypassMinFeeMsgTypes:            []string{"invalid"},
					MaxTotalBypassMinFeeMsgGasUsage: DefaultmaxTotalBypassMinFeeMsgGasUsage,
				},
			},
			true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := ValidateGenesis(test.genesisState)
			if test.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
