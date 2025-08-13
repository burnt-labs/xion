package types_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/xion/types"
)

func TestGenesisState_Validate(t *testing.T) {
	tests := []struct {
		name         string
		genesisState types.GenesisState
		expectError  bool
		errorMsg     string
	}{
		{
			name: "valid genesis state",
			genesisState: types.GenesisState{
				PlatformPercentage: 5000, // 50%
				PlatformMinimums:   sdk.NewCoins(sdk.NewInt64Coin("usd", 100)),
			},
			expectError: false,
		},
		{
			name: "valid genesis state at maximum percentage",
			genesisState: types.GenesisState{
				PlatformPercentage: 10000, // 100%
				PlatformMinimums:   sdk.NewCoins(),
			},
			expectError: false,
		},
		{
			name: "invalid genesis state - percentage too high",
			genesisState: types.GenesisState{
				PlatformPercentage: 10001, // 100.01%
				PlatformMinimums:   sdk.NewCoins(),
			},
			expectError: true,
			errorMsg:    "unable to set platform percentage to greater than 100%",
		},
		{
			name: "zero percentage is valid",
			genesisState: types.GenesisState{
				PlatformPercentage: 0,
				PlatformMinimums:   sdk.NewCoins(),
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.genesisState.Validate()
			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNewGenesisState(t *testing.T) {
	platformPercentage := uint32(2500) // 25%
	platformMinimums := sdk.NewCoins(sdk.NewInt64Coin("atom", 1000))

	genesisState := types.NewGenesisState(platformPercentage, platformMinimums)

	require.NotNil(t, genesisState)
	require.Equal(t, platformPercentage, genesisState.PlatformPercentage)
	require.Equal(t, platformMinimums, genesisState.PlatformMinimums)
}

func TestDefaultGenesisState(t *testing.T) {
	defaultState := types.DefaultGenesisState()

	require.NotNil(t, defaultState)
	require.Equal(t, uint32(0), defaultState.PlatformPercentage)
	require.True(t, defaultState.PlatformMinimums.IsZero())
}

func TestGetGenesisStateFromAppState(t *testing.T) {
	tests := []struct {
		name     string
		appState map[string]json.RawMessage
		expected types.GenesisState
	}{
		{
			name: "valid app state with xion module",
			appState: map[string]json.RawMessage{
				types.ModuleName: json.RawMessage(`{"platform_percentage": 1000, "platform_minimums": []}`),
			},
			expected: types.GenesisState{
				PlatformPercentage: 1000,
				PlatformMinimums:   sdk.NewCoins(),
			},
		},
		{
			name: "app state without xion module",
			appState: map[string]json.RawMessage{
				"bank": json.RawMessage(`{}`),
			},
			expected: types.GenesisState{
				PlatformPercentage: 0,
				PlatformMinimums:   nil,
			},
		},
	}

	cdc := codec.NewProtoCodec(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			genesisState := types.GetGenesisStateFromAppState(cdc, tt.appState)

			require.NotNil(t, genesisState)
			require.Equal(t, tt.expected.PlatformPercentage, genesisState.PlatformPercentage)
			if tt.expected.PlatformMinimums != nil {
				require.Equal(t, tt.expected.PlatformMinimums, genesisState.PlatformMinimums)
			}
		})
	}

	// Test panic case with invalid JSON
	t.Run("invalid JSON causes panic", func(t *testing.T) {
		appState := map[string]json.RawMessage{
			types.ModuleName: json.RawMessage(`invalid json`),
		}

		require.Panics(t, func() {
			types.GetGenesisStateFromAppState(cdc, appState)
		})
	})
}
