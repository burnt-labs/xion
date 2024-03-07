package v1

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/burnt-labs/xion/x/globalfee/types"
)

// MigrateStore performs in-place params migrations of
// BypassMinFeeMsgTypes and MaxTotalBypassMinFeeMsgGasUsage
// from app.toml to globalfee params.
// The migration includes:
// Add bypass-min-fee-msg-types params that are set
func MigrateStore(ctx sdk.Context, jwkSubspace paramtypes.Subspace) error {
	defaultParams := types.DefaultParams()

	if !jwkSubspace.HasKeyTable() {
		jwkSubspace = jwkSubspace.WithKeyTable(types.ParamKeyTable())
	}

	jwkSubspace.SetParamSet(ctx, &defaultParams)
	return nil
}
