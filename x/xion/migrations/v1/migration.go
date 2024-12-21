package v1

import (
	"fmt"
	"sync"
	"sync/atomic"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/xion/types"
)

func MigrateStore(
	ctx sdk.Context,
	wasmOpsKeeper wasmtypes.ContractOpsKeeper,
	wasmViewKeeper wasmtypes.ViewKeeper,
	aaKeeper types.AbstractAccountKeeper,
	newCodeID uint64,
) error {
	// get the previous account code ID
	aaParams, err := aaKeeper.GetParams(ctx)
	if err != nil {
		return err
	}
	if len(aaParams.AllowedCodeIDs) != 1 {
		return fmt.Errorf("expected one allowed code id for abstract account, got: %v", aaParams.AllowedCodeIDs)
	}

	originalCodeID := aaParams.AllowedCodeIDs[0]

	// the account contract should always be pinned
	err = wasmOpsKeeper.PinCode(ctx, newCodeID)
	if err != nil {
		return err
	}

	// setup concurrency control
	var wg sync.WaitGroup
	errors := make(chan error, 1)
	defer close(errors)
	semaphore := make(chan struct{}, 10) // Limits the number of concurrent migrations
	defer close(semaphore)

	// counter for migrated contracts
	var migratedCount uint64

	// iterate through all existing accounts at this code ID, and migrate them
	wasmViewKeeper.IterateContractsByCode(ctx, originalCodeID, func(instance sdk.AccAddress) bool {
		semaphore <- struct{}{} // acquire semaphore
		wg.Add(1)

		go func(instance sdk.AccAddress) {
			defer wg.Done()
			defer func() { <-semaphore }() // release semaphore

			ctx.Logger().Info("Migrating contract", "instance", instance.String(), "newCodeID", newCodeID)
			_, err = wasmOpsKeeper.Migrate(ctx, instance, instance, newCodeID, []byte("{}"))
			if err != nil {
				ctx.Logger().Error("Error migrating contract", "contract", instance.String(), "error", err.Error())
				errors <- err
			} else {
				// safely increment the counter
				atomic.AddUint64(&migratedCount, 1)
			}
		}(instance)

		return false
	})

	wg.Wait()

	select {
	case err = <-errors:
		return err
	default:
		// No errors, proceed
	}

	ctx.Logger().Info(fmt.Sprintf("Total contracts migrated: %d", migratedCount))

	// as the previous contract is no longer the main account target, it doesn't
	// need to be pinned
	err = wasmOpsKeeper.UnpinCode(ctx, originalCodeID)
	if err != nil {
		return err
	}

	// adjust the aa registration endpoint to point at the new code ID
	aaParams.AllowedCodeIDs = []uint64{newCodeID}
	err = aaKeeper.SetParams(ctx, aaParams)
	if err != nil {
		return err
	}

	return nil
}
