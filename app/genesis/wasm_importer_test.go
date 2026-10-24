package genesis_test

import (
	"context"
	"crypto/sha256"
	"testing"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	xionapp "github.com/burnt-labs/xion/app"
	"github.com/burnt-labs/xion/app/genesis"
	aatestdata "github.com/burnt-labs/xion/x/abstractaccount/testdata"
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

// TestWasmGenesisImporterWithKeeper imports code, a contract and a sequence
// through a real keeper and reads each back.
func TestWasmGenesisImporterWithKeeper(t *testing.T) {
	app := xionapp.Setup(t)
	ctx := app.NewContext(false)
	importer := genesis.NewWasmGenesisImporter(&app.WasmKeeper)

	const codeID = uint64(7)
	creator := xionapp.RandomAccAddress()
	checksum := sha256.Sum256(aatestdata.AccountWasm)
	codeInfo := wasmtypes.CodeInfo{
		CodeHash:          checksum[:],
		Creator:           creator.String(),
		InstantiateConfig: wasmtypes.AllowEverybody,
	}
	require.NoError(t, importer.ImportCode(ctx, codeID, codeInfo, aatestdata.AccountWasm))
	require.Equal(t, &codeInfo, app.WasmKeeper.GetCodeInfo(ctx, codeID))

	contractAddr := wasmkeeper.BuildContractAddressClassic(codeID, 1)
	position := &wasmtypes.AbsoluteTxPosition{BlockHeight: 1, TxIndex: 1}
	contractInfo := wasmtypes.ContractInfo{CodeID: codeID, Creator: creator.String(), Label: "imported", Created: position}
	history := []wasmtypes.ContractCodeHistoryEntry{{
		Operation: wasmtypes.ContractCodeHistoryOperationTypeGenesis,
		CodeID:    codeID,
		Updated:   position,
		Msg:       []byte("{}"),
	}}
	state := []wasmtypes.Model{{Key: []byte("config"), Value: []byte(`{"owner":"nobody"}`)}}
	require.NoError(t, importer.ImportContract(ctx, contractAddr, &contractInfo, state, history))
	require.Equal(t, &contractInfo, app.WasmKeeper.GetContractInfo(ctx, contractAddr))
	require.Equal(t, state[0].Value, app.WasmKeeper.QueryRaw(ctx, contractAddr, state[0].Key))

	// Code 7 is the highest imported id, so the code sequence has to resume above it.
	require.NoError(t, importer.ImportAutoIncrementID(ctx, wasmtypes.KeySequenceCodeID, codeID+1))
	next, err := app.WasmKeeper.PeekAutoIncrementID(ctx, wasmtypes.KeySequenceCodeID)
	require.NoError(t, err)
	require.Equal(t, codeID+1, next)
	require.ErrorIs(t, importer.ImportAutoIncrementID(ctx, wasmtypes.KeySequenceCodeID, codeID+2), wasmtypes.ErrDuplicate)
}
