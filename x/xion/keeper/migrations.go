package keeper

import (
	"fmt"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	v1 "github.com/burnt-labs/xion/x/xion/migrations/v1"
	"github.com/burnt-labs/xion/x/xion/types"
)

type Migrator struct {
	wasmOpsKeeper  wasmtypes.ContractOpsKeeper
	wasmViewKeeper wasmtypes.ViewKeeper
	aaKeeper       types.AbstractAccountKeeper
}

func NewMigrator(wasmOpsKeeper wasmtypes.ContractOpsKeeper, wasmViewKeeper wasmtypes.ViewKeeper, aaKeeper types.AbstractAccountKeeper) Migrator {
	return Migrator{wasmOpsKeeper: wasmOpsKeeper, wasmViewKeeper: wasmViewKeeper, aaKeeper: aaKeeper}
}

func (m Migrator) Migrate1to2(ctx sdk.Context) error {
	switch ctx.ChainID() {
	case "xion-mainnet-1":
		return nil // no migration needed
	case "xion-testnet-1":
		newCodeID := uint64(327)
		return v1.MigrateStore(ctx, m.wasmOpsKeeper, m.wasmViewKeeper, m.aaKeeper, newCodeID)
	case "xion-1": // integration tests chainID
		newCodeID := uint64(2)
		return v1.MigrateStore(ctx, m.wasmOpsKeeper, m.wasmViewKeeper, m.aaKeeper, newCodeID)
	default:
		return fmt.Errorf("unsupported chain id: %s", ctx.ChainID())
	}
}
