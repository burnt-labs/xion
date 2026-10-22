package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	cmtcfg "github.com/cometbft/cometbft/config"
	cmted25519 "github.com/cometbft/cometbft/crypto/ed25519"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	dbm "github.com/cosmos/cosmos-db"

	"cosmossdk.io/log/v2"

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
	// validatorKey is generated for every conversion. The home's existing
	// validator key is never used, so the copy cannot sign with a live key.
	var validatorKey cmted25519.PrivKey
	creator := func(logger log.Logger, db dbm.DB, appOpts servertypes.AppOptions) servertypes.Application {
		wasmApp := newApp(logger, db, appOpts).(*app.WasmApp)
		height, appHash := wasmApp.LastBlockHeight(), wasmApp.LastCommitID().Hash
		if height <= 0 {
			panic("in-place-testnet requires persisted blocks; start the source chain first")
		}
		pubKey := validatorKey.PubKey()
		if err := wasmApp.InitForTestnet(pubKey, operator, chainID, upgradeName); err != nil {
			panic(fmt.Errorf("initialize application for in-place testnet: %w", err))
		}
		ctx := wasmApp.NewUncachedContext(false, tmproto.Header{ChainID: chainID, Height: height})
		localValidator, err := wasmApp.StakingKeeper.GetValidatorByConsAddr(ctx, sdk.ConsAddress(pubKey.Address()))
		if err != nil {
			panic(fmt.Errorf("read replacement testnet validator: %w", err))
		}
		power := localValidator.ConsensusPower(wasmApp.StakingKeeper.PowerReduction(ctx))
		if err := prepareTestnetConsensus(config, chainID, height, appHash, power, validatorKey); err != nil {
			panic(fmt.Errorf("initialize consensus for in-place testnet: %w", err))
		}
		return wasmApp
	}
	cmd := server.StartCmd(creator, app.DefaultNodeHome)
	start := cmd.RunE
	cmd.Use = "in-place-testnet [newChainID] [newOperatorAddress]"
	cmd.Short = "Convert disposable persisted state into a local testnet and start it"
	cmd.Long = "Replace application and consensus validator state on a disposable copy, change its chain ID, and start a local testnet with a freshly generated validator key. Resume with normal start; do not convert the same home twice."
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
		serverCtx := server.GetServerContextFromCmd(cmd)
		sourceChainID, err := checkInPlaceTestnetTarget(cmd, serverCtx.Config, args[0])
		if err != nil {
			return err
		}
		// Generated in memory only; it is written to the home after the last
		// conversion check passes.
		newKey := cmted25519.GenPrivKey()
		cmd.PrintErrf("Converting home %s from source chain %s to new chain %s.\n", serverCtx.Config.RootDir, sourceChainID, args[0])
		cmd.PrintErrf("New validator consensus address: %s (fresh key; the home's existing validator key is moved aside and not used).\n", sdk.ConsAddress(newKey.PubKey().Address()))
		if !skip {
			cmd.PrintErrln("This irreversibly changes the selected home. Use only a disposable copy. Continue? (y/N)")
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
		config = serverCtx.Config
		validatorKey = newKey
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

// checkInPlaceTestnetTarget refuses a home that is not clearly a disposable
// copy, or a replacement chain ID that is unsafe for it, before anything in the
// home is opened. It returns the source chain ID from the home's genesis file.
// The key guards cannot recognize a remote-signer validator or a sentry, whose
// config/ holds an unused stub key, so an explicit --home is required and a
// remote-signer configuration is refused outright.
func checkInPlaceTestnetTarget(cmd *cobra.Command, config *cmtcfg.Config, chainID string) (string, error) {
	if !cmd.Flags().Changed(flags.FlagHome) {
		return "", errors.New("in-place-testnet requires an explicit --home naming the disposable copy to convert")
	}
	if config == nil {
		return "", errors.New("CometBFT config is required")
	}
	if laddr := strings.TrimSpace(config.PrivValidatorListenAddr); laddr != "" {
		return "", fmt.Errorf("refusing to convert a home that uses a remote signer (priv_validator_laddr = %q); convert a disposable copy with priv_validator_laddr cleared and a fresh local key", laddr)
	}
	sourceChainID, err := readGenesisChainID(config.GenesisFile())
	if err != nil {
		return "", err
	}
	if err := app.ValidateTestnetChainID(chainID, sourceChainID); err != nil {
		return "", err
	}
	return sourceChainID, nil
}

// readGenesisChainID streams the top-level chain_id out of a genesis file
// without loading the application state.
func readGenesisChainID(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("read source genesis: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()
	decoder := json.NewDecoder(bufio.NewReader(file))
	if token, err := decoder.Token(); err != nil || token != json.Delim('{') {
		return "", fmt.Errorf("read source genesis %s: not a JSON object", path)
	}
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return "", fmt.Errorf("read source genesis %s: %w", path, err)
		}
		if key, _ := token.(string); key == "chain_id" {
			var chainID string
			if err := decoder.Decode(&chainID); err != nil {
				return "", fmt.Errorf("read source genesis %s chain_id: %w", path, err)
			}
			if strings.TrimSpace(chainID) == "" {
				break
			}
			return chainID, nil
		}
		var skipped json.RawMessage
		if err := decoder.Decode(&skipped); err != nil {
			return "", fmt.Errorf("read source genesis %s: %w", path, err)
		}
	}
	return "", fmt.Errorf("source genesis %s has no chain_id", path)
}
