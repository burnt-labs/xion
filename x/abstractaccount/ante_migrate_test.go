package abstractaccount_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	simapptesting "github.com/burnt-labs/xion/x/abstractaccount/simapp/testing"
	"github.com/burnt-labs/xion/x/abstractaccount"
	"github.com/burnt-labs/xion/x/abstractaccount/types"
)

func TestMigrateValidationDecorator(t *testing.T) {
	app := simapptesting.MakeSimpleMockApp()
	ctx := app.NewContext(false)

	// Create an AbstractAccount with unique account number
	absAccAddr := simapptesting.MakeRandomAddress()
	absAcc := types.NewAbstractAccount(absAccAddr.String(), 100, 0)
	app.AccountKeeper.SetAccount(ctx, absAcc)

	// Create a regular account (non-AA) - use NewAccountWithAddress to let it assign account num
	regularAccAddr := simapptesting.MakeRandomAddress()
	unknownAccAddr := simapptesting.MakeRandomAddress()
	regularAcc := app.AccountKeeper.NewAccountWithAddress(ctx, regularAccAddr)
	app.AccountKeeper.SetAccount(ctx, regularAcc)

	// Set params to only allow code ID 1 and 2
	params, err := types.NewParams(false, []uint64{1, 2}, 1000000, 1000000)
	require.NoError(t, err)
	err = app.AbstractAccountKeeper.SetParams(ctx, params)
	require.NoError(t, err)

	decorator := abstractaccount.NewMigrateValidationDecorator(
		app.AbstractAccountKeeper,
		app.AccountKeeper,
	)

	for _, tc := range []struct {
		desc           string
		contractAddr   string
		codeID         uint64
		expOk          bool
		expErrContains string
	}{
		{
			desc:         "allowed code ID for AbstractAccount",
			contractAddr: absAccAddr.String(),
			codeID:       1,
			expOk:        true,
		},
		{
			desc:         "allowed code ID 2 for AbstractAccount",
			contractAddr: absAccAddr.String(),
			codeID:       2,
			expOk:        true,
		},
		{
			desc:           "disallowed code ID for AbstractAccount",
			contractAddr:   absAccAddr.String(),
			codeID:         999,
			expOk:          false,
			expErrContains: types.ErrNotAllowedCodeID.Error(),
		},
		{
			desc:         "any code ID for regular account (not an AA)",
			contractAddr: regularAccAddr.String(),
			codeID:       999,
			expOk:        true,
		},
		{
			desc:         "unknown account address",
			contractAddr: unknownAccAddr.String(),
			codeID:       999,
			expOk:        true,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			msg := &wasmtypes.MsgMigrateContract{
				Sender:   absAccAddr.String(),
				Contract: tc.contractAddr,
				CodeID:   tc.codeID,
				Msg:      []byte("{}"),
			}

			txBuilder := app.TxConfig().NewTxBuilder()
			require.NoError(t, txBuilder.SetMsgs(msg))
			tx := txBuilder.GetTx()

			_, err := decorator.AnteHandle(ctx, tx, false, anteTerminator)

			if tc.expOk {
				require.NoError(t, err, tc.desc)
			} else {
				require.Error(t, err, tc.desc)
				if tc.expErrContains != "" {
					require.Contains(t, err.Error(), tc.expErrContains, tc.desc)
				}
			}
		})
	}
}

func TestMigrateValidationDecorator_AllowAllCodeIDs(t *testing.T) {
	app := simapptesting.MakeSimpleMockApp()
	ctx := app.NewContext(false)

	// Create an AbstractAccount with unique account number
	absAccAddr := simapptesting.MakeRandomAddress()
	absAcc := types.NewAbstractAccount(absAccAddr.String(), 200, 0)
	app.AccountKeeper.SetAccount(ctx, absAcc)

	// Set params to allow all code IDs
	params, err := types.NewParams(true, []uint64{}, 1000000, 1000000)
	require.NoError(t, err)
	err = app.AbstractAccountKeeper.SetParams(ctx, params)
	require.NoError(t, err)

	decorator := abstractaccount.NewMigrateValidationDecorator(
		app.AbstractAccountKeeper,
		app.AccountKeeper,
	)

	msg := &wasmtypes.MsgMigrateContract{
		Sender:   absAccAddr.String(),
		Contract: absAccAddr.String(),
		CodeID:   999, // Would be rejected if AllowAllCodeIDs was false
		Msg:      []byte("{}"),
	}

	txBuilder := app.TxConfig().NewTxBuilder()
	require.NoError(t, txBuilder.SetMsgs(msg))
	tx := txBuilder.GetTx()

	_, err = decorator.AnteHandle(ctx, tx, false, anteTerminator)
	require.NoError(t, err)
}

func TestMigrateValidationDecorator_NonMigrateMsg(t *testing.T) {
	app := simapptesting.MakeSimpleMockApp()
	ctx := app.NewContext(false)

	// Set restrictive params
	params, err := types.NewParams(false, []uint64{1}, 1000000, 1000000)
	require.NoError(t, err)
	err = app.AbstractAccountKeeper.SetParams(ctx, params)
	require.NoError(t, err)

	decorator := abstractaccount.NewMigrateValidationDecorator(
		app.AbstractAccountKeeper,
		app.AccountKeeper,
	)

	// A non-migrate message should pass through
	msg := &wasmtypes.MsgExecuteContract{
		Sender:   simapptesting.MakeRandomAddress().String(),
		Contract: simapptesting.MakeRandomAddress().String(),
		Msg:      []byte("{}"),
	}

	txBuilder := app.TxConfig().NewTxBuilder()
	require.NoError(t, txBuilder.SetMsgs(msg))
	tx := txBuilder.GetTx()

	_, err = decorator.AnteHandle(ctx, tx, false, anteTerminator)
	require.NoError(t, err)
}
