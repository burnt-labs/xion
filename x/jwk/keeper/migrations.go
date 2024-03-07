package keeper

import (
	v1 "github.com/burnt-labs/xion/x/jwk/migrations/v1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

// Migrator is a struct for handling in-place store migrations.
type Migrator struct {
	jwkSubspace paramtypes.Subspace
}

// NewMigrator returns a new Migrator.
func NewMigrator(jwkSubspace paramtypes.Subspace) Migrator {
	return Migrator{jwkSubspace}
}

// Migrate1to2 migrates from version to 1
func (m Migrator) MigrateTo1(ctx sdk.Context) error {
	return v1.MigrateStore(ctx, m.jwkSubspace)
}
