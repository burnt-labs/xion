package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	dbm "github.com/cosmos/cosmos-db"

	"cosmossdk.io/log"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// panicMessage runs fn and returns the message of the panic it raised, failing
// the test when fn returns normally.
func panicMessage(t *testing.T, fn func()) string {
	t.Helper()
	var message string
	func() {
		defer func() {
			recovered := recover()
			require.NotNil(t, recovered, "expected a panic")
			message = fmt.Sprint(recovered)
		}()
		fn()
	}()
	return message
}

func writeUpgradeInfo(t *testing.T, home, contents string) {
	t.Helper()
	dataDir := filepath.Join(home, "data")
	require.NoError(t, os.MkdirAll(dataDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dataDir, upgradetypes.UpgradeInfoFilename), []byte(contents), 0o600))
}

// The upgrade info file is read while the app is being built, which is the
// only time a store loader can still be installed.
func TestRegisterUpgradeHandlersUpgradeInfoOnDisk(t *testing.T) {
	pendingUpgrade := fmt.Sprintf(`{"name":%q,"height":100}`, UpgradeName)

	t.Run("unreadable upgrade info aborts startup", func(t *testing.T) {
		home := t.TempDir()
		writeUpgradeInfo(t, home, "{not json")

		message := panicMessage(t, func() {
			NewWasmApp(log.NewNopLogger(), dbm.NewMemDB(), nil, true, simtestutil.AppOptionsMap{flags.FlagHome: home}, nil)
		})
		require.Contains(t, message, "failed to read upgrade info from disk")
	})

	t.Run("pending upgrade installs the store loader", func(t *testing.T) {
		home := t.TempDir()
		writeUpgradeInfo(t, home, pendingUpgrade)

		gapp := NewWasmApp(log.NewNopLogger(), dbm.NewMemDB(), nil, true, simtestutil.AppOptionsMap{flags.FlagHome: home}, nil)
		require.True(t, gapp.UpgradeKeeper.HasHandler(UpgradeName))
		require.False(t, gapp.UpgradeKeeper.IsSkipHeight(100))
	})

	t.Run("skipped height leaves the default store loader", func(t *testing.T) {
		home := t.TempDir()
		writeUpgradeInfo(t, home, pendingUpgrade)
		opts := simtestutil.AppOptionsMap{
			flags.FlagHome:                home,
			server.FlagUnsafeSkipUpgrades: []int{100},
		}

		gapp := NewWasmApp(log.NewNopLogger(), dbm.NewMemDB(), nil, true, opts, nil)
		require.True(t, gapp.UpgradeKeeper.HasHandler(UpgradeName))
		require.True(t, gapp.UpgradeKeeper.IsSkipHeight(100))
	})
}

func TestNextUpgradeHandlerMigrationFailure(t *testing.T) {
	gapp := Setup(t)
	ctx := gapp.NewContext(false)

	// Module versions start at 1; a zero version has no migration to run and
	// the handler must not pretend the upgrade succeeded.
	fromVM := gapp.ModuleManager.GetVersionMap()
	fromVM[banktypes.ModuleName] = 0

	message := panicMessage(t, func() {
		_, _ = gapp.NextUpgradeHandler(ctx, upgradetypes.Plan{Name: UpgradeName, Height: 100}, fromVM)
	})
	require.Contains(t, message, "failed to run migrations")
}

type unavailableParams struct{}

func (unavailableParams) Has(context.Context) (bool, error) {
	return true, errors.New("params store unavailable")
}

func TestIsModuleInitializedTreatsLookupFailureAsUninitialized(t *testing.T) {
	require.False(t, new(WasmApp).isModuleInitialized(context.Background(), unavailableParams{}))
}
