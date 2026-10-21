package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	abci "github.com/cometbft/cometbft/abci/types"

	dbm "github.com/cosmos/cosmos-db"

	"cosmossdk.io/log/v2"

	bam "github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/store/v2/snapshots"
	snapshottypes "github.com/cosmos/cosmos-sdk/store/v2/snapshots/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
)

func TestNewWasmAppRejectsInvalidWasmConfig(t *testing.T) {
	opts := simtestutil.AppOptionsMap{
		flags.FlagHome:           t.TempDir(),
		"wasm.memory_cache_size": "plenty",
	}

	message := panicMessage(t, func() {
		NewWasmApp(log.NewNopLogger(), dbm.NewMemDB(), nil, true, opts, nil)
	})
	require.Contains(t, message, "error while reading wasm config")
}

func TestNewWasmAppRejectsUnusableHome(t *testing.T) {
	// A regular file where the home directory should be: the wasm cache
	// cannot be created underneath it.
	home := filepath.Join(t.TempDir(), "home")
	require.NoError(t, os.WriteFile(home, []byte("not a directory"), 0o600))
	opts := simtestutil.AppOptionsMap{flags.FlagHome: home}

	require.Panics(t, func() {
		NewWasmApp(log.NewNopLogger(), dbm.NewMemDB(), nil, true, opts, nil)
	})
}

func TestNewWasmAppWithIndexerEnabled(t *testing.T) {
	gapp := NewWasmAppWithCustomOptions(t, false, SetupOptions{
		Logger: log.NewNopLogger(),
		DB:     dbm.NewMemDB(),
		AppOpts: simtestutil.AppOptionsMap{
			flags.FlagHome:    t.TempDir(),
			"indexer.enabled": true,
		},
	})
	t.Cleanup(func() { require.NoError(t, gapp.Close()) })

	manager := gapp.StreamingManager()
	require.Len(t, manager.ABCIListeners, 1)
	require.Same(t, gapp.IndexerService(), manager.ABCIListeners[0])
	require.False(t, manager.StopNodeOnErr)

	// The listener receives the first block's change set without disturbing it.
	_, err := gapp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1})
	require.NoError(t, err)
	_, err = gapp.Commit()
	require.NoError(t, err)
}

func TestTokenFactoryGrantsNoSudoAdmin(t *testing.T) {
	gapp := Setup(t)
	ctx := gapp.NewContext(false)

	require.False(t, gapp.TokenFactoryKeeper.IsSudoAdminFunc(ctx, RandomAccAddress().String()))
}

func TestInitChainerRejectsMalformedGenesis(t *testing.T) {
	gapp := Setup(t)
	ctx := gapp.NewContext(false)

	require.Panics(t, func() {
		_, _ = gapp.InitChainer(ctx, &abci.RequestInitChain{AppStateBytes: []byte("{")})
	})
}

func TestCloseReportsSnapshotStoreFailure(t *testing.T) {
	snapshotDir := filepath.Join(t.TempDir(), "snapshots")
	snapshotDB, err := dbm.NewDB("metadata", dbm.GoLevelDBBackend, snapshotDir)
	require.NoError(t, err)
	snapshotStore, err := snapshots.NewStore(snapshotDB, snapshotDir)
	require.NoError(t, err)

	gapp := NewWasmApp(
		log.NewNopLogger(),
		dbm.NewMemDB(),
		nil,
		true,
		simtestutil.AppOptionsMap{flags.FlagHome: t.TempDir()},
		nil,
		bam.SetSnapshot(snapshotStore, snapshottypes.SnapshotOptions{KeepRecent: 2}),
	)

	// Closing the snapshot store out from under the app makes its own close
	// fail, and that failure must surface instead of being swallowed.
	require.NoError(t, snapshotDB.Close())
	require.Error(t, gapp.Close())
}
