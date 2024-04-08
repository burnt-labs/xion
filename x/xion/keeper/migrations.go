package keeper

import (
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
	return v1.MigrateStore(ctx, m.wasmOpsKeeper, m.wasmViewKeeper, m.aaKeeper)
}
