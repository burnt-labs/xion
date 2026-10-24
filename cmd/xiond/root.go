package main

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/CosmWasm/wasmd/x/wasm"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	tmcfg "github.com/cometbft/cometbft/config"

	dbm "github.com/cosmos/cosmos-db"

	"cosmossdk.io/log/v2"
	confixcmd "cosmossdk.io/tools/confix/cmd"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/config"
	"github.com/cosmos/cosmos-sdk/client/debug"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/keys"
	"github.com/cosmos/cosmos-sdk/client/pruning"
	"github.com/cosmos/cosmos-sdk/client/rpc"
	"github.com/cosmos/cosmos-sdk/client/snapshot"
	"github.com/cosmos/cosmos-sdk/server"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/version"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	genutilcli "github.com/cosmos/cosmos-sdk/x/genutil/client/cli"

	"github.com/burnt-labs/xion/app"
	"github.com/burnt-labs/xion/app/params"
	indexercli "github.com/burnt-labs/xion/indexer/client/cli"
)

// NewRootCmd creates a new root command for xiond. It is called once in the
// main function.
func NewRootCmd() (*cobra.Command, params.EncodingConfig) {
	cfg := sdk.GetConfig()
	cfg.SetBech32PrefixForAccount(app.Bech32PrefixAccAddr, app.Bech32PrefixAccPub)
	cfg.SetBech32PrefixForValidator(app.Bech32PrefixValAddr, app.Bech32PrefixValPub)
	cfg.SetBech32PrefixForConsensusNode(app.Bech32PrefixConsAddr, app.Bech32PrefixConsPub)
	cfg.SetAddressVerifier(wasmtypes.VerifyAddressLen())
	cfg.Seal()

	tempApp := app.NewWasmApp(log.NewNopLogger(), dbm.NewMemDB(), false, simtestutil.NewAppOptionsWithFlagHome(tempDir()), []wasmkeeper.Option{})
	encodingConfig := params.EncodingConfig{
		InterfaceRegistry: tempApp.InterfaceRegistry(),
		Codec:             tempApp.AppCodec(),
		TxConfig:          tempApp.TxConfig(),
		Amino:             tempApp.LegacyAmino(),
	}

	initClientCtx := client.Context{}.
		WithCodec(encodingConfig.Codec).
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry).
		WithTxConfig(encodingConfig.TxConfig).
		WithLegacyAmino(encodingConfig.Amino).
		WithInput(os.Stdin).
		WithAccountRetriever(authtypes.AccountRetriever{}).
		WithHomeDir(app.DefaultNodeHome).
		WithViper("") // In xiond, we don't use any prefix for env variables.

	rootCmd := &cobra.Command{
		Use:   version.AppName,
		Short: "xion daemon (server)",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// set the default command outputs
			cmd.SetOut(cmd.OutOrStdout())
			cmd.SetErr(cmd.ErrOrStderr())

			initClientCtx = initClientCtx.WithCmdContext(cmd.Context())
			initClientCtx, err := client.ReadPersistentCommandFlags(initClientCtx, cmd.Flags())
			if err != nil {
				return err
			}

			initClientCtx, err = config.ReadFromClientConfig(initClientCtx)
			if err != nil {
				return err
			}

			if err := client.SetCmdClientContextHandler(initClientCtx, cmd); err != nil {
				return err
			}

			return interceptServerConfigs(cmd)
		},
	}

	initRootCmd(rootCmd, encodingConfig, tempApp.BasicModuleManager)

	// add keyring to autocli opts
	autoCliOpts := tempApp.AutoCliOpts()
	initClientCtx, _ = config.ReadFromClientConfig(initClientCtx)
	autoCliOpts.ClientCtx = initClientCtx

	if err := autoCliOpts.EnhanceRootCommand(rootCmd); err != nil {
		panic(err)
	}

	return rootCmd, encodingConfig
}

// initTendermintConfig helps to override default Tendermint Config values.
// return tmcfg.DefaultConfig if no custom configuration is required for the application.
func initTendermintConfig() *tmcfg.Config {
	return tmcfg.DefaultConfig()
}

// timeoutCommitFlag is the start-only flag that sets CometBFT's
// consensus.timeout_commit.
const timeoutCommitFlag = "consensus.timeout_commit"

// interceptServerConfigs loads config.toml and app.toml into the server
// context, then applies the validator timeout. It is the server half of the
// root pre-run.
func interceptServerConfigs(cmd *cobra.Command) error {
	// Read this before the SDK loads the configuration: it copies config.toml
	// and environment values into every flag the command line left unset and
	// so marks them as changed.
	timeoutOnCommandLine := cmd.Flags().Changed(timeoutCommitFlag)

	customAppTemplate, customAppConfig := initAppConfig()
	customTMConfig := initTendermintConfig()

	if err := server.InterceptConfigsPreRunHandler(cmd, customAppTemplate, customAppConfig, customTMConfig); err != nil {
		return err
	}

	return applyValidatorTimeout(cmd, timeoutOnCommandLine)
}

// setValidatorTimeout applies the timeout after config.toml is loaded.
func setValidatorTimeout(serverCtx *server.Context, timeoutCommit time.Duration) {
	serverCtx.Config.Consensus.TimeoutCommit = timeoutCommit
}

// applyValidatorTimeout sets consensus.timeout_commit for commands that
// register the start-only flag. Other commands also run the root pre-run, but
// do not register the flag and are left alone.
//
// The chain targets one-second blocks, so the flag's one-second default
// replaces whatever config.toml holds: the SDK writes 5s into every new
// config.toml, which is how validators drift to 5s. Only an explicit operator
// setting may change it, in this order:
//  1. --consensus.timeout_commit on the command line;
//  2. the environment variable the SDK binds to the flag
//     (XIOND_CONSENSUS_TIMEOUT_COMMIT for a binary named xiond);
//  3. otherwise the flag's default; config.toml is ignored.
//
// timeoutOnCommandLine must be read before the SDK loads the configuration.
func applyValidatorTimeout(cmd *cobra.Command, timeoutOnCommandLine bool) error {
	flag := cmd.Flags().Lookup(timeoutCommitFlag)
	if flag == nil {
		return nil
	}

	serverCtx := server.GetServerContextFromCmd(cmd)

	if timeoutOnCommandLine || timeoutCommitInEnv() {
		// The SDK has already resolved the command line over the environment
		// into the flag.
		timeoutCommit, err := cmd.Flags().GetDuration(timeoutCommitFlag)
		if err != nil {
			return err
		}
		setValidatorTimeout(serverCtx, timeoutCommit)
		return nil
	}

	timeoutCommit, err := time.ParseDuration(flag.DefValue)
	if err != nil {
		return fmt.Errorf("parse %s default %q: %w", timeoutCommitFlag, flag.DefValue, err)
	}
	if fromConfig := serverCtx.Config.Consensus.TimeoutCommit; fromConfig != timeoutCommit && serverCtx.Logger != nil {
		serverCtx.Logger.Info("ignoring consensus.timeout_commit from config.toml; set --consensus.timeout_commit to change it",
			"config_toml", fromConfig.String(), "timeout_commit", timeoutCommit.String())
	}
	setValidatorTimeout(serverCtx, timeoutCommit)
	return nil
}

// timeoutCommitInEnv reports whether the environment sets the timeout flag.
// It mirrors how the SDK binds the environment to flags in
// server.InterceptConfigsAndCreateContext (automatic <EXECUTABLE>_ prefix) and
// bindFlags (explicit <executable>_<FLAG> binding), without config.toml.
func timeoutCommitInEnv() bool {
	executable, err := os.Executable()
	if err != nil {
		return false
	}
	basename := path.Base(executable)

	v := viper.New()
	v.SetEnvPrefix(basename)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()
	if err := v.BindEnv(timeoutCommitFlag, fmt.Sprintf("%s_%s", basename, strings.ToUpper(strings.ReplaceAll(timeoutCommitFlag, "-", "_")))); err != nil {
		return false
	}
	return v.IsSet(timeoutCommitFlag)
}

// initAppConfig helps to override default appConfig template and configs.
// return "", nil if no custom configuration is required for the application.
func initAppConfig() (string, any) {
	return app.DefaultConfig()
}

func initRootCmd(rootCmd *cobra.Command,
	encodingConfig params.EncodingConfig,
	basicManager module.BasicManager,
) {
	rootCmd.AddCommand(
		genutilcli.InitCmd(basicManager, app.DefaultNodeHome),
		// testnetCmd(app.ModuleBasics, banktypes.GenesisBalancesIterator{}),
		debug.Cmd(),
		confixcmd.ConfigCommand(),
		pruning.Cmd(newApp, app.DefaultNodeHome),
		snapshot.Cmd(newApp),
	)

	server.AddCommands(rootCmd, app.DefaultNodeHome, newApp, appExport, addModuleInitFlags)

	// in-place-testnet: create and run a testnet from this node's state
	// (e.g. a restored snapshot) without a genesis export.
	rootCmd.AddCommand(inPlaceTestnetCmd())

	// add keybase, auxiliary RPC, query, and tx child commands
	rootCmd.AddCommand(
		server.StatusCommand(),
		genesisCommand(encodingConfig, basicManager),
		queryCommand(basicManager),
		txCommand(basicManager),
		keys.Commands(),
	)
	// add indexer
	rootCmd.AddCommand(indexercli.Indexer(newApp, app.DefaultNodeHome))
}

func addModuleInitFlags(startCmd *cobra.Command) {
	wasm.AddModuleInitFlags(startCmd)
	startCmd.Flags().Duration(timeoutCommitFlag, time.Second, "How long to wait after committing a block before starting the next height; overrides config.toml (env: XIOND_CONSENSUS_TIMEOUT_COMMIT)")
	startCmd.Flags().String(app.FlagTraceStore, "", "Write every committed store write and delete to this file (or FIFO) as JSON lines, for state indexers")
}

// genesisCommand builds genesis-related `simd genesis` command. Users may provide application specific commands as a parameter
func genesisCommand(encodingConfig params.EncodingConfig, basicManager module.BasicManager, cmds ...*cobra.Command) *cobra.Command {
	cmd := genutilcli.GenesisCoreCommand(encodingConfig.TxConfig, basicManager, app.DefaultNodeHome)

	for _, subCmd := range cmds {
		cmd.AddCommand(subCmd)
	}
	return cmd
}

func queryCommand(_ module.BasicManager) *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "query",
		Aliases:                    []string{"q"},
		Short:                      "Querying subcommands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		rpc.QueryEventForTxCmd(),
		server.QueryBlockCmd(),
		authcmd.QueryTxsByEventsCmd(),
		server.QueryBlocksCmd(),
		authcmd.QueryTxCmd(),
		server.QueryBlockResultsCmd(),
	)

	return cmd
}

func txCommand(manager module.BasicManager) *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "tx",
		Short:                      "Transactions subcommands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		authcmd.GetSignCommand(),
		authcmd.GetSignBatchCommand(),
		authcmd.GetMultiSignCommand(),
		authcmd.GetMultiSignBatchCmd(),
		authcmd.GetValidateSignaturesCommand(),
		authcmd.GetBroadcastCommand(),
		authcmd.GetEncodeCommand(),
		authcmd.GetDecodeCommand(),
		authcmd.GetSimulateCmd(),
	)

	manager.AddTxCommands(cmd)

	return cmd
}

// newApp creates the application
func newApp(
	logger log.Logger,
	db dbm.DB,
	appOpts servertypes.AppOptions,
) servertypes.Application {
	baseappOptions := server.DefaultBaseappOptions(appOpts)

	var wasmOpts []wasmkeeper.Option
	if cast.ToBool(appOpts.Get("telemetry.enabled")) {
		wasmOpts = append(wasmOpts, wasmkeeper.WithVMCacheMetrics(prometheus.DefaultRegisterer))
	}

	return app.NewWasmApp(
		logger, db, true,
		appOpts,
		wasmOpts,
		baseappOptions...,
	)
}

// appExport creates a new wasm app (optionally at a given height) and exports state.
func appExport(
	logger log.Logger,
	db dbm.DB,
	height int64,
	forZeroHeight bool,
	jailAllowedAddrs []string,
	appOpts servertypes.AppOptions,
	modulesToExport []string,
) (servertypes.ExportedApp, error) {
	var wasmApp *app.WasmApp
	homePath, ok := appOpts.Get(flags.FlagHome).(string)
	if !ok || homePath == "" {
		return servertypes.ExportedApp{}, errors.New("application home is not set")
	}

	viperAppOpts, ok := appOpts.(*viper.Viper)
	if !ok {
		return servertypes.ExportedApp{}, errors.New("appOpts is not viper.Viper")
	}

	// overwrite the FlagInvCheckPeriod
	viperAppOpts.Set(server.FlagInvCheckPeriod, 1)
	appOpts = viperAppOpts

	var emptyWasmOpts []wasmkeeper.Option
	wasmApp = app.NewWasmApp(
		logger,
		db,
		height == -1,
		appOpts,
		emptyWasmOpts,
	)

	if height != -1 {
		if err := wasmApp.LoadHeight(height); err != nil {
			return servertypes.ExportedApp{}, err
		}
	}

	return wasmApp.ExportAppStateAndValidators(forZeroHeight, jailAllowedAddrs, modulesToExport)
}

var tempDir = func() string {
	dir, err := os.MkdirTemp("", "wasmd")
	if err != nil {
		panic("failed to create temp dir: " + err.Error())
	}
	defer os.RemoveAll(dir)

	return dir
}
