package v2

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/burnt-labs/xion/x/jwk/types"
)

// MigrateStore sets TimeOffset to the default value (30 seconds in nanoseconds)
// for chains that were initialised before TimeOffset was introduced.
func MigrateStore(ctx sdk.Context, jwkSubspace paramtypes.Subspace) error {
	ctx.Logger().Info("Running x/jwk Migration v2 -> v3: setting TimeOffset default")

	if !jwkSubspace.HasKeyTable() {
		jwkSubspace = jwkSubspace.WithKeyTable(types.ParamKeyTable())
	}

	// Only set TimeOffset if it is currently unset (zero), so we do not
	// overwrite an operator-configured value on chains that already have it.
	var current uint64
	if jwkSubspace.Has(ctx, types.ParamStoreKeyTimeOffset) {
		jwkSubspace.Get(ctx, types.ParamStoreKeyTimeOffset, &current)
	}

	if current == 0 {
		timeOffset := uint64(30_000_000_000) // 30 seconds in nanoseconds
		jwkSubspace.Set(ctx, types.ParamStoreKeyTimeOffset, timeOffset)
		ctx.Logger().Info(fmt.Sprintf("x/jwk: set TimeOffset to %d", timeOffset))
	} else {
		ctx.Logger().Info(fmt.Sprintf("x/jwk: TimeOffset already set to %d, skipping", current))
	}

	return nil
}
