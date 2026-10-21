package abstractaccount_test

import (
	"strings"
	"testing"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/module"

	xionapp "github.com/burnt-labs/xion/app"
	"github.com/burnt-labs/xion/x/abstractaccount"
	"github.com/burnt-labs/xion/x/abstractaccount/types"
)

// freshConfigurator returns a configurator whose routers have nothing
// registered yet, so RegisterServices gets as far as the migrations.
func freshConfigurator(app *xionapp.WasmApp) module.Configurator {
	msgRouter := baseapp.NewMsgServiceRouter()
	msgRouter.SetInterfaceRegistry(app.InterfaceRegistry())
	queryRouter := baseapp.NewGRPCQueryRouter()
	queryRouter.SetInterfaceRegistry(app.InterfaceRegistry())

	return module.NewConfigurator(app.AppCodec(), msgRouter, queryRouter)
}

func noopMigration(sdk.Context) error { return nil }

// requirePanicsWithPrefix asserts that fn panics with a string starting with
// prefix; the SDK appends the source location of wrapped errors, which is
// not worth pinning.
func requirePanicsWithPrefix(t *testing.T, prefix string, fn func()) {
	t.Helper()
	defer func() {
		recovered := recover()
		require.NotNil(t, recovered, "expected a panic")
		msg, ok := recovered.(string)
		require.True(t, ok, "expected a string panic value, got %T", recovered)
		require.True(t, strings.HasPrefix(msg, prefix), "panic %q does not start with %q", msg, prefix)
	}()
	fn()
}

func TestAppModuleRegisterServicesFreshConfigurator(t *testing.T) {
	app := xionapp.Setup(t)
	appModule := abstractaccount.NewAppModule(app.AbstractAccountKeeper)

	require.NotPanics(t, func() {
		appModule.RegisterServices(freshConfigurator(app))
	})
}

func TestAppModuleRegisterServicesMigrationConflicts(t *testing.T) {
	app := xionapp.Setup(t)
	appModule := abstractaccount.NewAppModule(app.AbstractAccountKeeper)

	t.Run("version 1 already registered", func(t *testing.T) {
		cfg := freshConfigurator(app)
		require.NoError(t, cfg.RegisterMigration(types.ModuleName, 1, noopMigration))

		requirePanicsWithPrefix(t,
			"failed to migrate x/abstract-account from version 1 to 2: another migration for module abstractaccount and version 1 already exists",
			func() { appModule.RegisterServices(cfg) },
		)
	})

	t.Run("version 2 already registered", func(t *testing.T) {
		cfg := freshConfigurator(app)
		require.NoError(t, cfg.RegisterMigration(types.ModuleName, 2, noopMigration))

		requirePanicsWithPrefix(t,
			"failed to migrate x/abstract-account from version 2 to 3: another migration for module abstractaccount and version 2 already exists",
			func() { appModule.RegisterServices(cfg) },
		)
	})
}

// ---------------------------- Migrate validation --------------------------

func TestMigrateValidationDecorator_InvalidContractAddress(t *testing.T) {
	app := xionapp.Setup(t)
	ctx := app.NewContext(false)
	decorator := abstractaccount.NewMigrateValidationDecorator(app.AbstractAccountKeeper, app.AccountKeeper)

	txBuilder := app.TxConfig().NewTxBuilder()
	require.NoError(t, txBuilder.SetMsgs(&wasmtypes.MsgMigrateContract{
		Sender:   xionapp.RandomAccAddress().String(),
		Contract: "not-bech32",
		CodeID:   1,
		Msg:      []byte("{}"),
	}))

	_, err := decorator.AnteHandle(ctx, txBuilder.GetTx(), false, anteTerminator)
	require.ErrorContains(t, err, "decoding bech32")
}

func TestMigrateValidationDecorator_ParamsMissing(t *testing.T) {
	app := xionapp.Setup(t)
	ctx := app.NewContext(false)
	decorator := abstractaccount.NewMigrateValidationDecorator(app.AbstractAccountKeeper, app.AccountKeeper)

	absAccAddr := xionapp.RandomAccAddress()
	app.AccountKeeper.SetAccount(ctx, types.NewAbstractAccount(absAccAddr.String(), app.AccountKeeper.NextAccountNumber(ctx), 0))
	deleteModuleParams(app, ctx)

	txBuilder := app.TxConfig().NewTxBuilder()
	require.NoError(t, txBuilder.SetMsgs(&wasmtypes.MsgMigrateContract{
		Sender:   absAccAddr.String(),
		Contract: absAccAddr.String(),
		CodeID:   1,
		Msg:      []byte("{}"),
	}))

	_, err := decorator.AnteHandle(ctx, txBuilder.GetTx(), false, anteTerminator)
	require.ErrorIs(t, err, sdkerrors.ErrNotFound)
}
