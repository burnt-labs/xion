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
	// configured stands in for a timeout_commit persisted in config.toml; it
	// must survive unless the flag is explicitly passed.
	configured := 5 * time.Second

	tests := map[string]struct {
		register bool
		args     []string
		expected time.Duration
	}{
		"ignores commands without the start flag": {
			expected: configured,
		},
		"preserves the configured timeout when the flag is not passed": {
			register: true,
			expected: configured,
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
			serverCtx.Config.Consensus.TimeoutCommit = configured
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
