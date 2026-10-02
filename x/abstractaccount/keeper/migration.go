package keeper

import (
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	v2 "github.com/burnt-labs/xion/x/abstractaccount/migrations/v2"
	v3 "github.com/burnt-labs/xion/x/abstractaccount/migrations/v3"
)

// Migrator is a struct for handling in-place store migrations.
type Migrator struct {
	key storetypes.StoreKey
	cdc codec.BinaryCodec
}

// Migrate2to3 adds fixed-hash registration parameters. Registration stays
// paused until the chain upgrade configures its address derivation hash.
func (m Migrator) Migrate2to3(ctx sdk.Context) error {
	return v3.MigrateStore(ctx, m.key, m.cdc)
}

// NewMigrator returns a new Migrator.
func NewMigrator(key storetypes.StoreKey, cdc codec.BinaryCodec) Migrator {
	return Migrator{key: key, cdc: cdc}
}

// Migrate1to2 migrates from version 1 to 2.
func (m Migrator) Migrate1to2(ctx sdk.Context) error {
	return v2.MigrateStore(ctx, m.key, m.cdc)
}
