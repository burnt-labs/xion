package v3

import (
	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/zk/types"
)

// MigrateStore performs in-place migrations for the zk module from v2 to v3.
//
// This migration backfills newly-added Groth16 and UltraHonk proof/public input size params with defaults
// for chains that previously stored params without these fields.
func MigrateStore(
	ctx sdk.Context,
	paramsItem collections.Item[types.Params],
) error {
	ctx.Logger().Info("Running zk module migration from v2 to v3")

	p, err := paramsItem.Get(ctx)
	if err != nil {
		if errorsmod.IsOf(err, collections.ErrNotFound) {
			// No params were persisted yet; let GetParams fall back to DefaultParams.
			ctx.Logger().Info("zk params not found; skipping params migration")
			return nil
		}
		return err
	}

	updated := false
	if p.MaxGroth16ProofSizeBytes == 0 {
		p.MaxGroth16ProofSizeBytes = types.DefaultMaxGroth16ProofSizeBytes
		updated = true
	}
	if p.MaxGroth16PublicInputSizeBytes == 0 {
		p.MaxGroth16PublicInputSizeBytes = types.DefaultMaxGroth16PublicInputSizeBytes
		updated = true
	}

	if p.MaxUltraHonkProofSizeBytes == 0 {
		p.MaxUltraHonkProofSizeBytes = types.DefaultMaxUltraHonkProofSizeBytes
		updated = true
	}
	if p.MaxUltraHonkPublicInputSizeBytes == 0 {
		p.MaxUltraHonkPublicInputSizeBytes = types.DefaultMaxUltraHonkPublicInputSizeBytes
		updated = true
	}

	if updated {
		if err := paramsItem.Set(ctx, p); err != nil {
			return err
		}
	}

	ctx.Logger().Info("ZK module migration from v2 to v3 completed successfully")
	return nil
}
