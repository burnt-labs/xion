package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/zk/types"
)

// ConsumeVKeyGas charges gas for a vkey payload after ensuring it fits module limits.
func ConsumeVKeyGas(ctx sdk.Context, params types.Params, size int) error {
	gasCost, err := params.GasCostForSize(uint64(size))
	if err != nil {
		return err
	}

	ctx.GasMeter().ConsumeGas(gasCost, "zk vkey upload")
	return nil
}
