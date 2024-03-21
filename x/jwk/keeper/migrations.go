package keeper

import (
	v1 "github.com/burnt-labs/xion/x/jwk/migrations/v1"
	v2 "github.com/burnt-labs/xion/x/jwk/migrations/v2"
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

// Migrate1To2 migrates from version 1 to 2
func (m Migrator) Migrate1To2(ctx sdk.Context) error {
	return v1.MigrateStore(ctx, m.jwkSubspace)
}

// Migrate2To3 migrates from version 2 to 3
func (m Migrator) Migrate2To3(ctx sdk.Context) error {
	return v2.MigrateStore(ctx, m.jwkSubspace)
}
