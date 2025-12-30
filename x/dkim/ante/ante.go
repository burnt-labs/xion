package ante

import (
	"cosmossdk.io/errors"

	dkimkeeper "github.com/burnt-labs/xion/x/dkim/keeper"
	types "github.com/burnt-labs/xion/x/dkim/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DKIMDecorator is a decorator that checks if the transaction is a valid DKIM transaction.
type DKIMDecorator struct {
	Keeper *dkimkeeper.Keeper
}

func NewDKIMDecorator(keeper *dkimkeeper.Keeper) DKIMDecorator {
	return DKIMDecorator{
		Keeper: keeper,
	}
}

func (d DKIMDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	// do not validate on recheck
	if ctx.IsReCheckTx() {
		return next(ctx, tx, simulate)
	}
	params, err := d.Keeper.GetParams(ctx)
	if err != nil {
		return ctx, err
	}
	for _, msg := range tx.GetMsgs() {
		switch msg := msg.(type) {
		case *types.MsgAddDkimPubKeys:
			if err := dkimkeeper.ValidateDkimPubKeys(msg.DkimPubkeys, params); err != nil {
				return ctx, errors.Wrap(err, "invalid dkim public keys")
			}
			return next(ctx, tx, simulate)
		}
	}
	return next(ctx, tx, simulate)
}
