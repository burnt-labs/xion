package abstractaccount

import (
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"

	"github.com/burnt-labs/xion/x/abstractaccount/keeper"
	"github.com/burnt-labs/xion/x/abstractaccount/types"
)

var _ sdk.AnteDecorator = &MigrateValidationDecorator{}

// MigrateValidationDecorator validates that MsgMigrateContract for AbstractAccount
// contracts only migrates to code IDs in the AllowedCodeIDs list.
//
// This prevents attackers from migrating AA contracts to malicious code if
// code upload permissions are ever relaxed from "Nobody" to a more permissive
// setting.
type MigrateValidationDecorator struct {
	aak keeper.Keeper
	ak  authante.AccountKeeper
}

func NewMigrateValidationDecorator(aak keeper.Keeper, ak authante.AccountKeeper) MigrateValidationDecorator {
	return MigrateValidationDecorator{aak: aak, ak: ak}
}

func (d MigrateValidationDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	var params *types.Params

	for _, msg := range tx.GetMsgs() {
		migrateMsg, ok := msg.(*wasmtypes.MsgMigrateContract)
		if !ok {
			continue
		}

		// Check if contract is an AbstractAccount
		contractAddr, err := sdk.AccAddressFromBech32(migrateMsg.Contract)
		if err != nil {
			return ctx, err
		}

		acc := d.ak.GetAccount(ctx, contractAddr)
		if acc == nil {
			continue
		}

		_, isAbstractAccount := acc.(*types.AbstractAccount)
		if !isAbstractAccount {
			continue
		}

		// Validate new code ID against AllowedCodeIDs
		if params == nil {
			params, err = d.aak.GetParams(ctx)
			if err != nil {
				return ctx, err
			}
		}

		if !params.IsAllowed(migrateMsg.CodeID) {
			return ctx, types.ErrNotAllowedCodeID.Wrapf(
				"cannot migrate AbstractAccount to code ID %d",
				migrateMsg.CodeID,
			)
		}
	}

	return next(ctx, tx, simulate)
}
