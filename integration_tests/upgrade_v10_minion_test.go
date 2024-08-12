package integration_tests

import (
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	dbm "github.com/cosmos/cosmos-db"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	"github.com/stretchr/testify/require"
	"os"
	"testing"

	"cosmossdk.io/log"
	"github.com/burnt-labs/xion/app"
)

var emptyWasmOpts []wasmkeeper.Option

func TestUpgradeV10Minion(t *testing.T) {
	// Create a new instance of the app
	db := dbm.NewMemDB()
	simApp := app.NewWasmApp(
		log.NewLogger(os.Stdout),
		db,
		nil,
		true,
		simtestutil.NewAppOptionsWithFlagHome(t.TempDir()),
		emptyWasmOpts,
	)

	require.IsType(t, &app.WasmApp{}, simApp)
}
