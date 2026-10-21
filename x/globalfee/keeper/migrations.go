package keeper

import (
	corestore "cosmossdk.io/core/store"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	v3 "github.com/burnt-labs/xion/x/globalfee/migrations/v3"
)

type Migrator struct {
	k            Keeper
	legacyParams corestore.KVStoreService
	amino        *codec.LegacyAmino
}

func NewMigrator(k Keeper, legacyParams corestore.KVStoreService, amino *codec.LegacyAmino) Migrator {
	return Migrator{k: k, legacyParams: legacyParams, amino: amino}
}

func (m Migrator) Migrate2to3(ctx sdk.Context) error {
	return v3.MigrateStore(ctx, m.k, m.legacyParams, m.amino)
}
