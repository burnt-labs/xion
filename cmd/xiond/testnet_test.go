package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInPlaceTestnetDeclinedLeavesHomeUntouched(t *testing.T) {
	home := t.TempDir()
	marker := filepath.Join(home, "source-state")
	require.NoError(t, os.WriteFile(marker, []byte("unchanged"), 0o600))
	cmd := inPlaceTestnetCmd()
	cmd.SetArgs([]string{"fork-test", "operator", "--home", home})
	cmd.SetIn(strings.NewReader("no\n"))
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	require.NoError(t, cmd.Execute())
	data, err := os.ReadFile(marker)
	require.NoError(t, err)
	require.Equal(t, "unchanged", string(data))
	entries, err := os.ReadDir(home)
	require.NoError(t, err)
	require.Len(t, entries, 1, "declining conversion must not initialize or open node state")
}

func TestInPlaceTestnetRejectsStandaloneBeforeMutation(t *testing.T) {
	home := t.TempDir()
	cmd := inPlaceTestnetCmd()
	cmd.SetArgs([]string{"fork-test", "operator", "--home", home, "--with-comet=false", "--skip-confirmation"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	require.ErrorContains(t, cmd.Execute(), "requires in-process CometBFT")
	entries, err := os.ReadDir(home)
	require.NoError(t, err)
	require.Empty(t, entries, "rejected standalone conversion must not initialize or open node state")
}

func TestInPlaceTestnetRejectsPublicChainIDBeforeMutation(t *testing.T) {
	for _, chainID := range []string{"xion-mainnet-1", "xion-testnet-2", "xion-mainnet-2", "XION-Testnet-2"} {
		t.Run(chainID, func(t *testing.T) {
			home := t.TempDir()
			cmd := inPlaceTestnetCmd()
			cmd.SetArgs([]string{chainID, "operator", "--home", home})
			cmd.SetIn(strings.NewReader("y\n"))
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			require.ErrorContains(t, cmd.Execute(), "reuses the public network name")
			entries, err := os.ReadDir(home)
			require.NoError(t, err)
			require.Empty(t, entries, "rejected conversion must not initialize or open node state")
		})
	}
}
