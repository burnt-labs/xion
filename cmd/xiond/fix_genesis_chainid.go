package main

import (
	"fmt"

	cmtjson "github.com/cometbft/cometbft/libs/json"
	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/linxGnu/grocksdb"
	"github.com/spf13/cobra"

	"github.com/burnt-labs/xion/app"
)

// fixGenesisChainIDCmd patches the chain ID stored in CometBFT's state.db genesisDoc.
// This works around a cosmos-sdk bug where in-place-testnet saves the genesis doc
// back to state.db without updating its ChainID field.
func fixGenesisChainIDCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fix-genesis-chainid [chain-id]",
		Short: "Patch the chain ID in state.db's genesisDoc",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			newChainID := args[0]

			home, _ := cmd.Flags().GetString("home")
			if home == "" {
				home = app.DefaultNodeHome
			}

			stateDBPath := home + "/data/state.db"

			opts := grocksdb.NewDefaultOptions()
			db, err := grocksdb.OpenDb(opts, stateDBPath)
			if err != nil {
				return fmt.Errorf("opening state.db: %w", err)
			}
			defer db.Close()

			readOpts := grocksdb.NewDefaultReadOptions()
			val, err := db.Get(readOpts, []byte("genesisDoc"))
			if err != nil {
				return fmt.Errorf("reading genesisDoc: %w", err)
			}
			defer val.Free()

			if val.Data() == nil {
				return fmt.Errorf("genesisDoc not found in state.db")
			}

			var genDoc cmttypes.GenesisDoc
			if err := cmtjson.Unmarshal(val.Data(), &genDoc); err != nil {
				return fmt.Errorf("unmarshaling genesisDoc: %w", err)
			}

			if genDoc.ChainID == newChainID {
				fmt.Printf("genesisDoc already has chain ID %q\n", newChainID)
				return nil
			}

			fmt.Printf("patching genesisDoc chain ID: %q -> %q\n", genDoc.ChainID, newChainID)
			genDoc.ChainID = newChainID

			b, err := cmtjson.Marshal(genDoc)
			if err != nil {
				return fmt.Errorf("marshaling genesisDoc: %w", err)
			}

			writeOpts := grocksdb.NewDefaultWriteOptions()
			if err := db.Put(writeOpts, []byte("genesisDoc"), b); err != nil {
				return fmt.Errorf("writing genesisDoc: %w", err)
			}

			fmt.Println("done")
			return nil
		},
	}
}
