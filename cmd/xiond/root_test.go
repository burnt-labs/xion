package main

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	tmcfg "github.com/cometbft/cometbft/config"

	"github.com/cosmos/cosmos-sdk/server"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"

	"github.com/burnt-labs/xion/app"
)

func TestSetValidatorTimeout(t *testing.T) {
	serverCtx := server.NewDefaultContext()
	serverCtx.Config = tmcfg.DefaultConfig()

	setValidatorTimeout(serverCtx, 2500*time.Millisecond)

	require.Equal(t, 2500*time.Millisecond, serverCtx.Config.Consensus.TimeoutCommit)
}

func TestApplyValidatorTimeout(t *testing.T) {
	tests := map[string]struct {
		register      bool
		args          []string
		onCommandLine bool
		expected      time.Duration
	}{
		"ignores commands without the start flag": {
			expected: 5 * time.Second,
		},
		"replaces the loaded value with the default": {
			register: true,
			expected: time.Second,
		},
		"accepts a CLI override": {
			register:      true,
			args:          []string{"--consensus.timeout_commit=2500ms"},
			onCommandLine: true,
			expected:      2500 * time.Millisecond,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			serverCtx := server.NewDefaultContext()
			serverCtx.Config = tmcfg.DefaultConfig()
			serverCtx.Config.Consensus.TimeoutCommit = 5 * time.Second
			cmd := &cobra.Command{Use: "test"}
			cmd.SetContext(context.WithValue(context.Background(), server.ServerContextKey, serverCtx))
			if tc.register {
				addModuleInitFlags(cmd)
			}
			require.NoError(t, cmd.ParseFlags(tc.args))
			require.NoError(t, applyValidatorTimeout(cmd, tc.onCommandLine))
			require.Equal(t, tc.expected, serverCtx.Config.Consensus.TimeoutCommit)
		})
	}
}

func TestStartAcceptsTraceStore(t *testing.T) {
	// Indexer nodes run `xiond start --trace-store <fifo>`; SDK v0.55 dropped
	// the flag, so xiond registers it itself.
	cmd := &cobra.Command{Use: "start"}
	addModuleInitFlags(cmd)
	require.NoError(t, cmd.ParseFlags([]string{"--trace-store", "/tmp/trace.pipe"}))

	got, err := cmd.Flags().GetString(app.FlagTraceStore)
	require.NoError(t, err)
	require.Equal(t, "/tmp/trace.pipe", got)
}

// TestEffectiveTimeoutCommit runs the server pre-run the way xiond does
// (svrcmd.Execute, the SDK start command, config.toml on disk) and checks the
// timeout_commit the node would start with.
func TestEffectiveTimeoutCommit(t *testing.T) {
	executable, err := os.Executable()
	require.NoError(t, err)
	basename := path.Base(executable)
	envReplacer := strings.NewReplacer(".", "_", "-", "_")
	// The two names the SDK binds: the automatic <EXECUTABLE>_ prefix (for a
	// binary named xiond, XIOND_CONSENSUS_TIMEOUT_COMMIT) and the explicit
	// <executable>_<FLAG> binding.
	automaticEnv := envReplacer.Replace(strings.ToUpper(basename + "_" + timeoutCommitFlag))
	boundEnv := envReplacer.Replace(basename + "_" + strings.ToUpper(timeoutCommitFlag))

	tests := map[string]struct {
		configToml string
		args       []string
		env        map[string]string
		expected   time.Duration
	}{
		"config.toml 5s is replaced by the 1s default": {
			configToml: "5s",
			expected:   time.Second,
		},
		"config.toml 1s stays 1s": {
			configToml: "1s",
			expected:   time.Second,
		},
		"config.toml 300ms is replaced by the 1s default": {
			configToml: "300ms",
			expected:   time.Second,
		},
		"command line overrides config.toml": {
			configToml: "5s",
			args:       []string{"--consensus.timeout_commit=2500ms"},
			expected:   2500 * time.Millisecond,
		},
		"command line overrides the default": {
			configToml: "1s",
			args:       []string{"--consensus.timeout_commit", "3s"},
			expected:   3 * time.Second,
		},
		"automatic env overrides the default": {
			configToml: "1s",
			env:        map[string]string{automaticEnv: "7s"},
			expected:   7 * time.Second,
		},
		"bound env overrides the default": {
			configToml: "5s",
			env:        map[string]string{boundEnv: "4s"},
			expected:   4 * time.Second,
		},
		"command line overrides env": {
			configToml: "5s",
			args:       []string{"--consensus.timeout_commit=3s"},
			env:        map[string]string{automaticEnv: "7s"},
			expected:   3 * time.Second,
		},
		"empty env is ignored": {
			configToml: "5s",
			env:        map[string]string{automaticEnv: ""},
			expected:   time.Second,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			home := writeConsensusConfig(t, tc.configToml)

			got, err := runStartPreRun(home, tc.args)
			require.NoError(t, err)
			require.Equal(t, tc.expected, got)
		})
	}
}

// TestEffectiveTimeoutCommitOtherCommands checks that commands without the
// start flag keep config.toml's value.
func TestEffectiveTimeoutCommitOtherCommands(t *testing.T) {
	home := writeConsensusConfig(t, "5s")

	var got time.Duration
	other := &cobra.Command{
		Use: "other",
		RunE: func(cmd *cobra.Command, _ []string) error {
			got = server.GetServerContextFromCmd(cmd).Config.Consensus.TimeoutCommit
			return nil
		},
	}
	root := &cobra.Command{
		Use:               "xiond",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error { return interceptServerConfigs(cmd) },
	}
	root.AddCommand(other)
	root.SetArgs([]string{"other", "--home", home})
	root.SetOut(&strings.Builder{})

	require.NoError(t, svrcmd.Execute(root, "", home))
	require.Equal(t, 5*time.Second, got)
}

func writeConsensusConfig(t *testing.T, timeoutCommit string) string {
	t.Helper()
	home := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(home, "config"), 0o755))

	d, err := time.ParseDuration(timeoutCommit)
	require.NoError(t, err)
	cfg := tmcfg.DefaultConfig()
	cfg.SetRoot(home)
	cfg.Consensus.TimeoutCommit = d
	tmcfg.WriteConfigFile(filepath.Join(home, "config", "config.toml"), cfg)
	return home
}

// runStartPreRun executes `start` with the SDK start command and xiond's
// server pre-run, and returns the timeout_commit instead of starting the node.
func runStartPreRun(home string, args []string) (time.Duration, error) {
	var got time.Duration
	start := server.StartCmd(nil, home)
	addModuleInitFlags(start)
	start.RunE = func(cmd *cobra.Command, _ []string) error {
		got = server.GetServerContextFromCmd(cmd).Config.Consensus.TimeoutCommit
		return nil
	}

	root := &cobra.Command{
		Use:               "xiond",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error { return interceptServerConfigs(cmd) },
	}
	root.AddCommand(start)
	root.SetArgs(append([]string{"start", "--home", home}, args...))
	root.SetOut(&strings.Builder{})

	err := svrcmd.Execute(root, "", home)
	return got, err
}
