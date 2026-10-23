package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtcfg "github.com/cometbft/cometbft/config"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sm "github.com/cometbft/cometbft/state"
	"github.com/cometbft/cometbft/store"
	cmttypes "github.com/cometbft/cometbft/types"
)

func TestPrepareTestnetConsensusRestoresExtendedCommit(t *testing.T) {
	fixture := setupTestnetConsensusFixture(t)
	stateDB, err := cmtcfg.DefaultDBProvider(&cmtcfg.DBContext{ID: "state", Config: fixture.config})
	require.NoError(t, err)
	stateStore := sm.NewStore(stateDB, sm.StoreOptions{})
	state, err := stateStore.Load()
	require.NoError(t, err)
	state.ConsensusParams.ABCI.VoteExtensionsEnableHeight = fixture.height
	require.NoError(t, stateStore.Save(state))
	require.NoError(t, stateStore.Close())

	chainID := "extended-testnet"
	require.NoError(t, prepareTestnetConsensus(fixture.config, chainID, fixture.height, fixture.appHash, 100, fixture.newKey))
	blockDB, err := cmtcfg.DefaultDBProvider(&cmtcfg.DBContext{ID: "blockstore", Config: fixture.config})
	require.NoError(t, err)
	blockStore := store.NewBlockStore(blockDB)
	defer func() { require.NoError(t, blockStore.Close()) }()
	extendedCommit := blockStore.LoadBlockExtendedCommit(fixture.height)
	require.NotNil(t, extendedCommit)
	require.Equal(t, fixture.blockID, extendedCommit.BlockID)
	require.Len(t, extendedCommit.ExtendedSignatures, 1)
	vote := extendedCommit.GetExtendedVote(0)
	pubKey := fixture.newKey.PubKey()
	require.True(t, pubKey.VerifySignature(cmttypes.VoteSignBytes(chainID, vote.ToProto()), vote.Signature))
	require.True(t, pubKey.VerifySignature(cmttypes.VoteExtensionSignBytes(chainID, vote.ToProto()), vote.ExtensionSignature))
}

func TestPrepareTestnetConsensusRecoversDiscardedResponses(t *testing.T) {
	fixture := setupTestnetConsensusFixture(t)
	fixture.config.Storage.DiscardABCIResponses = true
	stateDB, err := cmtcfg.DefaultDBProvider(&cmtcfg.DBContext{ID: "state", Config: fixture.config})
	require.NoError(t, err)
	stateStore := sm.NewStore(stateDB, sm.StoreOptions{DiscardABCIResponses: true})
	state, err := stateStore.Load()
	require.NoError(t, err)
	blockDB, err := cmtcfg.DefaultDBProvider(&cmtcfg.DBContext{ID: "blockstore", Config: fixture.config})
	require.NoError(t, err)
	blockStore := store.NewBlockStore(blockDB)
	previousCommit := &cmttypes.Commit{Height: fixture.height, BlockID: fixture.blockID, Signatures: []cmttypes.CommitSig{cmttypes.NewCommitSigAbsent()}}
	block := cmttypes.MakeBlock(fixture.height+1, nil, previousCommit, nil)
	block.Header.ChainID = fixture.sourceChainID
	block.Header.Time = state.LastBlockTime.Add(time.Second)
	block.Header.LastBlockID = fixture.blockID
	block.Header.ValidatorsHash = state.Validators.Hash()
	block.Header.ProposerAddress = fixture.sourcePrivKey.PubKey().Address()
	require.NoError(t, block.ValidateBasic())
	parts, err := block.MakePartSet(cmttypes.BlockPartSizeBytes)
	require.NoError(t, err)
	blockID := cmttypes.BlockID{Hash: block.Hash(), PartSetHeader: parts.Header()}
	blockStore.SaveBlock(block, parts, &cmttypes.Commit{Height: block.Height, BlockID: blockID, Signatures: []cmttypes.CommitSig{cmttypes.NewCommitSigAbsent()}})
	require.NoError(t, blockStore.Close())

	appHash := bytes.Repeat([]byte{0xCD}, 32)
	response := &abci.ResponseFinalizeBlock{
		AppHash:               appHash,
		TxResults:             []*abci.ExecTxResult{{Code: 1, GasUsed: 123}},
		ConsensusParamUpdates: &cmtproto.ConsensusParams{Block: &cmtproto.BlockParams{MaxBytes: state.ConsensusParams.Block.MaxBytes, MaxGas: 123456}},
	}
	require.NoError(t, stateStore.SaveFinalizeBlockResponse(block.Height, response))
	require.NoError(t, stateStore.Close())

	require.NoError(t, prepareTestnetConsensus(fixture.config, "recovered-testnet", block.Height, appHash, 100, fixture.newKey))
	stateDB, err = cmtcfg.DefaultDBProvider(&cmtcfg.DBContext{ID: "state", Config: fixture.config})
	require.NoError(t, err)
	stateStore = sm.NewStore(stateDB, sm.StoreOptions{DiscardABCIResponses: true})
	defer func() { require.NoError(t, stateStore.Close()) }()
	recovered, err := stateStore.Load()
	require.NoError(t, err)
	require.Equal(t, block.Height, recovered.LastBlockHeight)
	require.Equal(t, blockID, recovered.LastBlockID)
	require.Equal(t, appHash, recovered.AppHash)
	require.Equal(t, sm.TxResultsHash(response.TxResults), recovered.LastResultsHash)
	require.Equal(t, int64(123456), recovered.ConsensusParams.Block.MaxGas)
}

func TestResetTestnetNetworkFilesPreservesUnrelatedPaths(t *testing.T) {
	config := cmtcfg.DefaultConfig().SetRoot(filepath.Join(t.TempDir(), "node[1]"))
	walPath := config.Consensus.WalFile()
	require.NoError(t, os.MkdirAll(filepath.Dir(walPath), 0o700))
	for _, suffix := range []string{"", ".000", ".1000"} {
		require.NoError(t, os.WriteFile(walPath+suffix, []byte("source WAL"), 0o600))
	}
	wallet := filepath.Join(walPath+"let", "keys")
	require.NoError(t, os.MkdirAll(filepath.Dir(wallet), 0o700))
	require.NoError(t, os.WriteFile(wallet, []byte("keep wallet"), 0o600))
	require.NoError(t, os.WriteFile(walPath+".backup", []byte("keep backup"), 0o600))

	require.NoError(t, resetTestnetNetworkFiles(config))
	for _, suffix := range []string{"", ".000", ".1000"} {
		_, err := os.Stat(walPath + suffix)
		require.True(t, os.IsNotExist(err))
	}
	contents, err := os.ReadFile(wallet)
	require.NoError(t, err)
	require.Equal(t, []byte("keep wallet"), contents)
	contents, err = os.ReadFile(walPath + ".backup")
	require.NoError(t, err)
	require.Equal(t, []byte("keep backup"), contents)
}
