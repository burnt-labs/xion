package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	cmtcfg "github.com/cometbft/cometbft/config"
	cmtcrypto "github.com/cometbft/cometbft/crypto"
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
// testnetGRPCOnlyFlag is the start command's flag for running without
// consensus (the SDK does not export its name).
const testnetGRPCOnlyFlag = "grpc-only"

func inPlaceTestnetCmd() *cobra.Command {
	var config *cmtcfg.Config
	var chainID, operator, upgradeName string
	// validatorKey is generated for every conversion. The home's existing
	// validator key is never used, so the copy cannot sign with a live key. The
	// one exception is a rerun on a home whose earlier conversion was
	// interrupted: it keeps the key that conversion installed.
	var validatorKey cmtcrypto.PrivKey
	creator := func(logger log.Logger, db dbm.DB, appOpts servertypes.AppOptions) servertypes.Application {
		wasmApp := newApp(logger, db, appOpts).(*app.WasmApp)
		height, appHash := wasmApp.LastBlockHeight(), wasmApp.LastCommitID().Hash
		if height <= 0 {
			panic("in-place-testnet requires persisted blocks; start the source chain first")
		}
		installedKey, err := installedTestnetValidatorKey(config, app.TestnetValidatorPower)
		if err != nil {
			panic(fmt.Errorf("check for an interrupted in-place testnet conversion: %w", err))
		}
		if installedKey != nil {
			validatorKey = installedKey
			logger.Info("finishing an interrupted in-place testnet conversion with the validator key it installed",
				"consensus_address", sdk.ConsAddress(installedKey.PubKey().Address()).String())
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
	cmd.Long = "Replace application and consensus validator state on a disposable copy, change its chain ID, and start a local testnet with a freshly generated validator key. Use an explicit --home; homes that use a remote signer are refused. Resume with normal start; do not convert the same home twice. If a conversion exits before the first new block commits, rerun the same command on the same home to finish it."
	// Record whether --home was given on the command line here: cobra validates
	// arguments after parsing flags but before the root pre-run, which applies
	// XIOND_HOME and config values through Flags().Set and so marks --home as
	// changed even when it was not typed.
	var homeOnCommandLine bool
	cmd.Args = func(cmd *cobra.Command, args []string) error {
		homeOnCommandLine = cmd.Flags().Changed(flags.FlagHome)
		return cobra.ExactArgs(2)(cmd, args)
	}
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
		// In gRPC-only mode no block is ever produced, so the application half
		// of the conversion, which commits with the first new block, never lands.
		grpcOnly, err := cmd.Flags().GetBool(testnetGRPCOnlyFlag)
		if err != nil {
			return err
		}
		if grpcOnly || serverCtx.Viper.GetBool(testnetGRPCOnlyFlag) {
			return fmt.Errorf("in-place-testnet cannot run with --%s: the conversion only completes when the first new block commits", testnetGRPCOnlyFlag)
		}
		sourceChainID, resuming, err := checkInPlaceTestnetTarget(homeOnCommandLine, serverCtx.Config, args[0])
		if err != nil {
			return err
		}
		// Generated in memory only; it is written to the home after the last
		// conversion check passes.
		newKey := cmted25519.GenPrivKey()
		if resuming {
			cmd.PrintErrf("Home %s already has chain ID %s in its genesis file: finishing the interrupted conversion to it, with the validator key that conversion installed.\n", serverCtx.Config.RootDir, args[0])
		} else {
			cmd.PrintErrf("Converting home %s from source chain %s to new chain %s.\n", serverCtx.Config.RootDir, sourceChainID, args[0])
			cmd.PrintErrf("New validator consensus address: %s (fresh key; the home's existing validator key is moved aside and not used).\n", sdk.ConsAddress(newKey.PubKey().Address()))
		}
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
// home is opened. It returns the source chain ID from the home's genesis file,
// and whether the home looks like an interrupted conversion to chainID: its
// genesis already carries chainID and a conversion has installed a validator
// key. prepareTestnetConsensus then decides from the persisted consensus state
// whether that conversion can be finished, and refuses the home otherwise.
// The key guards cannot recognize a remote-signer validator or a sentry, whose
// config/ holds an unused stub key, so an explicit --home on the command line
// is required (XIOND_HOME does not count) and a remote-signer configuration is
// refused outright.
func checkInPlaceTestnetTarget(homeOnCommandLine bool, config *cmtcfg.Config, chainID string) (string, bool, error) {
	if !homeOnCommandLine {
		return "", false, errors.New("in-place-testnet requires an explicit --home naming the disposable copy to convert")
	}
	if config == nil {
		return "", false, errors.New("CometBFT config is required")
	}
	if laddr := strings.TrimSpace(config.PrivValidatorListenAddr); laddr != "" {
		return "", false, fmt.Errorf("refusing to convert a home that uses a remote signer (priv_validator_laddr = %q); convert a disposable copy with priv_validator_laddr cleared and a fresh local key", laddr)
	}
	if err := checkTestnetPathsInHome(config); err != nil {
		return "", false, err
	}
	sourceChainID, err := readGenesisChainID(config.GenesisFile())
	if err != nil {
		return "", false, err
	}
	backup := config.PrivValidatorKeyFile() + testnetValidatorBackupSuffix
	converted := false
	if _, err := os.Lstat(backup); err == nil {
		converted = true
	} else if !os.IsNotExist(err) {
		return "", false, fmt.Errorf("check validator backup %s: %w", backup, err)
	}
	if converted && sourceChainID == chainID {
		if err := app.ValidateTestnetChainID(chainID, ""); err != nil {
			return "", false, err
		}
		return sourceChainID, true, nil
	}
	if err := app.ValidateTestnetChainID(chainID, sourceChainID); err != nil {
		if converted {
			return "", false, fmt.Errorf("%w (this home was already converted to %s; to finish an interrupted conversion, rerun with that chain ID)", err, sourceChainID)
		}
		return "", false, err
	}
	return sourceChainID, false, nil
}

// checkTestnetPathsInHome refuses a configuration that points any file the
// conversion moves or rewrites outside the home. A config.toml copied from a
// validator may name absolute paths, for example a key on a secrets mount;
// converting the copy would then move aside and replace the live validator's
// key and signing state, or rewrite its consensus state. Paths are compared
// after resolving symlinks, because a copy can also carry a symlink (data/ or
// a key file linked to the source node's) that the conversion would follow.
// Both the entry itself (its directory resolved) and what it points to must be
// inside the home: the conversion renames and removes some entries, which acts
// on the link, not its target, so an outside link into the home is refused too.
func checkTestnetPathsInHome(config *cmtcfg.Config) error {
	root, err := resolveTestnetPath(config.RootDir)
	if err != nil {
		return fmt.Errorf("resolve home %s: %w", config.RootDir, err)
	}
	for _, setting := range []struct{ name, path string }{
		{"genesis_file", config.GenesisFile()},
		{"priv_validator_key_file", config.PrivValidatorKeyFile()},
		{"priv_validator_state_file", config.PrivValidatorStateFile()},
		{"db_dir", config.DBDir()},
		{"consensus.wal_file", config.Consensus.WalFile()},
		{"p2p.addr_book_file", config.P2P.AddrBookFile()},
	} {
		abs, err := filepath.Abs(setting.path)
		if err != nil {
			return fmt.Errorf("resolve %s %s: %w", setting.name, setting.path, err)
		}
		dir, err := resolveTestnetPath(filepath.Dir(abs))
		if err != nil {
			return fmt.Errorf("resolve %s %s: %w", setting.name, setting.path, err)
		}
		target, err := resolveTestnetPath(abs)
		if err != nil {
			return fmt.Errorf("resolve %s %s: %w", setting.name, setting.path, err)
		}
		for _, path := range []string{filepath.Join(dir, filepath.Base(abs)), target} {
			rel, err := filepath.Rel(root, path)
			if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				return fmt.Errorf("refusing to convert: %s (%s) is outside the home %s; point it inside the disposable copy", setting.name, path, root)
			}
		}
	}
	return nil
}

// resolveTestnetPath returns the absolute path with every symlink in its
// existing part resolved. The part that does not exist yet (a file the
// conversion or node will create) is appended unchanged.
func resolveTestnetPath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	existing, missing := abs, ""
	for {
		resolved, err := filepath.EvalSymlinks(existing)
		if err == nil {
			return filepath.Join(resolved, missing), nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		// A dangling symlink: creating the file would follow it to a target
		// that cannot be checked here.
		if info, lerr := os.Lstat(existing); lerr == nil && info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("%s is a symlink to a missing target", existing)
		}
		parent := filepath.Dir(existing)
		if parent == existing {
			return abs, nil
		}
		missing = filepath.Join(filepath.Base(existing), missing)
		existing = parent
	}
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
		if err := skipJSONValue(decoder); err != nil {
			return "", fmt.Errorf("read source genesis %s: %w", path, err)
		}
	}
	return "", fmt.Errorf("source genesis %s has no chain_id", path)
}

// skipJSONValue consumes the next value token by token, so a large app_state
// ahead of chain_id is never held in memory as a whole.
func skipJSONValue(decoder *json.Decoder) error {
	depth := 0
	for {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		switch token {
		case json.Delim('{'), json.Delim('['):
			depth++
		case json.Delim('}'), json.Delim(']'):
			depth--
		}
		if depth == 0 {
			return nil
		}
	}
}
