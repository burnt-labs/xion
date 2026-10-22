package main

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
)

// newTestInPlaceTestnetCmd returns the command with a server context rooted at
// home, as the root command's pre-run handler would set it up.
func newTestInPlaceTestnetCmd(t *testing.T, home string, configure func(*server.Context)) *cobra.Command {
	t.Helper()
	serverCtx := server.NewDefaultContext()
	serverCtx.Config.SetRoot(home)
	if configure != nil {
		configure(serverCtx)
	}
	cmd := inPlaceTestnetCmd()
	cmd.SetContext(context.WithValue(context.Background(), server.ServerContextKey, serverCtx))
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	return cmd
}

// writeTestSourceHome writes a marker file and a genesis file for sourceChainID
// and returns a snapshot of every file in home.
func writeTestSourceHome(t *testing.T, home, sourceChainID string) map[string]string {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(home, "source-state"), []byte("unchanged"), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(home, "config"), 0o700))
	genesis := `{"genesis_time":"2024-01-01T00:00:00Z","chain_id":"` + sourceChainID + `","initial_height":1,"app_state":{"bank":{}}}`
	require.NoError(t, os.WriteFile(filepath.Join(home, "config", "genesis.json"), []byte(genesis), 0o600))
	return snapshotTestHome(t, home)
}

func snapshotTestHome(t *testing.T, home string) map[string]string {
	t.Helper()
	root, err := os.OpenRoot(home)
	require.NoError(t, err)
	defer func() { require.NoError(t, root.Close()) }()
	homeFS := root.FS()
	files := map[string]string{}
	require.NoError(t, fs.WalkDir(homeFS, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, err := fs.ReadFile(homeFS, path)
		if err != nil {
			return err
		}
		files[path] = string(data)
		return nil
	}))
	return files
}

func TestInPlaceTestnetDeclinedLeavesHomeUntouched(t *testing.T) {
	home := t.TempDir()
	before := writeTestSourceHome(t, home, "xion-devnet-1")
	cmd := newTestInPlaceTestnetCmd(t, home, nil)
	cmd.SetArgs([]string{"fork-test", "operator", "--home", home})
	cmd.SetIn(strings.NewReader("no\n"))
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)
	require.NoError(t, cmd.Execute())
	require.Contains(t, stderr.String(), "Converting home "+home+" from source chain xion-devnet-1 to new chain fork-test.")
	require.Contains(t, stderr.String(), "Conversion cancelled.")
	require.Equal(t, before, snapshotTestHome(t, home), "declining conversion must not initialize or open node state")
}

func TestInPlaceTestnetRejectsStandaloneBeforeMutation(t *testing.T) {
	home := t.TempDir()
	cmd := newTestInPlaceTestnetCmd(t, home, nil)
	cmd.SetArgs([]string{"fork-test", "operator", "--home", home, "--with-comet=false", "--skip-confirmation"})
	require.ErrorContains(t, cmd.Execute(), "requires in-process CometBFT")
	entries, err := os.ReadDir(home)
	require.NoError(t, err)
	require.Empty(t, entries, "rejected standalone conversion must not initialize or open node state")
}

func TestInPlaceTestnetRejectsPublicChainIDBeforeMutation(t *testing.T) {
	for _, chainID := range []string{"xion-mainnet-1", "xion-testnet-2", "xion-mainnet-2", "XION-Testnet-2"} {
		t.Run(chainID, func(t *testing.T) {
			home := t.TempDir()
			before := writeTestSourceHome(t, home, "xion-devnet-1")
			cmd := newTestInPlaceTestnetCmd(t, home, nil)
			cmd.SetArgs([]string{chainID, "operator", "--home", home})
			cmd.SetIn(strings.NewReader("y\n"))
			require.ErrorContains(t, cmd.Execute(), "reuses the public network name")
			require.Equal(t, before, snapshotTestHome(t, home), "rejected conversion must not change node state")
		})
	}
}

func TestInPlaceTestnetRequiresExplicitHome(t *testing.T) {
	home := t.TempDir()
	before := writeTestSourceHome(t, home, "xion-devnet-1")
	// The server context points at a home, as it would for the default node
	// home or one taken from the environment, but --home is not on the command line.
	cmd := newTestInPlaceTestnetCmd(t, home, nil)
	cmd.SetArgs([]string{"fork-test", "operator", "--skip-confirmation"})
	require.ErrorContains(t, cmd.Execute(), "requires an explicit --home")
	require.Equal(t, before, snapshotTestHome(t, home))
}

// The root pre-run applies XIOND_HOME and config values to unset flags with
// Flags().Set, which marks --home as changed. That must not count as an
// explicit --home.
func TestInPlaceTestnetRejectsHomeFromEnvironment(t *testing.T) {
	home := t.TempDir()
	before := writeTestSourceHome(t, home, "xion-devnet-1")
	root := &cobra.Command{
		Use: "xiond",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Flags().Set(flags.FlagHome, home)
		},
	}
	root.AddCommand(newTestInPlaceTestnetCmd(t, home, nil))
	serverCtx := server.NewDefaultContext()
	serverCtx.Config.SetRoot(home)
	root.SetArgs([]string{"in-place-testnet", "fork-test", "operator", "--skip-confirmation"})
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	err := root.ExecuteContext(context.WithValue(context.Background(), server.ServerContextKey, serverCtx))
	require.ErrorContains(t, err, "requires an explicit --home")
	require.Equal(t, before, snapshotTestHome(t, home))
}

// A remote-signer validator or a sentry keeps an unused stub key in config/,
// which passes the local key guards. Such a home must be refused before any of
// its state is opened, even with --skip-confirmation.
func TestInPlaceTestnetRejectsRemoteSignerHome(t *testing.T) {
	home := t.TempDir()
	before := writeTestSourceHome(t, home, "xion-devnet-1")
	cmd := newTestInPlaceTestnetCmd(t, home, func(serverCtx *server.Context) {
		serverCtx.Config.PrivValidatorListenAddr = "tcp://127.0.0.1:26659"
	})
	cmd.SetArgs([]string{"fork-test", "operator", "--home", home, "--skip-confirmation"})
	require.ErrorContains(t, cmd.Execute(), "uses a remote signer")
	require.Equal(t, before, snapshotTestHome(t, home))
}

func TestInPlaceTestnetRejectsMissingGenesis(t *testing.T) {
	home := t.TempDir()
	cmd := newTestInPlaceTestnetCmd(t, home, nil)
	cmd.SetArgs([]string{"fork-test", "operator", "--home", home, "--skip-confirmation"})
	require.ErrorContains(t, cmd.Execute(), "read source genesis")
	entries, err := os.ReadDir(home)
	require.NoError(t, err)
	require.Empty(t, entries)
}

func TestReadGenesisChainID(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) string {
		path := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
		return path
	}

	chainID, err := readGenesisChainID(write("after-app-state.json", `{"app_state":{"x":[1,{"chain_id":"nested"}]},"chain_id":"xion-testnet-2"}`))
	require.NoError(t, err)
	require.Equal(t, "xion-testnet-2", chainID)

	chainID, err = readGenesisChainID(write("scalars-first.json", `{"initial_height":1,"app_hash":"","consensus":null,"app_state":{"a":[[],{}],"b":"x"},"chain_id":"xion-devnet-1","validators":[]}`))
	require.NoError(t, err)
	require.Equal(t, "xion-devnet-1", chainID)

	_, err = readGenesisChainID(write("missing.json", `{"app_state":{"chain_id":"nested"}}`))
	require.ErrorContains(t, err, "has no chain_id")

	_, err = readGenesisChainID(write("empty.json", `{"chain_id":""}`))
	require.ErrorContains(t, err, "has no chain_id")

	_, err = readGenesisChainID(write("array.json", `[]`))
	require.ErrorContains(t, err, "not a JSON object")
}
