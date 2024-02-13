package v1

import (
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/burnt-labs/xion/x/xion/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func MigrateStore(ctx sdk.Context, wasmOpsKeeper wasmtypes.ContractOpsKeeper, wasmViewKeeper wasmtypes.ViewKeeper, aaKeeper types.AbstractAccountKeeper) error {
	const OriginalCodeId = 21
	const NewCodeId = -1 // todo: set

	// the account contract should always be pinned
	err := wasmOpsKeeper.PinCode(ctx, NewCodeId)
	if err != nil {
		return err
	}

	// iterate through all existing accounts at this code ID, and migrate them
	wasmViewKeeper.IterateContractsByCode(ctx, OriginalCodeId, func(instance sdk.AccAddress) bool {
		_, err = wasmOpsKeeper.Migrate(ctx, instance, instance, NewCodeId, nil)

		// if there is an error, abort iteration and report it
		return err == nil
	})
	if err != nil {
		return err
	}

	// as the previous contract is no longer the main account target, it doesn't
	// need to be pinned
	err = wasmOpsKeeper.UnpinCode(ctx, OriginalCodeId)
	if err != nil {
		return err
	}

	// adjust the aa registration endpoint to point at the new code ID
	aaParams, err := aaKeeper.GetParams(ctx)
	if err != nil {
		return err
	}
	aaParams.AllowedCodeIDs = []uint64{NewCodeId}
	err = aaKeeper.SetParams(ctx, aaParams)
	if err != nil {
		return err
	}

	return nil
}
