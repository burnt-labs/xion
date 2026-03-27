package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	v1 "github.com/burnt-labs/xion/x/jwk/migrations/v1"
	v3 "github.com/burnt-labs/xion/x/jwk/migrations/v3"
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

// Migrate2To3 migrates from version 2 to 3.
// It corrects the TimeOffset param that was set to 30_000 (30 microseconds)
// by Migrate1To2 instead of the correct 30_000_000_000 (30 seconds in nanoseconds).
func (m Migrator) Migrate2To3(ctx sdk.Context) error {
	return v3.MigrateStore(ctx, m.jwkSubspace)
}
