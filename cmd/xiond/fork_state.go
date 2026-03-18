package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cosmos/gogoproto/proto"
	"github.com/spf13/cobra"

	cmtcfg "github.com/cometbft/cometbft/config"
	cmtjson "github.com/cometbft/cometbft/libs/json"
	"github.com/cometbft/cometbft/node"
	cmtstate "github.com/cometbft/cometbft/proto/tendermint/state"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	pvm "github.com/cometbft/cometbft/privval"
	sm "github.com/cometbft/cometbft/state"
	"github.com/cometbft/cometbft/store"
	cmttypes "github.com/cometbft/cometbft/types"

	dbm "github.com/cosmos/cosmos-db"

	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/server"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"

	"github.com/burnt-labs/xion/app"
)

// forkStateCmd modifies app + CometBFT state to create a single-validator fork, then exits.
// Use `xiond start` afterwards.
func forkStateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fork-state [new-chain-id] [new-operator-address]",
		Short: "Modify state to create a single-validator fork, then exit",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			newChainID := args[0]
			newOperatorAddress := args[1]

			serverCtx := server.GetServerContextFromCmd(cmd)
			cfg := serverCtx.Config
			logger := log.NewLogger(os.Stdout)

			// ── 1. Patch genesis file chain ID ──
			genFilePath := cfg.GenesisFile()
			appGen, err := genutiltypes.AppGenesisFromFile(genFilePath)
			if err != nil {
				return fmt.Errorf("reading genesis: %w", err)
			}
			appGen.ChainID = newChainID
			if err := appGen.ValidateAndComplete(); err != nil {
				return fmt.Errorf("validating genesis: %w", err)
			}
			if err := appGen.SaveAs(genFilePath); err != nil {
				return fmt.Errorf("saving genesis: %w", err)
			}
			logger.Info("patched genesis chain ID", "chain_id", newChainID)

			// Clear addrbook
			_ = os.Remove(cfg.RootDir + "/config/addrbook.json")
			_ = os.WriteFile(cfg.RootDir+"/config/addrbook.json", []byte("{}"), 0o600)

			// ── 2. Open CometBFT databases ──
			blockStoreDB, err := cmtcfg.DefaultDBProvider(&cmtcfg.DBContext{ID: "blockstore", Config: cfg})
			if err != nil {
				return fmt.Errorf("opening blockstore: %w", err)
			}
			defer blockStoreDB.Close()
			blockStore := store.NewBlockStore(blockStoreDB)

			stateDB, err := cmtcfg.DefaultDBProvider(&cmtcfg.DBContext{ID: "state", Config: cfg})
			if err != nil {
				return fmt.Errorf("opening state db: %w", err)
			}
			defer stateDB.Close()

			// ── 3. Load validator key ──
			privValidator := pvm.LoadOrGenFilePV(cfg.PrivValidatorKeyFile(), cfg.PrivValidatorStateFile())
			userPubKey, err := privValidator.GetPubKey()
			if err != nil {
				return fmt.Errorf("getting pub key: %w", err)
			}
			validatorAddress := userPubKey.Address()
			logger.Info("using validator", "address", validatorAddress, "pubkey", userPubKey)

			// ── 4. Load CometBFT state ──
			// Pre-patch genesisDoc chain ID in state.db so LoadStateFromDBOrGenesisDocProvider
			// reads the correct chain ID (avoids genesis JSON integer parsing issues entirely)
			if gdBytes, err := stateDB.Get([]byte("genesisDoc")); err == nil && gdBytes != nil {
				var gd cmttypes.GenesisDoc
				if err := cmtjson.Unmarshal(gdBytes, &gd); err == nil {
					gd.ChainID = newChainID
					if b, err := cmtjson.Marshal(gd); err == nil {
						_ = stateDB.SetSync([]byte("genesisDoc"), b)
						logger.Info("pre-patched genesisDoc chain ID in state.db")
					}
				}
			}

			genDocProvider := node.DefaultGenesisDocProviderFunc(cfg)
			state, genDoc, err := node.LoadStateFromDBOrGenesisDocProvider(stateDB, genDocProvider)
			if err != nil {
				return fmt.Errorf("loading state: %w", err)
			}

			// ── 5. App state modifications ──
			// App state (new validator, funding, gov, wasm) is NOT modified here.
			// Instead, the newTestnetApp function in root.go checks for KeyIsTestnet
			// and calls InitXionAppForTestnet when the node starts. The app changes
			// are committed during the first block's Commit.
			//
			// We set a marker file so the spawner knows to pass testnet flags to xiond start.
			markerPath := filepath.Join(cfg.RootDir, "testnet_params.json")
			markerData := fmt.Sprintf(`{"operator":"%s","validator":"%x","pubkey":"%x"}`,
				newOperatorAddress, validatorAddress, userPubKey.Bytes())
			if err := os.WriteFile(markerPath, []byte(markerData), 0o644); err != nil {
				return fmt.Errorf("writing testnet params: %w", err)
			}

			// ── 6. Modify CometBFT state ──
			stateStore := sm.NewStore(stateDB, sm.StoreOptions{
				DiscardABCIResponses: cfg.Storage.DiscardABCIResponses,
			})

			// Load block
			block := blockStore.LoadBlock(blockStore.Height())
			if block == nil {
				return fmt.Errorf("block at height %d not found", blockStore.Height())
			}

			// Delete next seen commit
			_ = blockStoreDB.Delete(fmt.Appendf(nil, "SC:%v", blockStore.Height()+1))

			// Create new validator set first (needed for block header hashes)
			newVal := &cmttypes.Validator{
				Address:     validatorAddress,
				PubKey:      userPubKey,
				VotingPower: 900000000000000,
			}
			newValSet := &cmttypes.ValidatorSet{
				Validators: []*cmttypes.Validator{newVal},
				Proposer:   newVal,
			}

			// Get the actual app hash by opening the app briefly
			appDB, err := dbm.NewDB("application", dbm.BackendType(cfg.DBBackend), cfg.RootDir+"/data")
			if err != nil {
				return fmt.Errorf("opening application db: %w", err)
			}
			wasmApp := app.NewWasmApp(logger, appDB, nil, true, serverCtx.Viper, nil)
			appHash := wasmApp.LastCommitID().Hash
			appDB.Close()
			logger.Info("read app hash", "hash", fmt.Sprintf("%X", appHash))

			// Update chain ID and block header. Set state.LastBlockHeight = height
			// so no block replay happens (avoids commit validation issues).
			block.ChainID = newChainID
			state.ChainID = newChainID
			state.AppHash = appHash
			block.LastBlockID = state.LastBlockID
			block.LastCommit.BlockID = state.LastBlockID
			block.ValidatorsHash = newValSet.Hash()
			block.NextValidatorsHash = newValSet.Hash()
			block.ProposerAddress = validatorAddress

			// Re-serialize and save the modified block back to the blockstore
			blockParts, err := block.MakePartSet(cmttypes.BlockPartSizeBytes)
			if err != nil {
				return fmt.Errorf("making block parts: %w", err)
			}

			// Sign a vote with our validator at the blockstore height
			height := blockStore.Height()
			vote := cmttypes.Vote{
				Type:             cmtproto.PrecommitType,
				Height:           height,
				Round:            0,
				BlockID:          state.LastBlockID,
				Timestamp:        time.Now(),
				ValidatorAddress: validatorAddress,
				ValidatorIndex:   0,
				Signature:        []byte{},
			}
			voteProto := vote.ToProto()
			if err := privValidator.SignVote(newChainID, voteProto); err != nil {
				return fmt.Errorf("signing vote: %w", err)
			}
			vote.Signature = voteProto.Signature
			vote.Timestamp = voteProto.Timestamp

			// Replace block's last commit
			block.LastCommit.Signatures[0].ValidatorAddress = validatorAddress
			block.LastCommit.Signatures[0].Signature = vote.Signature
			block.LastCommit.Signatures = []cmttypes.CommitSig{block.LastCommit.Signatures[0]}

			// Replace seen commit — use blockStore.Height() since our minimal checkpoint
			// stores the seen commit at the checkpoint height, not state.LastBlockHeight
			seenCommit := blockStore.LoadSeenCommit(blockStore.Height())
			if seenCommit == nil {
				return fmt.Errorf("seen commit at height %d not found", blockStore.Height())
			}
			seenCommit.BlockID = state.LastBlockID
			seenCommit.Round = vote.Round
			seenCommit.Signatures[0].Signature = vote.Signature
			seenCommit.Signatures[0].ValidatorAddress = validatorAddress
			seenCommit.Signatures[0].Timestamp = vote.Timestamp
			seenCommit.Signatures = []cmttypes.CommitSig{seenCommit.Signatures[0]}
			if err := blockStore.SaveSeenCommit(height, seenCommit); err != nil {
				return fmt.Errorf("saving seen commit: %w", err)
			}

			// Save the modified block directly to the DB (SaveBlock only allows contiguous heights)
			// Block parts
			for i := 0; i < int(blockParts.Total()); i++ {
				part := blockParts.GetPart(i)
				pbPart, err := part.ToProto()
				if err != nil {
					return fmt.Errorf("converting part %d to proto: %w", i, err)
				}
				partBytes, err := proto.Marshal(pbPart)
				if err != nil {
					return fmt.Errorf("marshaling block part %d: %w", i, err)
				}
				if err := blockStoreDB.Set(fmt.Appendf(nil, "P:%v:%v", height, i), partBytes); err != nil {
					return fmt.Errorf("saving block part %d: %w", i, err)
				}
			}
			// Block meta
			blockMeta := cmttypes.NewBlockMeta(block, blockParts)
			pbMeta := blockMeta.ToProto()
			metaBytes, err := proto.Marshal(pbMeta)
			if err != nil {
				return fmt.Errorf("marshaling block meta: %w", err)
			}
			if err := blockStoreDB.Set(fmt.Appendf(nil, "H:%v", height), metaBytes); err != nil {
				return fmt.Errorf("saving block meta: %w", err)
			}
			// Commit
			pbCommit := seenCommit.ToProto()
			commitBytes, err := proto.Marshal(pbCommit)
			if err != nil {
				return fmt.Errorf("marshaling commit: %w", err)
			}
			if err := blockStoreDB.Set(fmt.Appendf(nil, "C:%v", height), commitBytes); err != nil {
				return fmt.Errorf("saving commit: %w", err)
			}

			// Set validator set in state
			state.Validators = newValSet
			state.LastValidators = newValSet
			state.NextValidators = newValSet
			state.LastBlockHeight = height
			state.LastHeightValidatorsChanged = height

			if err := stateStore.Save(state); err != nil {
				return fmt.Errorf("saving state: %w", err)
			}

			// Write validator info entries
			valSet, err := state.Validators.ToProto()
			if err != nil {
				return fmt.Errorf("converting validator set: %w", err)
			}
			valInfo := &cmtstate.ValidatorsInfo{
				ValidatorSet:      valSet,
				LastHeightChanged: state.LastBlockHeight,
			}
			buf, err := valInfo.Marshal()
			if err != nil {
				return fmt.Errorf("marshaling validator info: %w", err)
			}
			for _, h := range []int64{blockStore.Height() - 1, blockStore.Height(), blockStore.Height() + 1} {
				if err := stateDB.Set(fmt.Appendf(nil, "validatorsKey:%v", h), buf); err != nil {
					return fmt.Errorf("writing validators at %d: %w", h, err)
				}
			}

			// Save genesis doc with correct chain ID
			genDoc.ChainID = newChainID
			genDocBytes, err := cmtjson.Marshal(genDoc)
			if err != nil {
				return fmt.Errorf("marshaling genesis doc: %w", err)
			}
			if err := stateDB.SetSync([]byte("genesisDoc"), genDocBytes); err != nil {
				return fmt.Errorf("saving genesis doc: %w", err)
			}

			logger.Info("fork-state complete",
				"chain_id", newChainID,
				"operator", newOperatorAddress,
				"height", blockStore.Height(),
				"validator", validatorAddress,
			)
			fmt.Println("Done. Run 'xiond start' to start the fork.")
			return nil
		},
	}

	// Add flags that the server context needs
	cmd.Flags().String("db_backend", "rocksdb", "database backend")
	return cmd
}
