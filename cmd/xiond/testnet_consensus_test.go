package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	cmtcfg "github.com/cometbft/cometbft/config"
	cmted25519 "github.com/cometbft/cometbft/crypto/ed25519"
	cmtjson "github.com/cometbft/cometbft/libs/json"
	pvm "github.com/cometbft/cometbft/privval"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sm "github.com/cometbft/cometbft/state"
	"github.com/cometbft/cometbft/store"
	cmttypes "github.com/cometbft/cometbft/types"

	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
)

func TestPrepareTestnetConsensusConvertsPersistedState(t *testing.T) {
	fixture := setupTestnetConsensusFixture(t)
	newChainID := "xion-rehearsal-1"
	validatorPower := int64(900_000_000_000_000)

	// A restored data directory can contain a signing watermark produced by the
	// source key. It is moved aside with the home's key.
	foreignPV := pvm.NewFilePV(
		fixture.sourcePrivKey,
		filepath.Join(t.TempDir(), "unused-key.json"),
		fixture.config.PrivValidatorStateFile(),
	)
	foreignVote := &cmttypes.Vote{
		Type:             cmtproto.PrecommitType,
		Height:           fixture.height,
		Round:            0,
		BlockID:          fixture.blockID,
		Timestamp:        time.Now().UTC(),
		ValidatorAddress: foreignPV.GetAddress(),
		ValidatorIndex:   0,
	}
	require.NoError(t, foreignPV.SignVote(fixture.sourceChainID, foreignVote.ToProto()))

	require.NoError(t, prepareTestnetConsensus(
		fixture.config,
		newChainID,
		fixture.height,
		fixture.appHash,
		validatorPower,
		fixture.newKey,
	))

	stateDB, err := cmtcfg.DefaultDBProvider(&cmtcfg.DBContext{ID: "state", Config: fixture.config})
	require.NoError(t, err)
	stateStore := sm.NewStore(stateDB, sm.StoreOptions{})
	state, err := stateStore.Load()
	require.NoError(t, err)
	require.Equal(t, newChainID, state.ChainID)
	require.Equal(t, fixture.height, state.LastBlockHeight)
	require.Equal(t, fixture.blockID, state.LastBlockID)
	require.Equal(t, fixture.appHash, state.AppHash)
	require.Equal(t, fixture.height, state.LastHeightValidatorsChanged)

	localPubKey := fixture.newKey.PubKey()
	homePubKey, err := fixture.localPV.GetPubKey()
	require.NoError(t, err)
	require.NotEqual(t, homePubKey.Address(), localPubKey.Address())
	for _, validators := range []*cmttypes.ValidatorSet{state.LastValidators, state.Validators, state.NextValidators} {
		require.Len(t, validators.Validators, 1)
		require.Equal(t, localPubKey.Address(), validators.Validators[0].Address)
		require.Equal(t, validatorPower, validators.Validators[0].VotingPower)
	}
	for offset := -testnetValidatorHistoryRadius; offset <= testnetValidatorHistoryRadius; offset++ {
		historyHeight := fixture.height + offset
		if historyHeight <= 0 {
			continue
		}
		validators, err := stateStore.LoadValidators(historyHeight)
		require.NoError(t, err)
		require.Len(t, validators.Validators, 1)
		require.Equal(t, localPubKey.Address(), validators.Validators[0].Address)
		require.Equal(t, validatorPower, validators.Validators[0].VotingPower)
	}

	cachedGenesisBytes, err := stateDB.Get([]byte("genesisDoc"))
	require.NoError(t, err)
	var cachedGenesis cmttypes.GenesisDoc
	require.NoError(t, cmtjson.Unmarshal(cachedGenesisBytes, &cachedGenesis))
	require.Equal(t, newChainID, cachedGenesis.ChainID)
	require.NoError(t, stateStore.Close())

	blockStoreDB, err := cmtcfg.DefaultDBProvider(&cmtcfg.DBContext{ID: "blockstore", Config: fixture.config})
	require.NoError(t, err)
	blockStore := store.NewBlockStore(blockStoreDB)
	seenCommit := blockStore.LoadSeenCommit(fixture.height)
	require.NotNil(t, seenCommit)
	require.Equal(t, fixture.height, seenCommit.Height)
	require.Zero(t, seenCommit.Round)
	require.Equal(t, fixture.blockID, seenCommit.BlockID)
	require.Len(t, seenCommit.Signatures, 1)
	require.Equal(t, cmttypes.BlockIDFlagCommit, seenCommit.Signatures[0].BlockIDFlag)
	require.Equal(t, localPubKey.Address(), seenCommit.Signatures[0].ValidatorAddress)
	require.False(t, seenCommit.Signatures[0].Timestamp.IsZero())
	require.True(t, seenCommit.Signatures[0].Timestamp.After(state.LastBlockTime))
	require.NotEmpty(t, seenCommit.Signatures[0].Signature)
	vote := seenCommit.GetVote(0)
	require.True(t, localPubKey.VerifySignature(cmttypes.VoteSignBytes(newChainID, vote.ToProto()), vote.Signature))
	require.NoError(t, blockStore.Close())

	diskGenesis, err := cmttypes.GenesisDocFromFile(fixture.config.GenesisFile())
	require.NoError(t, err)
	require.Equal(t, newChainID, diskGenesis.ChainID)

	persistedPV := pvm.LoadFilePV(fixture.config.PrivValidatorKeyFile(), fixture.config.PrivValidatorStateFile())
	require.Equal(t, localPubKey, persistedPV.Key.PubKey)
	backupPV := pvm.LoadFilePV(
		fixture.config.PrivValidatorKeyFile()+testnetValidatorBackupSuffix,
		fixture.config.PrivValidatorStateFile()+testnetValidatorBackupSuffix,
	)
	require.Equal(t, homePubKey, backupPV.Key.PubKey)
	require.Equal(t, fixture.height, backupPV.LastSignState.Height, "source signing watermark is kept in the backup")
	require.Equal(t, fixture.height, persistedPV.LastSignState.Height)
	require.NotEmpty(t, persistedPV.LastSignState.SignBytes)
	require.NotEmpty(t, persistedPV.LastSignState.Signature)
	require.True(t, localPubKey.VerifySignature(persistedPV.LastSignState.SignBytes, persistedPV.LastSignState.Signature))

	walFiles, err := filepath.Glob(fixture.config.Consensus.WalFile() + "*")
	require.NoError(t, err)
	require.Empty(t, walFiles)
	addrBook, err := os.ReadFile(fixture.config.P2P.AddrBookFile())
	require.NoError(t, err)
	require.Equal(t, []byte("{}"), addrBook)
}

func TestPrepareTestnetConsensusReplacesHomeSourceValidatorKey(t *testing.T) {
	fixture := setupTestnetConsensusFixture(t)
	// The home holds a source validator key with its own signing watermark, as
	// a copy of a validator's home would.
	sourcePV := pvm.NewFilePV(
		fixture.sourcePrivKey,
		fixture.config.PrivValidatorKeyFile(),
		fixture.config.PrivValidatorStateFile(),
	)
	sourceVote := &cmttypes.Vote{
		Type:             cmtproto.PrecommitType,
		Height:           fixture.height,
		Round:            0,
		BlockID:          fixture.blockID,
		Timestamp:        time.Now().UTC(),
		ValidatorAddress: sourcePV.GetAddress(),
		ValidatorIndex:   0,
	}
	require.NoError(t, sourcePV.SignVote(fixture.sourceChainID, sourceVote.ToProto()))
	sourceKeyBytes, err := os.ReadFile(fixture.config.PrivValidatorKeyFile())
	require.NoError(t, err)

	newChainID := "xion-rehearsal-1"
	require.NoError(t, prepareTestnetConsensus(
		fixture.config,
		newChainID,
		fixture.height,
		fixture.appHash,
		900_000_000_000_000,
		fixture.newKey,
	))

	newPubKey := fixture.newKey.PubKey()
	require.NotEqual(t, fixture.sourcePrivKey.PubKey().Address(), newPubKey.Address())
	persistedPV := pvm.LoadFilePV(fixture.config.PrivValidatorKeyFile(), fixture.config.PrivValidatorStateFile())
	require.Equal(t, newPubKey, persistedPV.Key.PubKey)
	require.True(t, newPubKey.VerifySignature(persistedPV.LastSignState.SignBytes, persistedPV.LastSignState.Signature))
	backupKeyBytes, err := os.ReadFile(fixture.config.PrivValidatorKeyFile() + testnetValidatorBackupSuffix)
	require.NoError(t, err)
	require.Equal(t, sourceKeyBytes, backupKeyBytes)

	stateDB, err := cmtcfg.DefaultDBProvider(&cmtcfg.DBContext{ID: "state", Config: fixture.config})
	require.NoError(t, err)
	stateStore := sm.NewStore(stateDB, sm.StoreOptions{})
	state, err := stateStore.Load()
	require.NoError(t, err)
	require.NoError(t, stateStore.Close())
	require.Equal(t, newChainID, state.ChainID)
	for _, validators := range []*cmttypes.ValidatorSet{state.LastValidators, state.Validators, state.NextValidators} {
		require.Len(t, validators.Validators, 1)
		require.Equal(t, newPubKey.Address(), validators.Validators[0].Address)
	}

	blockStoreDB, err := cmtcfg.DefaultDBProvider(&cmtcfg.DBContext{ID: "blockstore", Config: fixture.config})
	require.NoError(t, err)
	blockStore := store.NewBlockStore(blockStoreDB)
	vote := blockStore.LoadSeenCommit(fixture.height).GetVote(0)
	require.NoError(t, blockStore.Close())
	require.Equal(t, newPubKey.Address(), vote.ValidatorAddress)
	require.True(t, newPubKey.VerifySignature(cmttypes.VoteSignBytes(newChainID, vote.ToProto()), vote.Signature))
}

func TestPrepareTestnetConsensusIgnoresIncompleteHomeSignerState(t *testing.T) {
	fixture := setupTestnetConsensusFixture(t)
	incomplete := []byte(`{"height":"1","round":0,"step":3}`)
	require.NoError(t, os.WriteFile(fixture.config.PrivValidatorStateFile(), incomplete, 0o600))

	require.NoError(t, prepareTestnetConsensus(
		fixture.config,
		"xion-rehearsal-1",
		fixture.height,
		fixture.appHash,
		900_000_000_000_000,
		fixture.newKey,
	))
	backup, err := os.ReadFile(fixture.config.PrivValidatorStateFile() + testnetValidatorBackupSuffix)
	require.NoError(t, err)
	require.Equal(t, incomplete, backup)
	persistedPV := pvm.LoadFilePV(fixture.config.PrivValidatorKeyFile(), fixture.config.PrivValidatorStateFile())
	require.Equal(t, fixture.newKey.PubKey(), persistedPV.Key.PubKey)
}

func TestPrepareTestnetConsensusRejectsNewKeyInSourceValidatorSet(t *testing.T) {
	fixture := setupTestnetConsensusFixture(t)
	before := snapshotTestHome(t, fixture.config.RootDir)

	err := prepareTestnetConsensus(
		fixture.config,
		"xion-rehearsal-1",
		fixture.height,
		fixture.appHash,
		900_000_000_000_000,
		fixture.sourcePrivKey,
	)
	require.ErrorContains(t, err, "belongs to the source consensus validator set")
	requireTestnetValidatorFilesUnchanged(t, fixture.config, before)
}

func TestInstallTestnetValidatorKeyKeepsFirstBackup(t *testing.T) {
	config := cmtcfg.DefaultConfig().SetRoot(t.TempDir())
	require.NoError(t, os.MkdirAll(filepath.Dir(config.PrivValidatorKeyFile()), 0o700))
	require.NoError(t, os.MkdirAll(filepath.Dir(config.PrivValidatorStateFile()), 0o700))
	original := pvm.GenFilePV(config.PrivValidatorKeyFile(), config.PrivValidatorStateFile())
	original.Save()
	originalKey, err := os.ReadFile(config.PrivValidatorKeyFile())
	require.NoError(t, err)

	first := cmted25519.GenPrivKey()
	_, err = installTestnetValidatorKey(config, first)
	require.NoError(t, err)
	second := cmted25519.GenPrivKey()
	installed, err := installTestnetValidatorKey(config, second)
	require.NoError(t, err)
	require.Equal(t, second.PubKey(), installed.Key.PubKey)
	require.Zero(t, installed.LastSignState.Height)

	backup, err := os.ReadFile(config.PrivValidatorKeyFile() + testnetValidatorBackupSuffix)
	require.NoError(t, err)
	require.Equal(t, originalKey, backup)
	persisted := pvm.LoadFilePV(config.PrivValidatorKeyFile(), config.PrivValidatorStateFile())
	require.Equal(t, second.PubKey(), persisted.Key.PubKey)
	require.Zero(t, persisted.LastSignState.Height)
}

func TestInstallTestnetValidatorKeyWithoutPreviousKey(t *testing.T) {
	config := cmtcfg.DefaultConfig().SetRoot(t.TempDir())
	key := cmted25519.GenPrivKey()
	installed, err := installTestnetValidatorKey(config, key)
	require.NoError(t, err)
	require.Equal(t, key.PubKey(), installed.Key.PubKey)
	// The empty key backup records that this conversion installed the key.
	backup, err := os.ReadFile(config.PrivValidatorKeyFile() + testnetValidatorBackupSuffix)
	require.NoError(t, err)
	require.Empty(t, backup)
	require.NoFileExists(t, config.PrivValidatorStateFile()+testnetValidatorBackupSuffix)

	// A second install keeps that record rather than backing up its own key.
	_, err = installTestnetValidatorKey(config, cmted25519.GenPrivKey())
	require.NoError(t, err)
	backup, err = os.ReadFile(config.PrivValidatorKeyFile() + testnetValidatorBackupSuffix)
	require.NoError(t, err)
	require.Empty(t, backup)
}

// requireTestnetValidatorFilesUnchanged asserts that a refused conversion left
// every file in the home as it was and created no validator backup.
func requireTestnetValidatorFilesUnchanged(t *testing.T, config *cmtcfg.Config, before map[string]string) {
	t.Helper()
	after := snapshotTestHome(t, config.RootDir)
	for _, path := range []string{
		config.GenesisFile(),
		config.PrivValidatorKeyFile(),
		config.PrivValidatorStateFile(),
		config.P2P.AddrBookFile(),
		config.Consensus.WalFile(),
	} {
		require.Equal(t, before[path], after[path], path)
	}
	require.NoFileExists(t, config.PrivValidatorKeyFile()+testnetValidatorBackupSuffix)
	require.NoFileExists(t, config.PrivValidatorStateFile()+testnetValidatorBackupSuffix)
}

func TestPrepareTestnetConsensusRejectsUnsafeChainID(t *testing.T) {
	tests := []struct {
		name    string
		chainID func(sourceChainID string) string
		wantErr string
	}{
		{
			name:    "same as source",
			chainID: func(source string) string { return source },
			wantErr: "must differ from source chain ID",
		},
		{
			name:    "revision of source",
			chainID: func(source string) string { return source + "-2" },
			wantErr: "is a revision of source chain ID",
		},
		{
			name:    "mainnet",
			chainID: func(string) string { return "xion-mainnet-1" },
			wantErr: "reuses the public network name xion-mainnet-N",
		},
		{
			name:    "testnet",
			chainID: func(string) string { return "xion-testnet-2" },
			wantErr: "reuses the public network name xion-testnet-N",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fixture := setupTestnetConsensusFixture(t)
			before := snapshotTestHome(t, fixture.config.RootDir)

			err := prepareTestnetConsensus(
				fixture.config,
				tc.chainID(fixture.sourceChainID),
				fixture.height,
				fixture.appHash,
				900_000_000_000_000,
				fixture.newKey,
			)
			require.ErrorContains(t, err, tc.wantErr)

			stateDB, err := cmtcfg.DefaultDBProvider(&cmtcfg.DBContext{ID: "state", Config: fixture.config})
			require.NoError(t, err)
			stateStore := sm.NewStore(stateDB, sm.StoreOptions{})
			state, err := stateStore.Load()
			require.NoError(t, err)
			require.Equal(t, fixture.sourceChainID, state.ChainID)
			require.NoError(t, stateStore.Close())

			requireTestnetValidatorFilesUnchanged(t, fixture.config, before)
		})
	}
}

type testnetConsensusFixture struct {
	config        *cmtcfg.Config
	height        int64
	appHash       []byte
	blockID       cmttypes.BlockID
	sourceChainID string
	sourcePrivKey cmted25519.PrivKey
	localPV       *pvm.FilePV
	newKey        cmted25519.PrivKey
}

func setupTestnetConsensusFixture(t *testing.T) testnetConsensusFixture {
	t.Helper()

	rootDir := t.TempDir()
	config := cmtcfg.DefaultConfig().SetRoot(rootDir)
	require.NoError(t, os.MkdirAll(filepath.Dir(config.GenesisFile()), 0o700))
	require.NoError(t, os.MkdirAll(config.DBDir(), 0o700))
	require.NoError(t, os.MkdirAll(filepath.Dir(config.PrivValidatorKeyFile()), 0o700))
	require.NoError(t, os.MkdirAll(filepath.Dir(config.PrivValidatorStateFile()), 0o700))

	sourceChainID := "source-chain"
	sourcePrivKey := cmted25519.GenPrivKey()
	sourcePubKey := sourcePrivKey.PubKey()
	sourceValidator := cmttypes.NewValidator(sourcePubKey, 100)
	sourceValidatorSet := cmttypes.NewValidatorSet([]*cmttypes.Validator{sourceValidator})
	genesis := &cmttypes.GenesisDoc{
		GenesisTime:     time.Now().Add(-time.Hour).UTC(),
		ChainID:         sourceChainID,
		InitialHeight:   1,
		ConsensusParams: cmttypes.DefaultConsensusParams(),
		Validators: []cmttypes.GenesisValidator{{
			Address: sourcePubKey.Address(),
			PubKey:  sourcePubKey,
			Power:   sourceValidator.VotingPower,
		}},
	}
	appGenesis := &genutiltypes.AppGenesis{
		GenesisTime:   genesis.GenesisTime,
		ChainID:       genesis.ChainID,
		InitialHeight: genesis.InitialHeight,
		AppHash:       genesis.AppHash,
		Consensus: &genutiltypes.ConsensusGenesis{
			Validators: genesis.Validators,
			Params:     genesis.ConsensusParams,
		},
	}
	require.NoError(t, appGenesis.SaveAs(config.GenesisFile()))

	// A recent snapshot may be ahead of the converting machine's clock.
	blockTime := time.Now().Add(time.Minute).UTC()
	block := cmttypes.MakeBlock(1, nil, &cmttypes.Commit{}, nil)
	block.Header.ChainID = sourceChainID
	block.Header.Time = blockTime
	block.Header.ValidatorsHash = sourceValidatorSet.Hash()
	block.Header.ProposerAddress = sourcePubKey.Address()
	require.NoError(t, block.ValidateBasic())
	blockParts, err := block.MakePartSet(cmttypes.BlockPartSizeBytes)
	require.NoError(t, err)
	blockID := cmttypes.BlockID{Hash: block.Hash(), PartSetHeader: blockParts.Header()}
	sourceSeenCommit := &cmttypes.Commit{
		Height:     1,
		Round:      0,
		BlockID:    blockID,
		Signatures: []cmttypes.CommitSig{cmttypes.NewCommitSigAbsent()},
	}

	blockStoreDB, err := cmtcfg.DefaultDBProvider(&cmtcfg.DBContext{ID: "blockstore", Config: config})
	require.NoError(t, err)
	blockStore := store.NewBlockStore(blockStoreDB)
	blockStore.SaveBlock(block, blockParts, sourceSeenCommit)
	require.NoError(t, blockStore.Close())

	state, err := sm.MakeGenesisState(genesis)
	require.NoError(t, err)
	appHash := bytes.Repeat([]byte{0xAB}, 32)
	state.LastBlockHeight = 1
	state.LastBlockID = blockID
	state.LastBlockTime = blockTime
	state.LastValidators = sourceValidatorSet.Copy()
	state.Validators = sourceValidatorSet.Copy()
	state.NextValidators = sourceValidatorSet.Copy()
	state.LastHeightValidatorsChanged = 1
	state.AppHash = bytes.Clone(appHash)

	stateDB, err := cmtcfg.DefaultDBProvider(&cmtcfg.DBContext{ID: "state", Config: config})
	require.NoError(t, err)
	stateStore := sm.NewStore(stateDB, sm.StoreOptions{})
	require.NoError(t, stateStore.Save(state))
	cachedGenesis, err := cmtjson.Marshal(genesis)
	require.NoError(t, err)
	require.NoError(t, stateDB.SetSync([]byte("genesisDoc"), cachedGenesis))
	require.NoError(t, stateStore.Close())

	localPV := pvm.GenFilePV(config.PrivValidatorKeyFile(), config.PrivValidatorStateFile())
	localPV.Save()

	require.NoError(t, os.MkdirAll(filepath.Dir(config.Consensus.WalFile()), 0o700))
	require.NoError(t, os.WriteFile(config.Consensus.WalFile(), []byte("source WAL"), 0o600))
	require.NoError(t, os.WriteFile(config.Consensus.WalFile()+".000", []byte("rotated source WAL"), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Dir(config.P2P.AddrBookFile()), 0o700))
	require.NoError(t, os.WriteFile(config.P2P.AddrBookFile(), []byte(`{"addrs":["source-peer"]}`), 0o600))

	return testnetConsensusFixture{
		config:        config,
		height:        1,
		appHash:       appHash,
		blockID:       blockID,
		sourceChainID: sourceChainID,
		sourcePrivKey: sourcePrivKey,
		localPV:       localPV,
		newKey:        cmted25519.GenPrivKey(),
	}
}
