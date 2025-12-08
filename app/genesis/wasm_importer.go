package genesis

import (
	"context"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// WasmGenesisImporter provides Xion-specific genesis import functionality
// for wasm contracts. It wraps the wasmd keeper with additional validation
// and logging specific to Xion's needs.
//
// This wrapper allows Xion to:
//  1. Add custom validation during genesis import
//  2. Extend genesis functionality without modifying wasmd
//  3. Maintain clear separation between module code and app code
//  4. Easily upgrade wasmd without conflicts
type WasmGenesisImporter struct {
	keeper *wasmkeeper.Keeper
}

// NewWasmGenesisImporter creates a new genesis importer for Xion.
// This should be used during genesis initialization to import wasm code
// and contracts with Xion-specific validation and handling.
func NewWasmGenesisImporter(keeper *wasmkeeper.Keeper) *WasmGenesisImporter {
	return &WasmGenesisImporter{keeper: keeper}
}

// ImportCode imports compiled Wasm code during genesis initialization.
// This method wraps wasmd's ImportCode with Xion-specific validation.
//
// Parameters:
//   - ctx: The context for the import operation
//   - codeID: The unique identifier for this code
//   - codeInfo: Metadata about the code (creator, instantiate permissions, etc.)
//   - wasmCode: The compiled WebAssembly bytecode
//
// Returns an error if:
//   - The code fails Xion-specific validation
//   - The underlying wasmd import fails
func (w *WasmGenesisImporter) ImportCode(
	ctx context.Context,
	codeID uint64,
	codeInfo wasmtypes.CodeInfo,
	wasmCode []byte,
) error {
	// Add Xion-specific validation
	if err := w.validateCodeForXion(ctx, codeInfo); err != nil {
		return err
	}

	// Use wasmd keeper's public method
	// This assumes wasmd has exported ImportCode as a public method
	return w.keeper.ImportCode(ctx, codeID, codeInfo, wasmCode)
}

// ImportContract imports a contract instance during genesis initialization.
// This method wraps wasmd's ImportContract with Xion-specific validation.
//
// Parameters:
//   - ctx: The context for the import operation
//   - contractAddr: The address of the contract being imported
//   - contractInfo: Contract metadata (code ID, creator, admin, label, etc.)
//   - state: The contract's storage state as key-value pairs
//   - historyEntries: The contract's code migration history
//
// Returns an error if:
//   - The contract fails Xion-specific validation
//   - The underlying wasmd import fails
func (w *WasmGenesisImporter) ImportContract(
	ctx context.Context,
	contractAddr sdk.AccAddress,
	contractInfo *wasmtypes.ContractInfo,
	state []wasmtypes.Model,
	historyEntries []wasmtypes.ContractCodeHistoryEntry,
) error {
	// Add Xion-specific validation or transformations
	if err := w.validateContractForXion(ctx, contractInfo); err != nil {
		return err
	}

	// Use wasmd keeper's public method
	// This assumes wasmd has exported ImportContract as a public method
	return w.keeper.ImportContract(ctx, contractAddr, contractInfo, state, historyEntries)
}

// ImportAutoIncrementID imports sequence counter values during genesis initialization.
// This ensures that auto-generated IDs (for code and contracts) start from the correct value.
//
// Parameters:
//   - ctx: The context for the import operation
//   - sequenceKey: The key identifying which sequence counter to set
//   - val: The value to set the counter to
//
// Returns an error if the underlying wasmd import fails.
func (w *WasmGenesisImporter) ImportAutoIncrementID(
	ctx context.Context,
	sequenceKey []byte,
	val uint64,
) error {
	// Use wasmd keeper's public method
	// This assumes wasmd has exported ImportAutoIncrementID as a public method
	return w.keeper.ImportAutoIncrementID(ctx, sequenceKey, val)
}

// validateCodeForXion performs Xion-specific validation on code being imported.
// This is where you can add custom business logic for code validation.
//
// Examples of potential validation:
//   - Check if instantiate permissions are appropriate for Xion
//   - Validate code creator addresses
//   - Check code size limits
//   - Verify code hash against known safe contracts
func (w *WasmGenesisImporter) validateCodeForXion(ctx context.Context, info wasmtypes.CodeInfo) error {
	// TODO: Add Xion-specific code validation as needed
	// For example:
	// - Validate instantiate permissions
	// - Check creator address format
	// - Verify code meets Xion requirements

	return nil
}

// validateContractForXion performs Xion-specific validation on contracts being imported.
// This is where you can add custom business logic for contract validation.
//
// Examples of potential validation:
//   - Verify admin addresses are properly formatted
//   - Check contract labels meet Xion standards
//   - Validate code ID references exist
//   - Ensure contract state meets Xion requirements
func (w *WasmGenesisImporter) validateContractForXion(ctx context.Context, info *wasmtypes.ContractInfo) error {
	// TODO: Add Xion-specific contract validation as needed
	// For example:
	// - Validate admin address if set
	// - Check label format
	// - Verify creator address
	// - Validate code ID exists

	return nil
}
