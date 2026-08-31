package abstractaccount_test

import (
	"testing"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"

	xionapp "github.com/burnt-labs/xion/app"
	"github.com/burnt-labs/xion/x/abstractaccount"
	"github.com/burnt-labs/xion/x/abstractaccount/types"
)

func TestMigrateValidationDecorator(t *testing.T) {
	app := xionapp.Setup(t)
	ctx := app.NewContext(false)

	// Create an AbstractAccount with unique account number
	absAccAddr := xionapp.RandomAccAddress()
	absAcc := types.NewAbstractAccount(absAccAddr.String(), app.AccountKeeper.NextAccountNumber(ctx), 0)
	app.AccountKeeper.SetAccount(ctx, absAcc)

	// Create a regular account (non-AA) - use NewAccountWithAddress to let it assign account num
	regularAccAddr := xionapp.RandomAccAddress()
	unknownAccAddr := xionapp.RandomAccAddress()
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
	app := xionapp.Setup(t)
	ctx := app.NewContext(false)

	// Create an AbstractAccount with unique account number
	absAccAddr := xionapp.RandomAccAddress()
	absAcc := types.NewAbstractAccount(absAccAddr.String(), app.AccountKeeper.NextAccountNumber(ctx), 0)
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
	app := xionapp.Setup(t)
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
		Sender:   xionapp.RandomAccAddress().String(),
		Contract: xionapp.RandomAccAddress().String(),
		Msg:      []byte("{}"),
	}

	txBuilder := app.TxConfig().NewTxBuilder()
	require.NoError(t, txBuilder.SetMsgs(msg))
	tx := txBuilder.GetTx()

	_, err = decorator.AnteHandle(ctx, tx, false, anteTerminator)
	require.NoError(t, err)
}

// TestMigrateValidationDecorator_AuthzWrapped checks that a migration wrapped in
// an authz MsgExec is held to the same AllowedCodeIDs rule as a top-level one.
//
// authz dispatches the contents of MsgExec through the message router with the
// granter as signer, so inspecting only tx.GetMsgs() would let a grantee migrate
// an abstract account to an excluded code ID.
func TestMigrateValidationDecorator_AuthzWrapped(t *testing.T) {
	app := xionapp.Setup(t)
	ctx := app.NewContext(false)

	absAccAddr := xionapp.RandomAccAddress()
	absAcc := types.NewAbstractAccount(absAccAddr.String(), app.AccountKeeper.NextAccountNumber(ctx), 0)
	app.AccountKeeper.SetAccount(ctx, absAcc)

	grantee := xionapp.RandomAccAddress()

	params, err := types.NewParams(false, []uint64{1, 2}, 1000000, 1000000)
	require.NoError(t, err)
	require.NoError(t, app.AbstractAccountKeeper.SetParams(ctx, params))

	decorator := abstractaccount.NewMigrateValidationDecorator(
		app.AbstractAccountKeeper,
		app.AccountKeeper,
	)

	migrateTo := func(codeID uint64) *wasmtypes.MsgMigrateContract {
		return &wasmtypes.MsgMigrateContract{
			Sender:   absAccAddr.String(),
			Contract: absAccAddr.String(),
			CodeID:   codeID,
			Msg:      []byte("{}"),
		}
	}

	for _, tc := range []struct {
		desc   string
		build  func() sdk.Msg
		expErr bool
	}{
		{
			desc: "authz-wrapped migration to a disallowed code ID is rejected",
			build: func() sdk.Msg {
				exec := authz.NewMsgExec(grantee, []sdk.Msg{migrateTo(999)})
				return &exec
			},
			expErr: true,
		},
		{
			desc: "authz-wrapped migration to an allowed code ID passes",
			build: func() sdk.Msg {
				exec := authz.NewMsgExec(grantee, []sdk.Msg{migrateTo(1)})
				return &exec
			},
			expErr: false,
		},
		{
			desc: "doubly nested authz-wrapped migration is still rejected",
			build: func() sdk.Msg {
				inner := authz.NewMsgExec(grantee, []sdk.Msg{migrateTo(999)})
				outer := authz.NewMsgExec(grantee, []sdk.Msg{&inner})
				return &outer
			},
			expErr: true,
		},
		{
			desc: "authz-wrapped unrelated message passes",
			build: func() sdk.Msg {
				exec := authz.NewMsgExec(grantee, []sdk.Msg{&wasmtypes.MsgExecuteContract{
					Sender:   absAccAddr.String(),
					Contract: absAccAddr.String(),
					Msg:      []byte("{}"),
				}})
				return &exec
			},
			expErr: false,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			txBuilder := app.TxConfig().NewTxBuilder()
			require.NoError(t, txBuilder.SetMsgs(tc.build()))

			_, err := decorator.AnteHandle(ctx, txBuilder.GetTx(), false, anteTerminator)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), types.ErrNotAllowedCodeID.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
