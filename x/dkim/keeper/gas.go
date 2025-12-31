package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/dkim/types"
)

// ConsumeDkimPubKeyGas charges gas for processing a DKIM public key after ensuring it fits module limits.
func ConsumeDkimPubKeyGas(ctx sdk.Context, params types.Params, encodedPubKey string) error {
	pubKeyBytes, err := types.DecodePubKeyWithLimit(encodedPubKey, params.MaxPubkeySizeBytes)
	if err != nil {
		return err
	}

	gasCost, err := params.GasCostForSize(uint64(len(pubKeyBytes)))
	if err != nil {
		return err
	}

	ctx.GasMeter().ConsumeGas(gasCost, "dkim pubkey upload")
	return nil
}
