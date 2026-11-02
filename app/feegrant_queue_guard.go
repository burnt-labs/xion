package app

import (
	"fmt"

	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
)

// feegrantEndBlockPruneLimit is the limit the feegrant EndBlocker passes to
// RemoveExpiredAllowances (cosmos-sdk v0.55.0 x/feegrant/module/abci.go).
const feegrantEndBlockPruneLimit = 200

// legacyFeeAllowanceQueueDue reports whether this block's feegrant EndBlocker
// would decode a queue value stored the pre-v32 way (empty). It visits the
// entries RemoveExpiredAllowances visits: in key order, expiry at or before the
// block time, at most feegrantEndBlockPruneLimit of them. It only reads, so in
// every block where it returns false the state is exactly what a binary
// without this check commits.
func (app *WasmApp) legacyFeeAllowanceQueueDue(ctx sdk.Context) (due bool, err error) {
	blockTime := ctx.BlockTime()
	keyCodec := app.FeeGrantKeeper.FeeAllowanceQueue.KeyCodec()
	store := ctx.KVStore(app.GetKey(feegrant.StoreKey))
	it := storetypes.KVStorePrefixIterator(store, feegrant.FeeAllowanceQueueKeyPrefix)
	defer func() {
		if cerr := it.Close(); err == nil {
			err = cerr
		}
	}()
	for visited := 0; it.Valid() && visited < feegrantEndBlockPruneLimit; it.Next() {
		_, key, err := keyCodec.Decode(it.Key()[len(feegrant.FeeAllowanceQueueKeyPrefix):])
		if err != nil {
			return false, fmt.Errorf("decode feegrant queue key: %w", err)
		}
		if key.K1().After(blockTime) {
			return false, nil
		}
		if len(it.Value()) == 0 {
			return true, nil
		}
		visited++
	}
	return false, nil
}

// fixDueLegacyFeeAllowanceQueue runs before the module EndBlockers. Chains that
// ran the v32 upgrade before its handler rewrote the queue (xion-mainnet-1)
// still hold legacy entries; the first block whose feegrant EndBlocker would
// fail on one rewrites all of them instead. A binary without this check cannot
// apply that block at all, so the rewrite diverges from no committed state.
func (app *WasmApp) fixDueLegacyFeeAllowanceQueue(ctx sdk.Context) error {
	due, err := app.legacyFeeAllowanceQueueDue(ctx)
	if err != nil || !due {
		return err
	}
	return app.migrateFeeAllowanceQueueValues(ctx)
}
