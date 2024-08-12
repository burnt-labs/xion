package integration_tests

import (
	"os"
	"testing"

	dbm "github.com/cosmos/cosmos-db"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"
	"github.com/burnt-labs/xion/app"
)

func TestUpgradeV10Minion(t *testing.T) {
	// Create a new instance of the app
	db := dbm.NewMemDB()
	simApp := app.NewWasmAppWithCustomOptions(t, false, app.SetupOptions{
		Logger:  log.NewLogger(os.Stdout),
		DB:      db,
		AppOpts: simtestutil.NewAppOptionsWithFlagHome(t.TempDir()),
	})

	require.IsType(t, &app.WasmApp{}, simApp)
}
