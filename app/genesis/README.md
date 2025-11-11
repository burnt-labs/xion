# Xion Genesis Package

This package contains Xion-specific genesis import and export functionality.

## Purpose

The genesis package provides wrappers around standard Cosmos SDK module genesis operations, allowing Xion to:

1. **Add custom validation** during chain initialization
2. **Extend genesis functionality** without modifying upstream dependencies
3. **Maintain clear separation** between module code and application code
4. **Simplify upgrades** by keeping customizations in the Xion codebase

## Architecture

### Why Not Modify wasmd Directly?

Instead of adding Xion-specific code to the wasmd fork, we use the **Adapter Pattern**:

```
┌─────────────────────────────────────────┐
│  Xion Application Layer                 │
│  ┌───────────────────────────────────┐  │
│  │ app/genesis/wasm_importer.go      │  │  ← Xion-specific logic
│  │ - Validation                      │  │
│  │ - Transformations                 │  │
│  │ - Business logic                  │  │
│  └─────────────┬─────────────────────┘  │
│                │ wraps                   │
│                ▼                         │
│  ┌───────────────────────────────────┐  │
│  │ wasmd keeper                      │  │  ← Standard module
│  │ (upstream or minimal fork)        │  │
│  └───────────────────────────────────┘  │
└─────────────────────────────────────────┘
```

### Benefits

| Aspect | wasmd Fork with xion_exports.go | Xion Genesis Package |
|--------|--------------------------------|---------------------|
| **Separation of Concerns** | ❌ Mixed | ✅ Clean |
| **Upgrade Complexity** | ❌ Must maintain exports | ✅ Only rebase wasmd |
| **Testing** | ⚠️ Harder to test | ✅ Easy to mock |
| **Cosmos SDK Patterns** | ❌ Non-standard | ✅ Standard |
| **Flexibility** | ❌ Limited | ✅ Full control |

## Usage

### During Genesis Import

```go
import (
    xiongenesis "github.com/burnt-labs/xion/app/genesis"
)

// In your InitChainer or genesis initialization
func (app *XionApp) InitChainer(ctx sdk.Context, req *abci.RequestInitChain) (*abci.ResponseInitChain, error) {
    // Create Xion-specific genesis importer
    wasmImporter := xiongenesis.NewWasmGenesisImporter(&app.WasmKeeper)

    // Import wasm code with Xion validation
    if err := wasmImporter.ImportCode(ctx, codeID, codeInfo, wasmCode); err != nil {
        return nil, err
    }

    // Import contracts with Xion validation
    if err := wasmImporter.ImportContract(ctx, contractAddr, contractInfo, state, history); err != nil {
        return nil, err
    }

    // ... rest of genesis
}
```

### Custom Validation

The `WasmGenesisImporter` provides hooks for Xion-specific validation:

1. **`validateCodeForXion`**: Validates wasm code during import
2. **`validateContractForXion`**: Validates contract metadata during import

Extend these methods to add your custom business logic:

```go
// In wasm_importer.go
func (w *WasmGenesisImporter) validateCodeForXion(ctx context.Context, info wasmtypes.CodeInfo) error {
    // Example: Ensure only certain addresses can create code
    if !isAllowedCreator(info.Creator) {
        return fmt.Errorf("creator %s not allowed during genesis", info.Creator)
    }

    // Example: Validate instantiate permissions
    if info.InstantiateConfig.Permission == wasmtypes.AccessTypeEverybody {
        return fmt.Errorf("open instantiation not allowed in Xion")
    }

    return nil
}
```

## Implementation Notes

### Dependencies on wasmd

This package assumes that wasmd has exported the following methods as public:

- `keeper.ImportCode()`
- `keeper.ImportContract()`
- `keeper.ImportAutoIncrementID()`

If these methods are not public in upstream wasmd, you need one of:

1. **Preferred**: Submit PR to wasmd to make them public
2. **Alternative**: Maintain minimal wasmd fork with public exports
3. **Workaround**: Reimplement the logic (not recommended - high maintenance)

### Genesis Optimization

The wasmd fork includes a critical optimization for genesis import:

**File**: `wasmd/x/wasm/keeper/keeper.go` (line ~1427)

```go
// OLD (slow - iterates existing history)
err = k.appendToContractHistory(ctx, contractAddr, historyEntries...)

// NEW (fast - assumes genesis empty state)
err = k.appendToContractHistoryGenesis(ctx, contractAddr, historyEntries...)
```

This optimization provides ~5x speedup during genesis import by avoiding unnecessary
iteration over existing contract history entries (which are guaranteed to be empty
during genesis).

## Testing

Tests for genesis import should:

1. Use the Xion test helpers from `app/test_helpers.go`
2. Mock wasmd keeper behavior
3. Validate Xion-specific business logic
4. Test error cases and edge conditions

Example test structure:

```go
func TestWasmGenesisImporter_ImportCode(t *testing.T) {
    // Setup
    app := SetupXionApp(t)
    importer := genesis.NewWasmGenesisImporter(&app.WasmKeeper)

    // Test valid import
    err := importer.ImportCode(ctx, codeID, validCodeInfo, wasmCode)
    require.NoError(t, err)

    // Test invalid import (Xion validation fails)
    err = importer.ImportCode(ctx, codeID, invalidCodeInfo, wasmCode)
    require.Error(t, err)
}
```

## Future Enhancements

### Planned Features

1. **Metrics**: Track genesis import performance
2. **Validation Library**: Common validation patterns for contracts
3. **Migration Support**: Tools for migrating genesis state between versions
4. **Export Optimization**: Fast genesis export similar to import optimization

### Extension Points

You can extend the importer with:

- Pre-import hooks (validation, transformation)
- Post-import hooks (indexing, caching)
- Custom error handling and retry logic
- Progress tracking for large genesis files

## Related Documentation

- [wasmd Genesis](https://github.com/CosmWasm/wasmd/blob/main/x/wasm/keeper/genesis.go)
- [Cosmos SDK Genesis](https://docs.cosmos.network/main/core/genesis)
- [Xion Architecture](../../docs/architecture/)
- [wasmd Fork Alternatives](../../docs/WASMD_FORK_ALTERNATIVES.md)

## Migration Guide

If you're migrating from the old `xion_exports.go` pattern in the wasmd fork:

### Before (in wasmd fork):
```go
// wasmd/x/wasm/keeper/xion_exports.go
func (k Keeper) ImportCode(...) { ... }
```

### After (in Xion repo):
```go
// xion/app/genesis/wasm_importer.go
func (w *WasmGenesisImporter) ImportCode(...) {
    // Xion validation
    // Call k.keeper.ImportCode(...)  // wasmd public method
}
```

### Benefits of Migration:
- ✅ Cleaner wasmd fork (easier to maintain)
- ✅ Xion-specific logic in Xion repo
- ✅ Easier to contribute wasmd changes upstream
- ✅ Better testing and mocking capabilities
