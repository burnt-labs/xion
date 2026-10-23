package main

import (
	"context"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	tmcfg "github.com/cometbft/cometbft/config"

	"github.com/cosmos/cosmos-sdk/server"

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
		register bool
		args     []string
		expected time.Duration
	}{
		"ignores commands without the start flag": {
			expected: tmcfg.DefaultConsensusConfig().TimeoutCommit,
		},
		"defaults to one second": {
			register: true,
			expected: time.Second,
		},
		"accepts a CLI override": {
			register: true,
			args:     []string{"--consensus.timeout_commit=2500ms"},
			expected: 2500 * time.Millisecond,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			serverCtx := server.NewDefaultContext()
			serverCtx.Config = tmcfg.DefaultConfig()
			cmd := &cobra.Command{Use: "test"}
			cmd.SetContext(context.WithValue(context.Background(), server.ServerContextKey, serverCtx))
			if tc.register {
				addModuleInitFlags(cmd)
			}
			require.NoError(t, cmd.ParseFlags(tc.args))
			require.NoError(t, applyValidatorTimeout(cmd))
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
