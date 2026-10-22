package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	cmtcfg "github.com/cometbft/cometbft/config"
	sm "github.com/cometbft/cometbft/state"
	"github.com/cometbft/cometbft/store"
	cmttypes "github.com/cometbft/cometbft/types"

	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
)

const resumeTestValidatorPower int64 = 900_000_000_000_000

// convertedTestnetConsensus is what the consensus half of a conversion leaves
// on disk, read back for comparison.
type convertedTestnetConsensus struct {
	state       sm.State
	seenCommit  *cmttypes.Commit
	signerState []byte
	genesis     []byte
	marker      []byte
}

func readConvertedTestnetConsensus(t *testing.T, config *cmtcfg.Config, height int64) convertedTestnetConsensus {
	t.Helper()
	stateDB, err := cmtcfg.DefaultDBProvider(&cmtcfg.DBContext{ID: "state", Config: config})
	require.NoError(t, err)
	stateStore := sm.NewStore(stateDB, sm.StoreOptions{})
	state, err := stateStore.Load()
	require.NoError(t, err)
	marker, err := stateDB.Get(testnetConversionCompleteKey)
	require.NoError(t, err)
	require.NoError(t, stateStore.Close())

	blockDB, err := cmtcfg.DefaultDBProvider(&cmtcfg.DBContext{ID: "blockstore", Config: config})
	require.NoError(t, err)
	blockStore := store.NewBlockStore(blockDB)
	seenCommit := blockStore.LoadSeenCommit(height)
	require.NoError(t, blockStore.Close())

	signerState, err := os.ReadFile(config.PrivValidatorStateFile())
	require.NoError(t, err)
	genesis, err := os.ReadFile(config.GenesisFile())
	require.NoError(t, err)
	return convertedTestnetConsensus{state: state, seenCommit: seenCommit, signerState: signerState, genesis: genesis, marker: marker}
}

// The application half of a conversion only commits with the first new block.
// A run that exits before then (for example on a busy p2p port) leaves the
// consensus half converted over an unconverted application. Rerunning the same
// conversion must finish it: consensus is accepted as already converted and
// left exactly as the first run wrote it.
func TestPrepareTestnetConsensusResumesInterruptedConversion(t *testing.T) {
	fixture := setupTestnetConsensusFixture(t)
	chainID := "xion-rehearsal-1"
	require.NoError(t, prepareTestnetConsensus(fixture.config, chainID, fixture.height, fixture.appHash, resumeTestValidatorPower))
	first := readConvertedTestnetConsensus(t, fixture.config, fixture.height)
	require.Equal(t, chainID, string(first.marker))

	// The interrupted node may already have written a WAL for the new chain.
	require.NoError(t, os.WriteFile(fixture.config.Consensus.WalFile(), []byte("testnet WAL"), 0o600))

	require.NoError(t, prepareTestnetConsensus(fixture.config, chainID, fixture.height, fixture.appHash, resumeTestValidatorPower))
	second := readConvertedTestnetConsensus(t, fixture.config, fixture.height)
	require.Equal(t, first, second)
	wal, err := os.ReadFile(fixture.config.Consensus.WalFile())
	require.NoError(t, err)
	require.Equal(t, "testnet WAL", string(wal), "resuming must not rewrite consensus files")
}

func TestPrepareTestnetConsensusInterruptedConversionNeedsSameChainID(t *testing.T) {
	fixture := setupTestnetConsensusFixture(t)
	require.NoError(t, prepareTestnetConsensus(fixture.config, "xion-rehearsal-1", fixture.height, fixture.appHash, resumeTestValidatorPower))
	first := readConvertedTestnetConsensus(t, fixture.config, fixture.height)

	err := prepareTestnetConsensus(fixture.config, "xion-rehearsal-2", fixture.height, fixture.appHash, resumeTestValidatorPower)
	require.ErrorContains(t, err, `interrupted after converting consensus state to chain ID "xion-rehearsal-1"`)
	require.ErrorContains(t, err, "rerun with that chain ID")
	require.Equal(t, first, readConvertedTestnetConsensus(t, fixture.config, fixture.height))
}

func TestPrepareTestnetConsensusRejectsUnresumableInterruptedConversion(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(t *testing.T, fixture testnetConsensusFixture) (appHash []byte, power int64)
		wantErr string
	}{
		{
			name: "genesis file not yet rewritten",
			mutate: func(t *testing.T, fixture testnetConsensusFixture) ([]byte, int64) {
				appGenesis, err := genutiltypes.AppGenesisFromFile(fixture.config.GenesisFile())
				require.NoError(t, err)
				appGenesis.ChainID = fixture.sourceChainID
				require.NoError(t, appGenesis.SaveAs(fixture.config.GenesisFile()))
				return fixture.appHash, resumeTestValidatorPower
			},
			wantErr: `genesis file chain ID is "source-chain"`,
		},
		{
			name: "different application state",
			mutate: func(_ *testing.T, fixture testnetConsensusFixture) ([]byte, int64) {
				appHash := append([]byte(nil), fixture.appHash...)
				appHash[0] ^= 0xFF
				return appHash, resumeTestValidatorPower
			},
			wantErr: "consensus application hash at height 1 does not match",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fixture := setupTestnetConsensusFixture(t)
			chainID := "xion-rehearsal-1"
			require.NoError(t, prepareTestnetConsensus(fixture.config, chainID, fixture.height, fixture.appHash, resumeTestValidatorPower))
			appHash, power := tc.mutate(t, fixture)

			err := prepareTestnetConsensus(fixture.config, chainID, fixture.height, appHash, power)
			require.ErrorContains(t, err, "interrupted and cannot be resumed")
			require.ErrorContains(t, err, tc.wantErr)
			require.ErrorContains(t, err, "start again from a fresh copy")
		})
	}
}

func setTestnetConversionMarker(t *testing.T, config *cmtcfg.Config, value []byte) {
	t.Helper()
	stateDB, err := cmtcfg.DefaultDBProvider(&cmtcfg.DBContext{ID: "state", Config: config})
	require.NoError(t, err)
	if value == nil {
		require.NoError(t, stateDB.DeleteSync(testnetConversionCompleteKey))
	} else {
		require.NoError(t, stateDB.SetSync(testnetConversionCompleteKey, value))
	}
	require.NoError(t, stateDB.Close())
}

// A run killed after the genesis writes but before the network-file reset
// finished leaves the source WAL and address book behind and no completion
// marker. Resuming must finish that reset.
func TestPrepareTestnetConsensusResumeFinishesNetworkReset(t *testing.T) {
	for name, marker := range map[string][]byte{
		"no marker":                 nil,
		"marker for an older chain": []byte("xion-rehearsal-0"),
	} {
		t.Run(name, func(t *testing.T) {
			fixture := setupTestnetConsensusFixture(t)
			chainID := "xion-rehearsal-1"
			require.NoError(t, prepareTestnetConsensus(fixture.config, chainID, fixture.height, fixture.appHash, resumeTestValidatorPower))
			setTestnetConversionMarker(t, fixture.config, marker)
			require.NoError(t, os.WriteFile(fixture.config.Consensus.WalFile(), []byte("source WAL"), 0o600))
			require.NoError(t, os.WriteFile(fixture.config.Consensus.WalFile()+".000", []byte("rotated source WAL"), 0o600))
			require.NoError(t, os.WriteFile(fixture.config.P2P.AddrBookFile(), []byte(`{"addrs":["source-peer"]}`), 0o600))
			before := readConvertedTestnetConsensus(t, fixture.config, fixture.height)

			require.NoError(t, prepareTestnetConsensus(fixture.config, chainID, fixture.height, fixture.appHash, resumeTestValidatorPower))
			after := readConvertedTestnetConsensus(t, fixture.config, fixture.height)
			require.Equal(t, chainID, string(after.marker))
			after.marker = before.marker
			require.Equal(t, before, after, "only the network files and the marker may change")
			walFiles, err := filepath.Glob(fixture.config.Consensus.WalFile() + "*")
			require.NoError(t, err)
			require.Empty(t, walFiles)
			addrBook, err := os.ReadFile(fixture.config.P2P.AddrBookFile())
			require.NoError(t, err)
			require.Equal(t, "{}", string(addrBook))
		})
	}
}

// A conversion whose network-file reset failed can be rerun once the cause is
// fixed, and the rerun completes the reset.
func TestPrepareTestnetConsensusRetriesFailedNetworkReset(t *testing.T) {
	fixture := setupTestnetConsensusFixture(t)
	chainID := "xion-rehearsal-1"
	walPath := fixture.config.Consensus.WalFile()
	require.NoError(t, os.Remove(walPath))
	require.NoError(t, os.Mkdir(walPath, 0o700))

	err := prepareTestnetConsensus(fixture.config, chainID, fixture.height, fixture.appHash, resumeTestValidatorPower)
	require.ErrorContains(t, err, "consensus WAL path is a directory")
	require.Empty(t, readConvertedTestnetConsensus(t, fixture.config, fixture.height).marker)

	require.NoError(t, os.Remove(walPath))
	require.NoError(t, prepareTestnetConsensus(fixture.config, chainID, fixture.height, fixture.appHash, resumeTestValidatorPower))
	require.Equal(t, chainID, string(readConvertedTestnetConsensus(t, fixture.config, fixture.height).marker))
	walFiles, err := filepath.Glob(walPath + "*")
	require.NoError(t, err)
	require.Empty(t, walFiles, "the rotated source WAL must be removed on the rerun")
}
