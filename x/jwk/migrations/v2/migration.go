package v2

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/burnt-labs/xion/x/jwk/types"
)

// MigrateStore performs in-place params migrations of
// TimeOffset and DeploymentGas
//
// this should correct a previous migration
func MigrateStore(ctx sdk.Context, jwkSubspace paramtypes.Subspace) error {
	defaultParams := types.DefaultParams()

	if !jwkSubspace.HasKeyTable() {
		jwkSubspace = jwkSubspace.WithKeyTable(types.ParamKeyTable())
	}

	jwkSubspace.SetParamSet(ctx, &defaultParams)
	return nil
}
