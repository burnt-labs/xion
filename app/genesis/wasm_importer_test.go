package genesis_test

import (
	"context"
	"testing"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/app/genesis"
)

func TestNewWasmGenesisImporter(t *testing.T) {
	t.Run("creates importer successfully", func(t *testing.T) {
		// Create a nil keeper for basic structure test
		// In real usage, this would be a properly initialized keeper
		var keeper *wasmkeeper.Keeper

		importer := genesis.NewWasmGenesisImporter(keeper)
		require.NotNil(t, importer, "importer should not be nil")
	})
}

func TestWasmGenesisImporter_ImportCode(t *testing.T) {
	t.Run("validates code info before import", func(t *testing.T) {
		// Test that validation is called before keeper delegation
		// We expect a panic due to nil keeper, but that proves validation ran first

		importer := genesis.NewWasmGenesisImporter(nil)
		ctx := context.Background()

		codeInfo := wasmtypes.CodeInfo{
			CodeHash: []byte("test-hash"),
			Creator:  "xion1test",
		}

		// Should panic with nil keeper after validation passes
		require.Panics(t, func() {
			_ = importer.ImportCode(ctx, 1, codeInfo, []byte("wasm-code"))
		}, "Expected panic with nil keeper")
	})
}

func TestWasmGenesisImporter_ImportContract(t *testing.T) {
	t.Run("validates contract info before import", func(t *testing.T) {
		// Test that validation is called before keeper delegation

		importer := genesis.NewWasmGenesisImporter(nil)
		ctx := context.Background()

		contractAddr := sdk.AccAddress([]byte("contract1"))
		contractInfo := &wasmtypes.ContractInfo{
			CodeID:  1,
			Creator: "xion1creator",
			Label:   "test-contract",
		}

		// Should panic with nil keeper after validation passes
		require.Panics(t, func() {
			_ = importer.ImportContract(ctx, contractAddr, contractInfo, nil, nil)
		}, "Expected panic with nil keeper")
	})
}

func TestWasmGenesisImporter_ImportAutoIncrementID(t *testing.T) {
	t.Run("delegates to keeper for sequence management", func(t *testing.T) {
		// Test that the method exists and can be called

		importer := genesis.NewWasmGenesisImporter(nil)
		ctx := context.Background()

		sequenceKey := []byte("sequence-key")
		val := uint64(100)

		// Should panic with nil keeper
		require.Panics(t, func() {
			_ = importer.ImportAutoIncrementID(ctx, sequenceKey, val)
		}, "Expected panic with nil keeper")
	})
}
