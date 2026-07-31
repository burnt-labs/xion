package main

import (
	"context"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	tmcfg "github.com/cometbft/cometbft/config"

	"github.com/cosmos/cosmos-sdk/server"
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
			cmd.SetContext(context.Background())
			require.NoError(t, server.SetCmdServerContext(cmd, serverCtx))
			if tc.register {
				addModuleInitFlags(cmd)
			}
			require.NoError(t, cmd.Flags().Parse(tc.args))
			require.NoError(t, applyValidatorTimeout(cmd))
			require.Equal(t, tc.expected, serverCtx.Config.Consensus.TimeoutCommit)
		})
	}
}
