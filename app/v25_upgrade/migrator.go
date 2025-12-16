package v25_upgrade

import (
	"fmt"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/gogo/protobuf/proto"

	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// MigrateContracts fixes corrupted contracts that cannot unmarshal
// This is called from the v25 upgrade handler
func MigrateContracts(ctx sdk.Context, storeKey storetypes.StoreKey) error {
	logger := ctx.Logger()
	logger.Info("v25: starting contract migration")

	store := ctx.KVStore(storeKey)

	var totalContracts int
	var fixedContracts int
	var skippedContracts int
	var failedContracts int

	prefix := []byte{ContractKeyPrefix}
	iterator := storetypes.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		totalContracts++

		key := iterator.Key()
		data := iterator.Value()

		// Get address for logging
		addressBytes := key[len(prefix):]
		address := sdk.AccAddress(addressBytes).String()

		// Try to unmarshal - if it succeeds, skip it (already working)
		var contractInfo wasmtypes.ContractInfo
		unmarshalErr := proto.Unmarshal(data, &contractInfo)

		if unmarshalErr == nil {
			// Contract is already working - skip it
			skippedContracts++
			if totalContracts%1000 == 0 {
				logger.Info("v25 migration progress",
					"processed", totalContracts,
					"fixed", fixedContracts,
					"skipped", skippedContracts)
			}
			continue
		}

		// Contract fails to unmarshal - needs fixing
		// Use the FixContract function from fixer.go
		result := FixContract(address, data)

		if !result.FixSucceeded || !result.UnmarshalAfter {
			logger.Error("v25: failed to fix contract",
				"address", address,
				"original_state", result.OriginalState,
				"error", result.Error)
			failedContracts++
			continue
		}

		// Write the fixed data
		store.Set(key, result.FixedData)
		fixedContracts++

		if fixedContracts%100 == 0 {
			logger.Info("v25 migration progress",
				"processed", totalContracts,
				"fixed", fixedContracts,
				"skipped", skippedContracts)
		}
	}

	logger.Info("v25 migration complete",
		"total", totalContracts,
		"fixed", fixedContracts,
		"skipped", skippedContracts,
		"failed", failedContracts)

	if failedContracts > 0 {
		return fmt.Errorf("failed to fix %d contracts", failedContracts)
	}

	return nil
}
