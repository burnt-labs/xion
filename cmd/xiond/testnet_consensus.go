package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	cometdbm "github.com/cometbft/cometbft-db"
	cmtcfg "github.com/cometbft/cometbft/config"
	cmtjson "github.com/cometbft/cometbft/libs/json"
	pvm "github.com/cometbft/cometbft/privval"
	cmtstate "github.com/cometbft/cometbft/proto/tendermint/state"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sm "github.com/cometbft/cometbft/state"
	"github.com/cometbft/cometbft/store"
	cmttypes "github.com/cometbft/cometbft/types"

	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"

	"github.com/burnt-labs/xion/app"
)

const testnetValidatorHistoryRadius int64 = 10

// prepareTestnetConsensus converts the persisted CometBFT state to the local
// validator and chain ID. The application must already be loaded and converted,
// but not committed, before this function is called.
func prepareTestnetConsensus(
	config *cmtcfg.Config,
	chainID string,
	appHeight int64,
	appHash []byte,
	validatorPower int64,
) (returnErr error) {
	if config == nil {
		return errors.New("CometBFT config is required")
	}
	if strings.TrimSpace(chainID) == "" {
		return errors.New("testnet chain ID is required")
	}
	if len(chainID) > cmttypes.MaxChainIDLen {
		return fmt.Errorf("testnet chain ID is too long: got %d bytes, maximum is %d", len(chainID), cmttypes.MaxChainIDLen)
	}
	if appHeight <= 0 {
		return fmt.Errorf("persisted application height must be positive: %d", appHeight)
	}
	if len(appHash) == 0 {
		return errors.New("persisted application hash is required")
	}
	if validatorPower <= 0 {
		return fmt.Errorf("testnet validator power must be positive: %d", validatorPower)
	}
	if validatorPower > cmttypes.MaxTotalVotingPower {
		return fmt.Errorf("testnet validator power %d exceeds maximum %d", validatorPower, cmttypes.MaxTotalVotingPower)
	}

	appGenesis, err := genutiltypes.AppGenesisFromFile(config.GenesisFile())
	if err != nil {
		return fmt.Errorf("read application genesis: %w", err)
	}
	if err := appGenesis.ValidateAndComplete(); err != nil {
		return fmt.Errorf("validate application genesis: %w", err)
	}
	diskGenesis, err := appGenesis.ToGenesisDoc()
	if err != nil {
		return fmt.Errorf("convert application genesis to CometBFT genesis: %w", err)
	}

	stateDB, err := cmtcfg.DefaultDBProvider(&cmtcfg.DBContext{ID: "state", Config: config})
	if err != nil {
		return fmt.Errorf("open CometBFT state database: %w", err)
	}
	stateStore := sm.NewStore(stateDB, sm.StoreOptions{
		DiscardABCIResponses: config.Storage.DiscardABCIResponses,
	})
	defer func() {
		returnErr = errors.Join(returnErr, stateStore.Close())
	}()

	blockStoreDB, err := cmtcfg.DefaultDBProvider(&cmtcfg.DBContext{ID: "blockstore", Config: config})
	if err != nil {
		return fmt.Errorf("open CometBFT block store database: %w", err)
	}
	blockStore := store.NewBlockStore(blockStoreDB)
	defer func() {
		returnErr = errors.Join(returnErr, blockStore.Close())
	}()

	state, err := stateStore.Load()
	if err != nil {
		return fmt.Errorf("load CometBFT state: %w", err)
	}
	if state.LastBlockHeight <= 0 {
		return fmt.Errorf("persisted CometBFT state height must be positive: %d", state.LastBlockHeight)
	}
	if err := app.ValidateTestnetChainID(chainID, state.ChainID); err != nil {
		return err
	}
	if state.LastBlockHeight > appHeight {
		return fmt.Errorf("CometBFT state height %d is ahead of application height %d", state.LastBlockHeight, appHeight)
	}
	if state.LastBlockHeight < appHeight-1 {
		return fmt.Errorf("CometBFT state height %d is more than one block behind application height %d", state.LastBlockHeight, appHeight)
	}
	if blockStore.Height() < appHeight {
		return fmt.Errorf("CometBFT block store height %d is behind application height %d", blockStore.Height(), appHeight)
	}

	block := blockStore.LoadBlock(appHeight)
	if block == nil {
		return fmt.Errorf("load committed block at application height %d", appHeight)
	}
	blockMeta := blockStore.LoadBlockMeta(appHeight)
	if blockMeta == nil {
		return fmt.Errorf("load committed block metadata at application height %d", appHeight)
	}

	cachedGenesis, err := loadCachedTestnetGenesis(stateDB, diskGenesis)
	if err != nil {
		return err
	}
	if cachedGenesis.ChainID != state.ChainID {
		return fmt.Errorf("cached genesis chain ID %q does not match CometBFT state chain ID %q", cachedGenesis.ChainID, state.ChainID)
	}

	privValidator := pvm.LoadFilePV(config.PrivValidatorKeyFile(), config.PrivValidatorStateFile())
	validatorPubKey, err := privValidator.GetPubKey()
	if err != nil {
		return fmt.Errorf("load local validator public key: %w", err)
	}
	validatorAddress := validatorPubKey.Address()
	for _, validators := range []*cmttypes.ValidatorSet{state.LastValidators, state.Validators, state.NextValidators} {
		if validators != nil && validators.HasAddress(validatorAddress) {
			return fmt.Errorf("local validator key %X belongs to the source consensus validator set", validatorAddress)
		}
	}

	lastSignState := privValidator.LastSignState
	hasSignerState := lastSignState.Height != 0 ||
		lastSignState.Round != 0 ||
		lastSignState.Step != 0 ||
		len(lastSignState.SignBytes) != 0 ||
		len(lastSignState.Signature) != 0
	if hasSignerState {
		if len(lastSignState.SignBytes) == 0 || len(lastSignState.Signature) == 0 {
			return errors.New("persisted validator signing state is incomplete")
		}
		if validatorPubKey.VerifySignature(lastSignState.SignBytes, lastSignState.Signature) {
			return errors.New("local validator key already has persisted signing state")
		}
		// The data directory can carry the source node's signing watermark while
		// config contains a fresh fork key. Ignore that foreign watermark only
		// after proving this key is not part of the source consensus validator set.
		privValidator = pvm.LoadFilePVEmptyState(config.PrivValidatorKeyFile(), config.PrivValidatorStateFile())
	}

	if state.LastBlockHeight == appHeight {
		if !bytes.Equal(state.AppHash, appHash) {
			return fmt.Errorf("CometBFT application hash at height %d does not match the loaded application", appHeight)
		}
	} else {
		finalizeResponse, err := stateStore.LoadLastFinalizeBlockResponse(appHeight)
		if err != nil {
			return fmt.Errorf("load finalize-block response at application height %d: %w", appHeight, err)
		}
		state.LastResultsHash = sm.TxResultsHash(finalizeResponse.TxResults)
		if finalizeResponse.ConsensusParamUpdates != nil {
			nextParams := state.ConsensusParams.Update(finalizeResponse.ConsensusParamUpdates)
			if err := nextParams.ValidateBasic(); err != nil {
				return fmt.Errorf("validate consensus parameters at application height %d: %w", appHeight, err)
			}
			if err := state.ConsensusParams.ValidateUpdate(finalizeResponse.ConsensusParamUpdates, appHeight); err != nil {
				return fmt.Errorf("apply consensus parameters at application height %d: %w", appHeight, err)
			}
			state.ConsensusParams = nextParams
			state.Version.Consensus.App = nextParams.Version.App
			state.LastHeightConsensusParamsChanged = appHeight + 1
		}
	}

	newValidator := cmttypes.NewValidator(validatorPubKey, validatorPower)
	newValidatorSet := cmttypes.NewValidatorSet([]*cmttypes.Validator{newValidator})

	state.ChainID = chainID
	state.LastBlockHeight = appHeight
	state.LastBlockID = blockMeta.BlockID
	state.LastBlockTime = block.Time
	state.LastValidators = newValidatorSet.Copy()
	state.Validators = newValidatorSet.Copy()
	state.NextValidators = newValidatorSet.Copy()
	state.LastHeightValidatorsChanged = appHeight
	state.AppHash = bytes.Clone(appHash)

	voteTime := time.Now().UTC()
	if !voteTime.After(block.Time) {
		voteTime = block.Time.Add(time.Nanosecond)
	}
	vote := &cmttypes.Vote{
		Type:             cmtproto.PrecommitType,
		Height:           appHeight,
		Round:            0,
		BlockID:          blockMeta.BlockID,
		Timestamp:        voteTime,
		ValidatorAddress: validatorAddress,
		ValidatorIndex:   0,
	}
	voteProto := vote.ToProto()
	if err := privValidator.SignVote(chainID, voteProto); err != nil {
		return fmt.Errorf("sign synthetic testnet commit: %w", err)
	}
	vote.Signature = voteProto.Signature
	vote.Timestamp = voteProto.Timestamp
	vote.Extension = voteProto.Extension
	vote.ExtensionSignature = voteProto.ExtensionSignature
	seenCommit := &cmttypes.Commit{
		Height:     appHeight,
		Round:      vote.Round,
		BlockID:    blockMeta.BlockID,
		Signatures: []cmttypes.CommitSig{vote.CommitSig()},
	}
	if err := seenCommit.ValidateBasic(); err != nil {
		return fmt.Errorf("validate synthetic testnet commit: %w", err)
	}

	for blockStore.Height() > appHeight {
		if err := blockStore.DeleteLatestBlock(); err != nil {
			return fmt.Errorf("discard block above loaded application height: %w", err)
		}
	}
	if err := blockStoreDB.DeleteSync([]byte(fmt.Sprintf("SC:%d", appHeight+1))); err != nil {
		return fmt.Errorf("clear stale seen commit above application height: %w", err)
	}
	if err := stateStore.Save(state); err != nil {
		return fmt.Errorf("save converted CometBFT state: %w", err)
	}
	if err := saveTestnetValidatorHistory(stateDB, appHeight, newValidatorSet); err != nil {
		return err
	}
	if err := blockStore.SaveSeenCommit(appHeight, seenCommit); err != nil {
		return fmt.Errorf("save synthetic testnet commit: %w", err)
	}
	if state.ConsensusParams.ABCI.VoteExtensionsEnabled(appHeight) {
		extendedCommit := &cmttypes.ExtendedCommit{
			Height:             appHeight,
			Round:              vote.Round,
			BlockID:            blockMeta.BlockID,
			ExtendedSignatures: []cmttypes.ExtendedCommitSig{vote.ExtendedCommitSig()},
		}
		extendedCommitBytes, err := extendedCommit.ToProto().Marshal()
		if err != nil {
			return fmt.Errorf("marshal synthetic testnet extended commit: %w", err)
		}
		if err := blockStoreDB.SetSync([]byte(fmt.Sprintf("EC:%d", appHeight)), extendedCommitBytes); err != nil {
			return fmt.Errorf("save synthetic testnet extended commit: %w", err)
		}
	}

	cachedGenesis.ChainID = chainID
	cachedGenesisBytes, err := cmtjson.Marshal(cachedGenesis)
	if err != nil {
		return fmt.Errorf("marshal cached testnet genesis: %w", err)
	}
	if err := stateDB.SetSync([]byte("genesisDoc"), cachedGenesisBytes); err != nil {
		return fmt.Errorf("save cached testnet genesis: %w", err)
	}

	diskGenesis.ChainID = chainID
	if err := diskGenesis.SaveAs(config.GenesisFile()); err != nil {
		return fmt.Errorf("save canonical testnet genesis: %w", err)
	}
	if err := resetTestnetNetworkFiles(config); err != nil {
		return err
	}

	return nil
}

func loadCachedTestnetGenesis(stateDB cometdbm.DB, fallback *cmttypes.GenesisDoc) (*cmttypes.GenesisDoc, error) {
	genesisBytes, err := stateDB.Get([]byte("genesisDoc"))
	if err != nil {
		return nil, fmt.Errorf("read cached CometBFT genesis: %w", err)
	}
	if len(genesisBytes) == 0 {
		copy := *fallback
		return &copy, nil
	}

	var genesis cmttypes.GenesisDoc
	if err := cmtjson.Unmarshal(genesisBytes, &genesis); err != nil {
		return nil, fmt.Errorf("decode cached CometBFT genesis: %w", err)
	}
	if err := genesis.ValidateAndComplete(); err != nil {
		return nil, fmt.Errorf("validate cached CometBFT genesis: %w", err)
	}
	return &genesis, nil
}

func saveTestnetValidatorHistory(stateDB cometdbm.DB, height int64, validators *cmttypes.ValidatorSet) error {
	validatorSet, err := validators.ToProto()
	if err != nil {
		return fmt.Errorf("encode testnet validator set: %w", err)
	}
	batch := stateDB.NewBatch()
	defer func() {
		_ = batch.Close()
	}()
	for offset := -testnetValidatorHistoryRadius; offset <= testnetValidatorHistoryRadius; offset++ {
		historyHeight := height + offset
		if historyHeight <= 0 {
			continue
		}
		lastHeightChanged := height
		if historyHeight < height {
			lastHeightChanged = historyHeight
		}
		validatorInfo := &cmtstate.ValidatorsInfo{
			ValidatorSet:      validatorSet,
			LastHeightChanged: lastHeightChanged,
		}
		validatorInfoBytes, err := validatorInfo.Marshal()
		if err != nil {
			return fmt.Errorf("marshal testnet validator history at height %d: %w", historyHeight, err)
		}
		if err := batch.Set([]byte(fmt.Sprintf("validatorsKey:%d", historyHeight)), validatorInfoBytes); err != nil {
			return fmt.Errorf("stage testnet validator history at height %d: %w", historyHeight, err)
		}
	}
	if err := batch.WriteSync(); err != nil {
		return fmt.Errorf("save testnet validator history: %w", err)
	}
	return nil
}

func resetTestnetNetworkFiles(config *cmtcfg.Config) error {
	walPath := config.Consensus.WalFile()
	walDir, walBase := filepath.Dir(walPath), filepath.Base(walPath)
	entries, err := os.ReadDir(walDir)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read consensus WAL directory: %w", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if name != walBase {
			suffix, ok := strings.CutPrefix(name, walBase+".")
			if !ok || len(suffix) < 3 || strings.Trim(suffix, "0123456789") != "" {
				continue
			}
		}
		if entry.IsDir() {
			return fmt.Errorf("consensus WAL path is a directory: %s", filepath.Join(walDir, name))
		}
		if err := os.Remove(filepath.Join(walDir, name)); err != nil {
			return fmt.Errorf("remove source consensus WAL file %s: %w", name, err)
		}
	}

	addrBookPath := config.P2P.AddrBookFile()
	if err := os.Remove(addrBookPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove source address book: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(addrBookPath), 0o700); err != nil {
		return fmt.Errorf("create testnet address book directory: %w", err)
	}
	if err := os.WriteFile(addrBookPath, []byte("{}"), 0o600); err != nil {
		return fmt.Errorf("reset testnet address book: %w", err)
	}
	return nil
}
