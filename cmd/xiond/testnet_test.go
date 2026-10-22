package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	cmtcfg "github.com/cometbft/cometbft/config"

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
	files := map[string]string{}
	require.NoError(t, filepath.Walk(home, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			files[path] = "symlink to " + target
			return err
		}
		data, err := os.ReadFile(path)
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
	before := writeTestSourceHome(t, home, "xion-mainnet-1")
	cmd := newTestInPlaceTestnetCmd(t, home, nil)
	cmd.SetArgs([]string{"fork-test", "operator", "--home", home})
	cmd.SetIn(strings.NewReader("no\n"))
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)
	require.NoError(t, cmd.Execute())
	require.Contains(t, stderr.String(), "Converting home "+home+" from source chain xion-mainnet-1 to new chain fork-test.")
	require.Contains(t, stderr.String(), "New validator consensus address: ")
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

func TestInPlaceTestnetRejectsGRPCOnlyBeforeMutation(t *testing.T) {
	home := t.TempDir()
	before := writeTestSourceHome(t, home, "xion-devnet-1")
	cmd := newTestInPlaceTestnetCmd(t, home, nil)
	cmd.SetArgs([]string{"fork-test", "operator", "--home", home, "--grpc-only", "--skip-confirmation"})
	require.ErrorContains(t, cmd.Execute(), "cannot run with --grpc-only")
	require.Equal(t, before, snapshotTestHome(t, home), "a gRPC-only run must not open or convert node state")

	// The same setting from the node's configuration or environment.
	cmd = newTestInPlaceTestnetCmd(t, home, func(serverCtx *server.Context) {
		serverCtx.Viper.Set("grpc-only", true)
	})
	cmd.SetArgs([]string{"fork-test", "operator", "--home", home, "--skip-confirmation"})
	require.ErrorContains(t, cmd.Execute(), "cannot run with --grpc-only")
	require.Equal(t, before, snapshotTestHome(t, home))
}

func TestInPlaceTestnetRejectsPathsOutsideHome(t *testing.T) {
	outside := t.TempDir()
	for name, configure := range map[string]func(*server.Context){
		"priv_validator_key_file": func(serverCtx *server.Context) {
			serverCtx.Config.PrivValidatorKey = filepath.Join(outside, "priv_validator_key.json")
		},
		"priv_validator_state_file": func(serverCtx *server.Context) {
			serverCtx.Config.PrivValidatorState = filepath.Join(outside, "priv_validator_state.json")
		},
		"db_dir": func(serverCtx *server.Context) {
			serverCtx.Config.DBPath = "../live-data"
		},
		"consensus.wal_file": func(serverCtx *server.Context) {
			serverCtx.Config.Consensus.WalPath = filepath.Join(outside, "cs.wal", "wal")
		},
		"p2p.addr_book_file": func(serverCtx *server.Context) {
			serverCtx.Config.P2P.AddrBook = filepath.Join(outside, "addrbook.json")
		},
	} {
		t.Run(name, func(t *testing.T) {
			home := t.TempDir()
			before := writeTestSourceHome(t, home, "xion-devnet-1")
			cmd := newTestInPlaceTestnetCmd(t, home, configure)
			cmd.SetArgs([]string{"fork-test", "operator", "--home", home, "--skip-confirmation"})
			err := cmd.Execute()
			require.ErrorContains(t, err, name+" (")
			require.ErrorContains(t, err, "is outside the home")
			require.Equal(t, before, snapshotTestHome(t, home))
		})
	}
	entries, err := os.ReadDir(outside)
	require.NoError(t, err)
	require.Empty(t, entries)
}

func TestInPlaceTestnetRejectsSymlinksOutsideHome(t *testing.T) {
	for name, tc := range map[string]struct {
		link func(t *testing.T, home, outside string)
		want string
	}{
		// A copied home whose data/ still links to the source node's data.
		// The signing state is the first setting under data/ that is checked.
		"data directory": {
			link: func(t *testing.T, home, outside string) {
				require.NoError(t, os.Mkdir(filepath.Join(outside, "data"), 0o700))
				require.NoError(t, os.Symlink(filepath.Join(outside, "data"), filepath.Join(home, "data")))
			},
			want: "priv_validator_state_file (",
		},
		// A key file linked to the source validator's key.
		"validator key": {
			link: func(t *testing.T, home, outside string) {
				require.NoError(t, os.WriteFile(filepath.Join(outside, "priv_validator_key.json"), []byte("source key"), 0o600))
				require.NoError(t, os.Symlink(filepath.Join(outside, "priv_validator_key.json"), filepath.Join(home, "config", "priv_validator_key.json")))
			},
			want: "priv_validator_key_file (",
		},
		// A dangling link: writing the signing state would create its target.
		"dangling signing state": {
			link: func(t *testing.T, home, outside string) {
				require.NoError(t, os.Mkdir(filepath.Join(home, "data"), 0o700))
				require.NoError(t, os.Symlink(filepath.Join(outside, "missing", "priv_validator_state.json"), filepath.Join(home, "data", "priv_validator_state.json")))
			},
			want: "symlink to a missing target",
		},
	} {
		t.Run(name, func(t *testing.T) {
			home, outside := t.TempDir(), t.TempDir()
			writeTestSourceHome(t, home, "xion-devnet-1")
			tc.link(t, home, outside)
			beforeHome, beforeOutside := snapshotTestHome(t, home), snapshotTestHome(t, outside)
			cmd := newTestInPlaceTestnetCmd(t, home, nil)
			cmd.SetArgs([]string{"fork-test", "operator", "--home", home, "--skip-confirmation"})
			err := cmd.Execute()
			require.ErrorContains(t, err, tc.want)
			require.Equal(t, beforeHome, snapshotTestHome(t, home))
			require.Equal(t, beforeOutside, snapshotTestHome(t, outside), "a refused conversion must not write through the link")
		})
	}
}

// A setting that names an entry outside the home is refused even when that
// entry is a symlink into the home: the conversion renames the key file and
// removes the address book, which moves or deletes the outside link itself.
func TestInPlaceTestnetRejectsOutsideLinksIntoHome(t *testing.T) {
	for name, tc := range map[string]struct {
		file      string
		configure func(serverCtx *server.Context, link string)
	}{
		"priv_validator_key_file": {
			file: "priv_validator_key.json",
			configure: func(serverCtx *server.Context, link string) {
				serverCtx.Config.PrivValidatorKey = link
			},
		},
		"p2p.addr_book_file": {
			file: "addrbook.json",
			configure: func(serverCtx *server.Context, link string) {
				serverCtx.Config.P2P.AddrBook = link
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			home, outside := t.TempDir(), t.TempDir()
			writeTestSourceHome(t, home, "xion-devnet-1")
			require.NoError(t, os.WriteFile(filepath.Join(home, "config", tc.file), []byte("home file"), 0o600))
			link := filepath.Join(outside, tc.file)
			require.NoError(t, os.Symlink(filepath.Join(home, "config", tc.file), link))
			beforeHome, beforeOutside := snapshotTestHome(t, home), snapshotTestHome(t, outside)
			cmd := newTestInPlaceTestnetCmd(t, home, func(serverCtx *server.Context) {
				tc.configure(serverCtx, link)
			})
			cmd.SetArgs([]string{"fork-test", "operator", "--home", home, "--skip-confirmation"})
			err := cmd.Execute()
			require.ErrorContains(t, err, name+" (")
			require.ErrorContains(t, err, "is outside the home")
			require.Equal(t, beforeHome, snapshotTestHome(t, home))
			require.Equal(t, beforeOutside, snapshotTestHome(t, outside), "a refused conversion must not move or remove the outside link")
		})
	}
}

// A ".." after a directory is refused: the conversion reaches some files
// through the path as configured, which the kernel resolves after following a
// symlink, and some through filepath.Dir, which drops "dir/.." lexically.
func TestInPlaceTestnetRejectsDotDotAfterDirectory(t *testing.T) {
	for name, tc := range map[string]struct {
		setting   string
		configure func(serverCtx *server.Context, home, outside string)
	}{
		// home/s/../key.json with s linked out: the rename follows s.
		"priv_validator_key_file": {
			setting: "priv_validator_key_file (",
			configure: func(serverCtx *server.Context, home, outside string) {
				require.NoError(t, os.Mkdir(filepath.Join(outside, "subdir"), 0o700))
				require.NoError(t, os.WriteFile(filepath.Join(outside, "priv_validator_key.json"), []byte("source key"), 0o600))
				require.NoError(t, os.Symlink(filepath.Join(outside, "subdir"), filepath.Join(home, "s")))
				serverCtx.Config.PrivValidatorKey = home + "/s/../priv_validator_key.json"
			},
		},
		// outside/s/../wal with s linked into the home resolves inside it,
		// but the WAL reset lists filepath.Dir of the setting: outside.
		"consensus.wal_file": {
			setting: "consensus.wal_file (",
			configure: func(serverCtx *server.Context, home, outside string) {
				require.NoError(t, os.Mkdir(filepath.Join(home, "sub"), 0o700))
				require.NoError(t, os.WriteFile(filepath.Join(outside, "wal"), []byte("source wal"), 0o600))
				require.NoError(t, os.WriteFile(filepath.Join(outside, "wal.001"), []byte("source wal"), 0o600))
				require.NoError(t, os.Symlink(filepath.Join(home, "sub"), filepath.Join(outside, "s")))
				serverCtx.Config.Consensus.WalPath = outside + "/s/../wal"
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			home, outside := t.TempDir(), t.TempDir()
			writeTestSourceHome(t, home, "xion-devnet-1")
			// The configure hook runs here and creates the links.
			cmd := newTestInPlaceTestnetCmd(t, home, func(serverCtx *server.Context) {
				tc.configure(serverCtx, home, outside)
			})
			beforeHome, beforeOutside := snapshotTestHome(t, home), snapshotTestHome(t, outside)
			cmd.SetArgs([]string{"fork-test", "operator", "--home", home, "--skip-confirmation"})
			err := cmd.Execute()
			require.ErrorContains(t, err, tc.setting)
			require.ErrorContains(t, err, `has ".." after a directory`)
			require.Equal(t, beforeHome, snapshotTestHome(t, home))
			require.Equal(t, beforeOutside, snapshotTestHome(t, outside))
		})
	}
}

func TestHasInnerDotDot(t *testing.T) {
	for path, want := range map[string]bool{
		"/home/node/config/key.json":   false,
		"../copy/config/key.json":      false,
		"/../home/node/key.json":       false,
		"/home/./node//key.json":       false,
		"/home/node/s/../key.json":     true,
		"copy/../other/data":           true,
		"/home/node/data/..":           true,
		"/home/node/data/..file/state": false,
	} {
		require.Equal(t, want, hasInnerDotDot(path), path)
	}
}

// The databases are opened beneath db_dir and data/, so a database linked to
// the source node's is refused even when the directory itself is in the home.
func TestInPlaceTestnetRejectsDatabaseLinksOutsideHome(t *testing.T) {
	for name, tc := range map[string]struct {
		dbPath, db string
	}{
		"application database":                    {dbPath: "data", db: "application.db"},
		"consensus database in a separate db_dir": {dbPath: "cometbft", db: "blockstore.db"},
		"nested link":                             {dbPath: "data", db: filepath.Join("snapshots", "metadata.db")},
	} {
		t.Run(name, func(t *testing.T) {
			home, outside := t.TempDir(), t.TempDir()
			writeTestSourceHome(t, home, "xion-devnet-1")
			require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(home, tc.dbPath, tc.db)), 0o700))
			require.NoError(t, os.MkdirAll(filepath.Join(outside, "source.db"), 0o700))
			require.NoError(t, os.WriteFile(filepath.Join(outside, "source.db", "000001.ldb"), []byte("source state"), 0o600))
			require.NoError(t, os.Symlink(filepath.Join(outside, "source.db"), filepath.Join(home, tc.dbPath, tc.db)))
			beforeHome, beforeOutside := snapshotTestHome(t, home), snapshotTestHome(t, outside)
			cmd := newTestInPlaceTestnetCmd(t, home, func(serverCtx *server.Context) {
				serverCtx.Config.DBPath = tc.dbPath
			})
			cmd.SetArgs([]string{"fork-test", "operator", "--home", home, "--skip-confirmation"})
			err := cmd.Execute()
			require.ErrorContains(t, err, tc.db+" under ")
			require.ErrorContains(t, err, "outside the home")
			require.Equal(t, beforeHome, snapshotTestHome(t, home))
			require.Equal(t, beforeOutside, snapshotTestHome(t, outside))
		})
	}
}

// A database linked to another directory inside the home is fine.
func TestTestnetPathsInHomeAllowsDatabaseLinksInsideHome(t *testing.T) {
	home := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(home, "config"), 0o700))
	require.NoError(t, os.MkdirAll(filepath.Join(home, "data"), 0o700))
	require.NoError(t, os.MkdirAll(filepath.Join(home, "disk2", "application.db"), 0o700))
	require.NoError(t, os.Symlink(filepath.Join(home, "disk2", "application.db"), filepath.Join(home, "data", "application.db")))
	config := cmtcfg.DefaultConfig()
	config.SetRoot(home)
	require.NoError(t, checkTestnetPathsInHome(config))
}

func TestTestnetPathsInHomeFollowsSymlinkedHome(t *testing.T) {
	target := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(target, "config"), 0o700))
	require.NoError(t, os.Mkdir(filepath.Join(target, "data"), 0o700))
	home := filepath.Join(t.TempDir(), "home")
	require.NoError(t, os.Symlink(target, home))
	config := cmtcfg.DefaultConfig()
	config.SetRoot(home)
	require.NoError(t, checkTestnetPathsInHome(config), "a home reached through a symlink keeps its own files inside it")
}

func TestInPlaceTestnetRequiresExplicitHome(t *testing.T) {
	home := t.TempDir()
	before := writeTestSourceHome(t, home, "xion-mainnet-1")
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

func TestInPlaceTestnetRejectsRemoteSignerHome(t *testing.T) {
	home := t.TempDir()
	before := writeTestSourceHome(t, home, "xion-mainnet-1")
	cmd := newTestInPlaceTestnetCmd(t, home, func(serverCtx *server.Context) {
		serverCtx.Config.PrivValidatorListenAddr = "tcp://127.0.0.1:26659"
	})
	cmd.SetArgs([]string{"fork-test", "operator", "--home", home, "--skip-confirmation"})
	require.ErrorContains(t, cmd.Execute(), "uses a remote signer")
	require.Equal(t, before, snapshotTestHome(t, home))
}

func TestInPlaceTestnetRejectsUnsafeChainIDBeforeConfirmation(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		chainID string
		wantErr string
	}{
		{name: "testnet fork named mainnet", source: "xion-testnet-2", chainID: "xion-mainnet-1", wantErr: "reuses the public network name xion-mainnet-N"},
		{name: "mainnet fork named testnet", source: "xion-mainnet-1", chainID: "xion-testnet-2", wantErr: "reuses the public network name xion-testnet-N"},
		{name: "mainnet next revision", source: "xion-mainnet-1", chainID: "xion-mainnet-2", wantErr: "reuses the public network name xion-mainnet-N"},
		{name: "public name in another case", source: "xion-devnet-1", chainID: "XION-Testnet-2", wantErr: "reuses the public network name xion-testnet-N"},
		{name: "same as source", source: "xion-devnet-1", chainID: "xion-devnet-1", wantErr: "must differ from source chain ID"},
		{name: "revision of source", source: "xion-devnet-1", chainID: "xion-devnet-2", wantErr: "is a revision of source chain ID"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			before := writeTestSourceHome(t, home, tc.source)
			cmd := newTestInPlaceTestnetCmd(t, home, nil)
			cmd.SetArgs([]string{tc.chainID, "operator", "--home", home})
			cmd.SetIn(strings.NewReader("y\n"))
			require.ErrorContains(t, cmd.Execute(), tc.wantErr)
			require.Equal(t, before, snapshotTestHome(t, home))
		})
	}
}

// A home whose genesis already carries the new chain ID and whose validator key
// was replaced by a conversion is a rerun after an interrupted conversion. It
// passes the command-level source check (consensus state decides whether it can
// be finished), but a public network name is still refused.
func TestInPlaceTestnetRerunAfterInterruptedConversion(t *testing.T) {
	home := t.TempDir()
	writeTestSourceHome(t, home, "fork-test")
	require.NoError(t, os.WriteFile(filepath.Join(home, "config", "priv_validator_key.json"+testnetValidatorBackupSuffix), []byte("{}"), 0o600))
	before := snapshotTestHome(t, home)

	cmd := newTestInPlaceTestnetCmd(t, home, nil)
	cmd.SetArgs([]string{"fork-test", "operator", "--home", home})
	cmd.SetIn(strings.NewReader("no\n"))
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)
	require.NoError(t, cmd.Execute())
	require.Contains(t, stderr.String(), "Home "+home+" already has chain ID fork-test in its genesis file: finishing the interrupted conversion")
	require.NotContains(t, stderr.String(), "New validator consensus address")
	require.Contains(t, stderr.String(), "Conversion cancelled.")
	require.Equal(t, before, snapshotTestHome(t, home))

	// Another revision of the chain it was converted to points back at it.
	cmd = newTestInPlaceTestnetCmd(t, home, nil)
	cmd.SetArgs([]string{"fork-test-2", "operator", "--home", home})
	cmd.SetIn(strings.NewReader("y\n"))
	err := cmd.Execute()
	require.ErrorContains(t, err, "is a revision of source chain ID")
	require.ErrorContains(t, err, "already converted to fork-test; to finish an interrupted conversion, rerun with that chain ID")
	require.Equal(t, before, snapshotTestHome(t, home))

	home = t.TempDir()
	writeTestSourceHome(t, home, "xion-testnet-2")
	require.NoError(t, os.WriteFile(filepath.Join(home, "config", "priv_validator_key.json"+testnetValidatorBackupSuffix), []byte("{}"), 0o600))
	before = snapshotTestHome(t, home)
	cmd = newTestInPlaceTestnetCmd(t, home, nil)
	cmd.SetArgs([]string{"xion-testnet-2", "operator", "--home", home})
	cmd.SetIn(strings.NewReader("y\n"))
	require.ErrorContains(t, cmd.Execute(), "reuses the public network name")
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

	_, err = readGenesisChainID(write("missing.json", `{"app_state":{"chain_id":"nested"}}`))
	require.ErrorContains(t, err, "has no chain_id")

	_, err = readGenesisChainID(write("empty.json", `{"chain_id":""}`))
	require.ErrorContains(t, err, "has no chain_id")

	_, err = readGenesisChainID(write("array.json", `[]`))
	require.ErrorContains(t, err, "not a JSON object")
}

func TestResolveTestnetPath(t *testing.T) {
	dir, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "real", "sub"), 0o700))
	require.NoError(t, os.Symlink(filepath.Join(dir, "real", "sub"), filepath.Join(dir, "link")))

	// A relative path starts from the real working directory.
	t.Chdir(filepath.Join(dir, "link"))
	entry, target, err := resolveTestnetPath("new/file")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(dir, "real", "sub", "new", "file"), target)
	require.Equal(t, target, entry)

	// ".." after a link leaves the link's target, not the link.
	entry, target, err = resolveTestnetPath(dir + "/link/../file")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(dir, "real", "file"), target)
	require.Equal(t, target, entry)

	// The entry keeps the link's own name; the target follows it.
	entry, target, err = resolveTestnetPath(filepath.Join(dir, "link"))
	require.NoError(t, err)
	require.Equal(t, filepath.Join(dir, "link"), entry)
	require.Equal(t, filepath.Join(dir, "real", "sub"), target)

	// ".." below a directory that does not exist yet is refused.
	_, _, err = resolveTestnetPath(dir + "/missing/../file")
	require.ErrorContains(t, err, `".." below the missing`)
}
