package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	cmtcfg "github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/privval"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	dbm "github.com/cosmos/cosmos-db"

	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/app"
)

// inPlaceTestnetCmd uses normal SDK startup with chain-owned application and
// consensus conversion. Normal start never invokes this creator.
func inPlaceTestnetCmd() *cobra.Command {
	var config *cmtcfg.Config
	var chainID, operator, upgradeName string
	creator := func(logger log.Logger, db dbm.DB, traceStore io.Writer, appOpts servertypes.AppOptions) servertypes.Application {
		wasmApp := newApp(logger, db, traceStore, appOpts).(*app.WasmApp)
		height, appHash := wasmApp.LastBlockHeight(), wasmApp.LastCommitID().Hash
		if height <= 0 {
			panic("in-place-testnet requires persisted blocks; start the source chain first")
		}
		validator := privval.LoadOrGenFilePV(config.PrivValidatorKeyFile(), config.PrivValidatorStateFile())
		pubKey, err := validator.GetPubKey()
		if err != nil {
			panic(fmt.Errorf("load local testnet validator key: %w", err))
		}
		if err := wasmApp.InitForTestnet(pubKey, operator, chainID, upgradeName); err != nil {
			panic(fmt.Errorf("initialize application for in-place testnet: %w", err))
		}
		ctx := wasmApp.NewUncachedContext(false, tmproto.Header{ChainID: chainID, Height: height})
		localValidator, err := wasmApp.StakingKeeper.GetValidatorByConsAddr(ctx, sdk.ConsAddress(pubKey.Address()))
		if err != nil {
			panic(fmt.Errorf("read replacement testnet validator: %w", err))
		}
		power := localValidator.ConsensusPower(wasmApp.StakingKeeper.PowerReduction(ctx))
		if err := prepareTestnetConsensus(config, chainID, height, appHash, power); err != nil {
			panic(fmt.Errorf("initialize consensus for in-place testnet: %w", err))
		}
		return wasmApp
	}
	cmd := server.StartCmd(creator, app.DefaultNodeHome)
	start := cmd.RunE
	cmd.Use = "in-place-testnet [newChainID] [newOperatorAddress]"
	cmd.Short = "Convert disposable persisted state into a local testnet and start it"
	cmd.Long = "Replace application and consensus validator state on a disposable copy, change its chain ID, and start a local testnet. Use fresh local keys. Resume with normal start; do not convert the same home twice."
	cmd.Args = cobra.ExactArgs(2)
	cmd.Flags().Bool("skip-confirmation", false, "Skip confirmation of disposable-state conversion")
	cmd.Flags().String(server.KeyTriggerTestnetUpgrade, "", "Optionally schedule an upgrade ten blocks after the loaded height")
	addModuleInitFlags(cmd)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		withComet, err := cmd.Flags().GetBool("with-comet")
		if err != nil {
			return err
		}
		if !withComet {
			return fmt.Errorf("in-place-testnet requires in-process CometBFT")
		}
		skip, err := cmd.Flags().GetBool("skip-confirmation")
		if err != nil {
			return err
		}
		// Refuse public network chain IDs before anything in the home is opened;
		// the source chain ID is checked against persisted state during conversion.
		if err := app.ValidateTestnetChainID(args[0], ""); err != nil {
			return err
		}
		if !skip {
			cmd.PrintErrln("This irreversibly changes the selected home. Use only a disposable copy with fresh keys. Continue? (y/N)")
			answer, err := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
			if err != nil {
				return fmt.Errorf("read conversion confirmation: %w", err)
			}
			answer = strings.ToLower(strings.TrimSpace(answer))
			if answer != "y" && answer != "yes" {
				cmd.PrintErrln("Conversion cancelled.")
				return nil
			}
		}
		serverCtx := server.GetServerContextFromCmd(cmd)
		config = serverCtx.Config
		chainID, operator = args[0], args[1]
		upgradeName, err = cmd.Flags().GetString(server.KeyTriggerTestnetUpgrade)
		if err != nil {
			return err
		}
		// Conversion is implemented here, not by the SDK's testnetify path.
		serverCtx.Viper.Set(server.KeyIsTestnet, false)
		serverCtx.Viper.Set(flags.FlagChainID, chainID)
		return start(cmd, nil)
	}
	return cmd
}
