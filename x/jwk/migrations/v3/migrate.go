package v3

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/burnt-labs/xion/x/jwk/types"
)

// MigrateStore corrects the TimeOffset param that was set to 30_000 (30
// microseconds) by the v1→v2 migration instead of the correct value of
// 30_000_000_000 (30 seconds in nanoseconds).
func MigrateStore(ctx sdk.Context, jwkSubspace paramtypes.Subspace) error {
	ctx.Logger().Info("Running x/jwk Migration v2 to v3: correcting TimeOffset")

	if !jwkSubspace.HasKeyTable() {
		jwkSubspace = jwkSubspace.WithKeyTable(types.ParamKeyTable())
	}

	correctTimeOffset := uint64(30_000_000_000) // 30 seconds in nanoseconds
	ctx.Logger().Info(fmt.Sprintf("setting TimeOffset to %d", correctTimeOffset))
	jwkSubspace.Set(ctx, types.ParamStoreKeyTimeOffset, correctTimeOffset)

	return nil
}
