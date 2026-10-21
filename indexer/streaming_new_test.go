package indexer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"
)

func TestNew_UnusableHomeDir(t *testing.T) {
	_, cdc, addrCodec := setupTest(t)

	// the home directory is a regular file, so the data directory cannot be created
	homeDir := filepath.Join(t.TempDir(), "home")
	require.NoError(t, os.WriteFile(homeDir, []byte("not a directory"), 0o600))

	require.Panics(t, func() {
		New(homeDir, cdc, addrCodec, log.NewNopLogger())
	})
}
