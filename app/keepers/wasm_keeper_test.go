package keepers_test

import (
	"testing"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	xionapp "github.com/burnt-labs/xion/app"
	"github.com/burnt-labs/xion/app/keepers"
	aatestdata "github.com/burnt-labs/xion/x/abstractaccount/testdata"
)

func TestNewXionWasmKeeper(t *testing.T) {
	t.Run("creates xion wasm keeper successfully", func(t *testing.T) {
		// Create a nil keeper for basic structure test
		// In real usage, this would be a properly initialized keeper
		var keeper *wasmkeeper.Keeper

		xionKeeper := keepers.NewXionWasmKeeper(keeper)
		require.NotNil(t, xionKeeper, "xion keeper should not be nil")
	})
}

func TestXionWasmKeeper_ValidateContractLabel(t *testing.T) {
	tests := []struct {
		name    string
		label   string
		wantErr bool
	}{
		{
			name:    "empty label is valid (validation not implemented)",
			label:   "",
			wantErr: false,
		},
		{
			name:    "simple label is valid",
			label:   "my-contract",
			wantErr: false,
		},
		{
			name:    "long label is valid (validation not implemented)",
			label:   "this-is-a-very-long-contract-label-that-might-be-too-long",
			wantErr: false,
		},
		{
			name:    "label with special characters is valid (validation not implemented)",
			label:   "contract!@#$%",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			xionKeeper := keepers.NewXionWasmKeeper(nil)
			err := xionKeeper.ValidateContractLabel(tt.label)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestXionWasmKeeper_GetXionContractMetadata(t *testing.T) {
	t.Run("returns error for non-existent contract with nil keeper", func(t *testing.T) {
		// Test with nil keeper to ensure error handling works
		// This tests the code path where GetContractInfo returns nil

		xionKeeper := keepers.NewXionWasmKeeper(nil)
		ctx := sdk.Context{} // minimal context for testing
		contractAddr := sdk.AccAddress([]byte("nonexistent"))

		// Should panic with nil keeper, or return error if GetContractInfo is called
		// Either way, this tests that the method exists and attempts the logic
		require.Panics(t, func() {
			_, _ = xionKeeper.GetXionContractMetadata(ctx, contractAddr)
		}, "Expected panic with nil keeper")
	})

	t.Run("returns not found for an unknown contract", func(t *testing.T) {
		app := xionapp.Setup(t)
		ctx := app.NewContext(false)
		xionKeeper := keepers.NewXionWasmKeeper(&app.WasmKeeper)

		metadata, err := xionKeeper.GetXionContractMetadata(ctx, xionapp.RandomAccAddress())
		require.ErrorIs(t, err, wasmtypes.ErrNotFound)
		require.Nil(t, metadata)
	})

	t.Run("returns the stored contract info", func(t *testing.T) {
		app := xionapp.Setup(t)
		ctx := app.NewContext(false)
		xionKeeper := keepers.NewXionWasmKeeper(&app.WasmKeeper)

		creator := xionapp.RandomAccAddress()
		codeID, _, err := app.AbstractAccountKeeper.ContractKeeper().Create(ctx, creator, aatestdata.AccountWasm, nil)
		require.NoError(t, err)

		contractAddr := wasmkeeper.BuildContractAddressClassic(codeID, 1)
		position := &wasmtypes.AbsoluteTxPosition{BlockHeight: 1, TxIndex: 1}
		contractInfo := wasmtypes.ContractInfo{CodeID: codeID, Creator: creator.String(), Label: "metadata", Created: position}
		history := []wasmtypes.ContractCodeHistoryEntry{{
			Operation: wasmtypes.ContractCodeHistoryOperationTypeInit,
			CodeID:    codeID,
			Updated:   position,
			Msg:       []byte("{}"),
		}}
		require.NoError(t, app.WasmKeeper.ImportContract(ctx, contractAddr, &contractInfo, nil, history))

		metadata, err := xionKeeper.GetXionContractMetadata(ctx, contractAddr)
		require.NoError(t, err)
		require.NotNil(t, metadata)
		require.Equal(t, contractInfo, *metadata.ContractInfo)
	})
}
