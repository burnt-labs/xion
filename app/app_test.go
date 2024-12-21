package app

import (
	"os"
	"testing"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/stretchr/testify/require"

	"github.com/cometbft/cometbft/abci/types"

	dbm "github.com/cosmos/cosmos-db"

	"cosmossdk.io/log"

	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var emptyWasmOpts []wasmkeeper.Option

func TestWasmdExport(t *testing.T) {
	db := dbm.NewMemDB()
	gapp := NewWasmAppWithCustomOptions(t, false, SetupOptions{
		Logger:  log.NewLogger(os.Stdout),
		DB:      db,
		AppOpts: simtestutil.NewAppOptionsWithFlagHome(t.TempDir()),
	})
	_, err := gapp.FinalizeBlock(&types.RequestFinalizeBlock{
		Height: 1,
	})
	require.NoError(t, err, "FinalizeBlock should not have an error")
	_, err = gapp.Commit()
	require.NoError(t, err, "Commit should not have an error")

	// Making a new app object with the db, so that initchain hasn't been called
	newGapp := NewWasmApp(
		log.NewLogger(os.Stdout),
		db,
		nil,
		true,
		simtestutil.NewAppOptionsWithFlagHome(t.TempDir()),
		emptyWasmOpts,
	)
	_, err = newGapp.ExportAppStateAndValidators(false, []string{}, nil)
	require.NoError(t, err, "ExportAppStateAndValidators should not have an error")
}

// ensure that blocked addresses are properly set in bank keeper
func TestBlockedAddrs(t *testing.T) {
	gapp := Setup(t)

	for acc := range BlockedAddresses() {
		t.Run(acc, func(t *testing.T) {
			var addr sdk.AccAddress
			if modAddr, err := sdk.AccAddressFromBech32(acc); err == nil {
				addr = modAddr
			} else {
				addr = gapp.AccountKeeper.GetModuleAddress(acc)
			}
			require.True(t, gapp.BankKeeper.BlockedAddr(addr), "ensure that blocked addresses are properly set in bank keeper")
		})
	}
}

func TestGetMaccPerms(t *testing.T) {
	dup := GetMaccPerms()
	require.Equal(t, maccPerms, dup, "duplicated module account permissions differed from actual module account permissions")
}
