package v1

import (
	"fmt"
	"github.com/cometbft/cometbft/libs/log"
	"os"
	"sync"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/burnt-labs/xion/x/xion/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func MigrateStore(ctx sdk.Context, wasmOpsKeeper wasmtypes.ContractOpsKeeper, wasmViewKeeper wasmtypes.ViewKeeper, aaKeeper types.AbstractAccountKeeper) error {
	const NewCodeId = 327 // todo: set

	// get the previous account code ID
	aaParams, err := aaKeeper.GetParams(ctx)
	if err != nil {
		return err
	}
	if len(aaParams.AllowedCodeIDs) != 1 {
		return fmt.Errorf("expected one allowed code id for abstract account, got: %v", aaParams.AllowedCodeIDs)
	}

	originalCodeId := aaParams.AllowedCodeIDs[0]

	// the account contract should always be pinned
	err = wasmOpsKeeper.PinCode(ctx, NewCodeId)
	if err != nil {
		return err
	}

	// setup concurrency control
	var wg sync.WaitGroup
	errors := make(chan error, 1)
	defer close(errors)
	semaphore := make(chan struct{}, 10) // Limits the number of concurrent migrations
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))

	// iterate through all existing accounts at this code ID, and migrate them
	wasmViewKeeper.IterateContractsByCode(ctx, originalCodeId, func(instance sdk.AccAddress) bool {
		semaphore <- struct{}{} // acquire semaphore
		wg.Add(1)

		go func(instance sdk.AccAddress) {
			defer wg.Done()
			defer func() { <-semaphore }() // release semaphore

			logger.Info("Migrating contract", "contract", instance.String())
			_, err := wasmOpsKeeper.Migrate(ctx, instance, instance, NewCodeId, []byte("{}"))
			if err != nil {
				logger.Error("Error migrating contract", "contract", instance.String(), "error", err.Error())
				errors <- err
			}
		}(instance)

		return false
	})

	wg.Wait()

	select {
	case err := <-errors:
		return err
	default:
		// No errors, proceed
	}

	// as the previous contract is no longer the main account target, it doesn't
	// need to be pinned
	err = wasmOpsKeeper.UnpinCode(ctx, originalCodeId)
	if err != nil {
		return err
	}

	// adjust the aa registration endpoint to point at the new code ID
	aaParams.AllowedCodeIDs = []uint64{NewCodeId}
	err = aaKeeper.SetParams(ctx, aaParams)
	if err != nil {
		return err
	}

	return nil
}
