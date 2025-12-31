package ante

import (
	"cosmossdk.io/errors"

	zkkeeper "github.com/burnt-labs/xion/x/zk/keeper"
	"github.com/burnt-labs/xion/x/zk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ZKDecorator enforces vkey size limits and gas costs before messages enter the mempool.
type ZKDecorator struct {
	Keeper *zkkeeper.Keeper
}

func NewZKDecorator(keeper *zkkeeper.Keeper) ZKDecorator {
	return ZKDecorator{
		Keeper: keeper,
	}
}

func (d ZKDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if ctx.IsReCheckTx() {
		return next(ctx, tx, simulate)
	}

	params, err := d.Keeper.GetParams(ctx)
	if err != nil {
		return ctx, err
	}

	for _, msg := range tx.GetMsgs() {
		switch msg := msg.(type) {
		case *types.MsgAddVKey:
			decoded, err := types.DecodeAndValidateVKeyBytes(msg.VkeyBytes, params.MaxVkeySizeBytes)
			if err != nil {
				return ctx, errors.Wrap(err, "invalid vkey_bytes")
			}
			msg.VkeyBytes = decoded

			if err := zkkeeper.ConsumeVKeyGas(ctx, params, len(msg.VkeyBytes)); err != nil {
				return ctx, errors.Wrap(err, "invalid vkey_bytes")
			}
		case *types.MsgUpdateVKey:
			decoded, err := types.DecodeAndValidateVKeyBytes(msg.VkeyBytes, params.MaxVkeySizeBytes)
			if err != nil {
				return ctx, errors.Wrap(err, "invalid vkey_bytes")
			}
			msg.VkeyBytes = decoded

			if err := zkkeeper.ConsumeVKeyGas(ctx, params, len(msg.VkeyBytes)); err != nil {
				return ctx, errors.Wrap(err, "invalid vkey_bytes")
			}
		}
	}

	return next(ctx, tx, simulate)
}
