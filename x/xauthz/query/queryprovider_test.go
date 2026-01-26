package query_test

import (
	"context"
	"testing"

	storetypes "cosmossdk.io/store/types"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/burnt-labs/xion/x/xauthz/query"
)

// mockQuerier implements ContractInfoQuerier for testing
type mockQuerier struct {
	contractInfo *wasmtypes.ContractInfo
}

func (m *mockQuerier) GetContractInfo(_ context.Context, _ sdk.AccAddress) *wasmtypes.ContractInfo {
	return m.contractInfo
}

func TestNewProvider(t *testing.T) {
	t.Run("panics with nil keeper", func(t *testing.T) {
		require.Panics(t, func() {
			query.NewProvider(nil)
		})
	})
}

func TestNewProviderWithWasmQuerier(t *testing.T) {
	t.Run("panics with nil wasm querier", func(t *testing.T) {
		require.Panics(t, func() {
			query.NewProviderWithWasmQuerier(nil)
		})
	})

	t.Run("creates provider with valid querier", func(t *testing.T) {
		querier := &mockQuerier{}
		provider := query.NewProviderWithWasmQuerier(querier)
		require.NotNil(t, provider)
	})
}

// TestQueryContractInfoGasConsumption verifies that QueryContractInfo properly
// consumes gas from the context's gas meter. This test uses the actual function
// to prove that gas consumption works correctly through UnwrapSDKContext.
func TestQueryContractInfoGasConsumption(t *testing.T) {
	t.Run("consumes gas on successful query", func(t *testing.T) {
		// Create a mock querier that returns valid contract info
		querier := &mockQuerier{
			contractInfo: &wasmtypes.ContractInfo{
				CodeID: 1,
			},
		}
		provider := query.NewProviderWithWasmQuerier(querier)

		// Create a context with a gas meter
		gasMeter := storetypes.NewGasMeter(1_000_000)
		ctx := sdk.Context{}.WithGasMeter(gasMeter)

		initialGas := gasMeter.GasConsumed()
		require.Equal(t, uint64(0), initialGas)

		// Call the actual QueryContractInfo function
		_, err := provider.QueryContractInfo(ctx, "cosmos1hsk6jryyqjfhp5dhc55tc9jtckygx0eph6dd02")
		require.NoError(t, err)

		// Verify gas was consumed
		consumedGas := gasMeter.GasConsumed()
		require.Equal(t, query.GasCostPerQuery, consumedGas,
			"QueryContractInfo should consume exactly GasCostPerQuery gas")
	})

	t.Run("consumes gas even when query fails with invalid address", func(t *testing.T) {
		querier := &mockQuerier{}
		provider := query.NewProviderWithWasmQuerier(querier)

		// Create a context with a gas meter
		gasMeter := storetypes.NewGasMeter(1_000_000)
		ctx := sdk.Context{}.WithGasMeter(gasMeter)

		// Call with invalid address - should fail but still consume gas
		_, err := provider.QueryContractInfo(ctx, "invalid-address")
		require.Error(t, err)
		require.Contains(t, err.Error(), "bech32")

		// Gas should still be consumed before the error
		consumedGas := gasMeter.GasConsumed()
		require.Equal(t, query.GasCostPerQuery, consumedGas,
			"gas should be consumed even when query fails")
	})

	t.Run("consumes gas even when contract not found", func(t *testing.T) {
		// Mock querier returns nil (contract not found)
		querier := &mockQuerier{contractInfo: nil}
		provider := query.NewProviderWithWasmQuerier(querier)

		gasMeter := storetypes.NewGasMeter(1_000_000)
		ctx := sdk.Context{}.WithGasMeter(gasMeter)

		_, err := provider.QueryContractInfo(ctx, "cosmos1hsk6jryyqjfhp5dhc55tc9jtckygx0eph6dd02")
		require.Error(t, err)
		require.Contains(t, err.Error(), "empty contract information")

		// Gas should still be consumed
		consumedGas := gasMeter.GasConsumed()
		require.Equal(t, query.GasCostPerQuery, consumedGas,
			"gas should be consumed even when contract not found")
	})

	t.Run("panics with insufficient gas", func(t *testing.T) {
		querier := &mockQuerier{
			contractInfo: &wasmtypes.ContractInfo{CodeID: 1},
		}
		provider := query.NewProviderWithWasmQuerier(querier)

		// Create a context with less gas than required
		insufficientGas := query.GasCostPerQuery - 1
		gasMeter := storetypes.NewGasMeter(insufficientGas)
		ctx := sdk.Context{}.WithGasMeter(gasMeter)

		// Should panic with out of gas
		require.Panics(t, func() {
			_, _ = provider.QueryContractInfo(ctx, "cosmos1hsk6jryyqjfhp5dhc55tc9jtckygx0eph6dd02")
		}, "should panic when gas meter has insufficient gas")
	})

	t.Run("multiple queries consume cumulative gas", func(t *testing.T) {
		querier := &mockQuerier{
			contractInfo: &wasmtypes.ContractInfo{CodeID: 1},
		}
		provider := query.NewProviderWithWasmQuerier(querier)

		gasMeter := storetypes.NewGasMeter(1_000_000)
		ctx := sdk.Context{}.WithGasMeter(gasMeter)

		// Execute 3 queries
		for i := 0; i < 3; i++ {
			_, err := provider.QueryContractInfo(ctx, "cosmos1hsk6jryyqjfhp5dhc55tc9jtckygx0eph6dd02")
			require.NoError(t, err)
		}

		// Verify cumulative gas consumption
		expectedGas := query.GasCostPerQuery * 3
		consumedGas := gasMeter.GasConsumed()
		require.Equal(t, expectedGas, consumedGas,
			"multiple queries should consume cumulative gas")
	})
}
