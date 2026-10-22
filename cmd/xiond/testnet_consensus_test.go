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
	// source key while config contains the fresh local fork key.
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

	localPubKey, err := fixture.localPV.GetPubKey()
	require.NoError(t, err)
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

func TestPrepareTestnetConsensusRejectsUsedLocalSigner(t *testing.T) {
	fixture := setupTestnetConsensusFixture(t)
	localPubKey, err := fixture.localPV.GetPubKey()
	require.NoError(t, err)
	vote := &cmttypes.Vote{
		Type:             cmtproto.PrecommitType,
		Height:           fixture.height,
		Round:            0,
		BlockID:          fixture.blockID,
		Timestamp:        time.Now().UTC(),
		ValidatorAddress: localPubKey.Address(),
		ValidatorIndex:   0,
	}
	require.NoError(t, fixture.localPV.SignVote("another-local-chain", vote.ToProto()))

	err = prepareTestnetConsensus(
		fixture.config,
		"xion-rehearsal-1",
		fixture.height,
		fixture.appHash,
		900_000_000_000_000,
	)
	require.ErrorContains(t, err, "local validator key already has persisted signing state")

	stateDB, openErr := cmtcfg.DefaultDBProvider(&cmtcfg.DBContext{ID: "state", Config: fixture.config})
	require.NoError(t, openErr)
	stateStore := sm.NewStore(stateDB, sm.StoreOptions{})
	state, loadErr := stateStore.Load()
	require.NoError(t, loadErr)
	require.Equal(t, fixture.sourceChainID, state.ChainID)
	require.NoError(t, stateStore.Close())
}

func TestPrepareTestnetConsensusRejectsIncompleteSignerState(t *testing.T) {
	fixture := setupTestnetConsensusFixture(t)
	require.NoError(t, os.WriteFile(
		fixture.config.PrivValidatorStateFile(),
		[]byte(`{"height":"1","round":0,"step":3}`),
		0o600,
	))

	err := prepareTestnetConsensus(
		fixture.config,
		"xion-rehearsal-1",
		fixture.height,
		fixture.appHash,
		900_000_000_000_000,
	)
	require.ErrorContains(t, err, "persisted validator signing state is incomplete")
}

func TestPrepareTestnetConsensusRejectsSourceValidatorKey(t *testing.T) {
	fixture := setupTestnetConsensusFixture(t)
	sourcePV := pvm.NewFilePV(
		fixture.sourcePrivKey,
		fixture.config.PrivValidatorKeyFile(),
		fixture.config.PrivValidatorStateFile(),
	)
	sourcePV.Save()

	err := prepareTestnetConsensus(
		fixture.config,
		"xion-rehearsal-1",
		fixture.height,
		fixture.appHash,
		900_000_000_000_000,
	)
	require.ErrorContains(t, err, "belongs to the source consensus validator set")
}

type testnetConsensusFixture struct {
	config        *cmtcfg.Config
	height        int64
	appHash       []byte
	blockID       cmttypes.BlockID
	sourceChainID string
	sourcePrivKey cmted25519.PrivKey
	localPV       *pvm.FilePV
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
	}
}
